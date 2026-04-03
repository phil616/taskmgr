package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"ops-timer-backend/internal/api/handler"
	"ops-timer-backend/internal/api/router"
	"ops-timer-backend/internal/config"
	"ops-timer-backend/internal/model"
	"ops-timer-backend/internal/pkg/auth"
	"ops-timer-backend/internal/pkg/email"
	"ops-timer-backend/internal/pkg/scheduler"
	"ops-timer-backend/internal/repository"
	"ops-timer-backend/internal/service"

	"github.com/glebarez/sqlite"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func main() {
	initAdmin := flag.Bool("init-admin", false, "初始化/重置管理员账户")
	adminUser := flag.String("admin-user", "admin", "管理员用户名")
	adminPassEnv := os.Getenv("TASK_MANAGER_INIT_ADMIN_PASSWORD")
	adminPass := flag.String("admin-pass", adminPassEnv, "初始管理员密码（建议通过 TASK_MANAGER_INIT_ADMIN_PASSWORD 环境变量传入）")
	flag.Parse()

	if *adminPass == "" {
		*adminPass = "Admin@12345"
	}

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	zapLogger := initLogger(cfg.Log)
	defer zapLogger.Sync()

	// 安全检查：JWT Secret 必须足够复杂
	validateJWTSecret(cfg.Auth.JWTSecret, zapLogger)

	db := initDatabase(cfg.Database)

	autoMigrate(db)

	// Repositories
	userRepo := repository.NewUserRepository(db)
	projectRepo := repository.NewProjectRepository(db)
	unitRepo := repository.NewUnitRepository(db)
	unitLogRepo := repository.NewUnitLogRepository(db)
	todoRepo := repository.NewTodoRepository(db)
	todoGroupRepo := repository.NewTodoGroupRepository(db)
	notifRepo := repository.NewNotificationRepository(db)
	scheduleRepo := repository.NewScheduleRepository(db)
	walletRepo := repository.NewWalletRepository(db)
	categoryRepo := repository.NewBudgetCategoryRepository(db)
	txRepo := repository.NewTransactionRepository(db)

	// Auth
	jwtManager := auth.NewJWTManager(cfg.Auth.JWTSecret, cfg.Auth.JWTExpiryHours)

	// Email
	emailSvc := email.NewService(&cfg.SMTP)
	if emailSvc.Enabled() {
		zapLogger.Info("SMTP 邮件通知已启用", zap.String("host", cfg.SMTP.Host))
	} else {
		zapLogger.Info("SMTP 邮件通知未配置，跳过邮件功能")
	}

	// Services
	authService := service.NewAuthService(userRepo, jwtManager, &cfg.Auth)
	unitService := service.NewUnitService(unitRepo, unitLogRepo)
	projectService := service.NewProjectService(projectRepo, unitRepo)
	todoService := service.NewTodoService(todoRepo, todoGroupRepo)
	notifService := service.NewNotificationService(notifRepo)
	scheduleService := service.NewScheduleService(scheduleRepo, projectRepo, unitRepo, todoRepo)
	budgetService := service.NewBudgetService(walletRepo, categoryRepo, txRepo)

	if err := authService.EnsureAdminExists(*adminUser, *adminPass); err != nil {
		zapLogger.Fatal("创建管理员账户失败", zap.Error(err))
	}

	if err := budgetService.InitDefaultCategories(); err != nil {
		zapLogger.Warn("初始化默认预算分类失败", zap.Error(err))
	}

	if *initAdmin {
		zapLogger.Info("管理员账户已初始化", zap.String("username", *adminUser))
		os.Exit(0)
	}

	// Handlers
	authHandler := handler.NewAuthHandler(authService, emailSvc)
	unitHandler := handler.NewUnitHandler(unitService)
	projectHandler := handler.NewProjectHandler(projectService, unitService)
	todoHandler := handler.NewTodoHandler(todoService)
	notifHandler := handler.NewNotificationHandler(notifService)
	scheduleHandler := handler.NewScheduleHandler(scheduleService)
	budgetHandler := handler.NewBudgetHandler(budgetService)
	var mcpHandler *handler.MCPHandler
	if cfg.MCP.Enabled {
		mcpHandler = handler.NewMCPHandler(handler.MCPHandlerConfig{
			AuthService:    authService,
			BaseURL:        fmt.Sprintf("http://127.0.0.1:%d", cfg.Server.Port),
			ExternalURL:    cfg.MCP.ExternalURL,
			MCPPath:        cfg.MCP.Path,
			ServerName:     cfg.MCP.ServerName,
			ServerVersion:  cfg.MCP.ServerVersion,
			TimeoutSeconds: cfg.MCP.TimeoutSeconds,
		})
		zapLogger.Info("MCP 服务已启用", zap.String("path", cfg.MCP.Path))
	} else {
		zapLogger.Info("MCP 服务未启用")
	}

	// Router
	r := router.NewRouter(&router.RouterConfig{
		AuthHandler:     authHandler,
		UnitHandler:     unitHandler,
		ProjectHandler:  projectHandler,
		TodoHandler:     todoHandler,
		NotifHandler:    notifHandler,
		ScheduleHandler: scheduleHandler,
		BudgetHandler:   budgetHandler,
		MCPHandler:      mcpHandler,
		JWTManager:      jwtManager,
		AuthService:     authService,
		Logger:          zapLogger,
		CorsOrigins:     cfg.Server.CorsOrigins,
		MCPPath:         cfg.MCP.Path,
	})

	engine := r.Setup()

	// Scheduler
	sched := scheduler.NewScheduler(unitRepo, notifRepo, userRepo, emailSvc, zapLogger)
	if err := sched.Start(cfg.Scheduler.NotificationScanInterval); err != nil {
		zapLogger.Error("启动定时任务失败", zap.Error(err))
	}
	defer sched.Stop()

	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	zapLogger.Info("任务管理器服务已启动", zap.String("addr", addr))

	if err := engine.Run(addr); err != nil {
		zapLogger.Fatal("服务启动失败", zap.Error(err))
	}
}

