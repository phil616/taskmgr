package router

import (
	"ops-timer-backend/internal/api/handler"
	"ops-timer-backend/internal/api/middleware"
	"ops-timer-backend/internal/pkg/auth"
	"ops-timer-backend/internal/pkg/response"
	"ops-timer-backend/internal/service"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type Router struct {
	engine          *gin.Engine
	authHandler     *handler.AuthHandler
	unitHandler     *handler.UnitHandler
	projectHandler  *handler.ProjectHandler
	todoHandler     *handler.TodoHandler
	notifHandler    *handler.NotificationHandler
	scheduleHandler *handler.ScheduleHandler
	budgetHandler   *handler.BudgetHandler
	mcpHandler      *handler.MCPHandler
	jwtManager      *auth.JWTManager
	authService     *service.AuthService
	logger          *zap.Logger
	corsOrigins     []string
	mcpPath         string
}

type RouterConfig struct {
	AuthHandler     *handler.AuthHandler
	UnitHandler     *handler.UnitHandler
	ProjectHandler  *handler.ProjectHandler
	TodoHandler     *handler.TodoHandler
	NotifHandler    *handler.NotificationHandler
	ScheduleHandler *handler.ScheduleHandler
	BudgetHandler   *handler.BudgetHandler
	MCPHandler      *handler.MCPHandler
	JWTManager      *auth.JWTManager
	AuthService     *service.AuthService
	Logger          *zap.Logger
	CorsOrigins     []string
	MCPPath         string
}

func NewRouter(cfg *RouterConfig) *Router {
	mcpPath := cfg.MCPPath
	if mcpPath == "" {
		mcpPath = "/mcp"
	}

	return &Router{
		authHandler:     cfg.AuthHandler,
		unitHandler:     cfg.UnitHandler,
		projectHandler:  cfg.ProjectHandler,
		todoHandler:     cfg.TodoHandler,
		notifHandler:    cfg.NotifHandler,
		scheduleHandler: cfg.ScheduleHandler,
		budgetHandler:   cfg.BudgetHandler,
		mcpHandler:      cfg.MCPHandler,
		jwtManager:      cfg.JWTManager,
		authService:     cfg.AuthService,
		logger:          cfg.Logger,
		corsOrigins:     cfg.CorsOrigins,
		mcpPath:         mcpPath,
	}
}

func (r *Router) Setup() *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	engine := gin.New()
	engine.Use(gin.Recovery())
	engine.Use(middleware.SecurityHeadersMiddleware())
	engine.Use(middleware.BodySizeLimitMiddleware(4 << 20)) // 4 MiB 请求体上限
	engine.Use(middleware.CORSMiddleware(r.corsOrigins))
	engine.Use(middleware.LoggerMiddleware(r.logger))

	engine.GET("/health", func(c *gin.Context) {
		response.Success(c, gin.H{"status": "ok"})
	})
	if r.mcpHandler != nil {
		engine.POST(r.mcpPath, r.mcpHandler.Handle)
		engine.GET(r.mcpPath+"/config", r.mcpHandler.GetConfig)
	}

	api := engine.Group("/api/v1")

	// Public routes
	api.POST("/auth/login", r.authHandler.Login)

	// Protected routes
	protected := api.Group("")
	protected.Use(middleware.AuthMiddleware(r.jwtManager, r.authService))

	// Auth
	protected.POST("/auth/logout", r.authHandler.Logout)
	protected.GET("/auth/profile", r.authHandler.GetProfile)
	protected.PUT("/auth/profile", r.authHandler.UpdateProfile)
	protected.PUT("/auth/password", r.authHandler.ChangePassword)
	protected.GET("/auth/token", r.authHandler.GetToken)
	protected.POST("/auth/token/regenerate", r.authHandler.RegenerateToken)
	protected.POST("/auth/test-email", r.authHandler.TestEmail)
	protected.GET("/auth/smtp-status", r.authHandler.SMTPStatus)

	// Units
	protected.GET("/units", r.unitHandler.List)
	protected.POST("/units", r.unitHandler.Create)
	protected.GET("/units/summary", r.unitHandler.Summary)
	protected.GET("/units/:id", r.unitHandler.Get)
	protected.PUT("/units/:id", r.unitHandler.Update)
	protected.PATCH("/units/:id", r.unitHandler.Update)
	protected.DELETE("/units/:id", r.unitHandler.Delete)
	protected.PATCH("/units/:id/status", r.unitHandler.UpdateStatus)
	protected.POST("/units/:id/step", r.unitHandler.Step)
	protected.PUT("/units/:id/value", r.unitHandler.SetValue)
	protected.GET("/units/:id/logs", r.unitHandler.GetLogs)

	// Projects
	protected.GET("/projects", r.projectHandler.List)
	protected.POST("/projects", r.projectHandler.Create)
	protected.GET("/projects/:id", r.projectHandler.Get)
	protected.PUT("/projects/:id", r.projectHandler.Update)
	protected.PATCH("/projects/:id", r.projectHandler.Update)
	protected.DELETE("/projects/:id", r.projectHandler.Delete)
	protected.GET("/projects/:id/units", r.projectHandler.GetUnits)
	protected.GET("/projects/:id/budget", r.projectHandler.GetBudgetStats)

	// Todos
	protected.GET("/todos", r.todoHandler.List)
	protected.POST("/todos", r.todoHandler.Create)
	protected.GET("/todos/:id", r.todoHandler.Get)
	protected.PUT("/todos/:id", r.todoHandler.Update)
	protected.PATCH("/todos/:id", r.todoHandler.Update)
	protected.DELETE("/todos/:id", r.todoHandler.Delete)
	protected.PATCH("/todos/:id/status", r.todoHandler.UpdateStatus)
	protected.POST("/todos/batch", r.todoHandler.BatchAction)

	// Todo Groups
	protected.GET("/todo-groups", r.todoHandler.ListGroups)
	protected.POST("/todo-groups", r.todoHandler.CreateGroup)
	protected.PUT("/todo-groups/:id", r.todoHandler.UpdateGroup)
	protected.DELETE("/todo-groups/:id", r.todoHandler.DeleteGroup)

	// Notifications
	protected.GET("/notifications", r.notifHandler.List)
	protected.PATCH("/notifications/:id/read", r.notifHandler.MarkAsRead)
	protected.POST("/notifications/read-all", r.notifHandler.MarkAllAsRead)
	protected.GET("/notifications/unread-count", r.notifHandler.UnreadCount)
	protected.DELETE("/notifications/:id", r.notifHandler.Delete)

	// Schedules
	protected.GET("/schedules", r.scheduleHandler.List)
	protected.POST("/schedules", r.scheduleHandler.Create)
	protected.GET("/schedules/:id", r.scheduleHandler.Get)
	protected.PUT("/schedules/:id", r.scheduleHandler.Update)
	protected.PATCH("/schedules/:id", r.scheduleHandler.Update)
	protected.DELETE("/schedules/:id", r.scheduleHandler.Delete)
	protected.POST("/schedules/:id/resources", r.scheduleHandler.AddResource)
	protected.DELETE("/schedules/:id/resources/:resource_id", r.scheduleHandler.RemoveResource)

	// Budget - Wallets
	protected.GET("/wallets", r.budgetHandler.ListWallets)
	protected.POST("/wallets", r.budgetHandler.CreateWallet)
	protected.GET("/wallets/:id", r.budgetHandler.GetWallet)
	protected.PUT("/wallets/:id", r.budgetHandler.UpdateWallet)
	protected.DELETE("/wallets/:id", r.budgetHandler.DeleteWallet)

	// Budget - Categories
	protected.GET("/budget/categories", r.budgetHandler.ListCategories)
	protected.POST("/budget/categories", r.budgetHandler.CreateCategory)
	protected.PUT("/budget/categories/:id", r.budgetHandler.UpdateCategory)
	protected.DELETE("/budget/categories/:id", r.budgetHandler.DeleteCategory)

	// Budget - Transactions
	protected.GET("/transactions", r.budgetHandler.ListTransactions)
	protected.POST("/transactions", r.budgetHandler.CreateTransaction)
	protected.GET("/transactions/:id", r.budgetHandler.GetTransaction)
	protected.PUT("/transactions/:id", r.budgetHandler.UpdateTransaction)
	protected.DELETE("/transactions/:id", r.budgetHandler.DeleteTransaction)

	// Budget - Stats
	protected.GET("/budget/stats", r.budgetHandler.GetStats)

	r.engine = engine
	return engine
}