func initLogger(cfg config.LogConfig) *zap.Logger {
	var zapCfg zap.Config
	if cfg.Format == "json" {
		zapCfg = zap.NewProductionConfig()
	} else {
		zapCfg = zap.NewDevelopmentConfig()
	}

	switch cfg.Level {
	case "debug":
		zapCfg.Level.SetLevel(zap.DebugLevel)
	case "warn":
		zapCfg.Level.SetLevel(zap.WarnLevel)
	case "error":
		zapCfg.Level.SetLevel(zap.ErrorLevel)
	default:
		zapCfg.Level.SetLevel(zap.InfoLevel)
	}

	l, err := zapCfg.Build()
	if err != nil {
		log.Fatalf("初始化日志失败: %v", err)
	}
	return l
}

func initDatabase(cfg config.DatabaseConfig) *gorm.DB {
	dir := filepath.Dir(cfg.DSN)
	if dir != "." {
		os.MkdirAll(dir, 0755)
	}

	db, err := gorm.Open(sqlite.Open(cfg.DSN), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		log.Fatalf("连接数据库失败: %v", err)
	}
	return db
}

func autoMigrate(db *gorm.DB) {
	err := db.AutoMigrate(
		&model.User{},
		&model.Project{},
		&model.Unit{},
		&model.UnitLog{},
		&model.TodoGroup{},
		&model.Todo{},
		&model.Notification{},
		&model.Schedule{},
		&model.ScheduleResource{},
		&model.Wallet{},
		&model.BudgetCategory{},
		&model.Transaction{},
	)
	if err != nil {
		log.Fatalf("数据库迁移失败: %v", err)
	}
}

// knownWeakSecrets 是常见弱密钥列表，启动时检测防止误用
var knownWeakSecrets = []string{
	"CHANGE_ME_TO_A_RANDOM_SECRET",
	"change_me_to_a_random_secret_in_production",
	"secret",
	"your_jwt_secret",
	"jwt_secret",
}

// validateJWTSecret 检查 JWT Secret 强度，不符合要求时 Fatal 退出
func validateJWTSecret(secret string, logger *zap.Logger) {
	if secret == "" {
		logger.Fatal("JWT Secret 未配置，请通过 TASK_MANAGER_AUTH_JWT_SECRET 环境变量或 .env 文件设置")
	}
	if len(secret) < 32 {
		logger.Fatal("JWT Secret 长度不足（当前不足 32 位），请生成强随机字符串",
			zap.Int("current_len", len(secret)))
	}
	for _, weak := range knownWeakSecrets {
		if secret == weak {
			logger.Fatal("检测到已知弱 JWT Secret，请立即替换为强随机字符串（可用: openssl rand -hex 32）",
				zap.String("hint", "TASK_MANAGER_AUTH_JWT_SECRET=<your-random-secret>"))
		}
	}
}
