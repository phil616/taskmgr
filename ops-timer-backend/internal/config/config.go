package config

import (
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	Server    ServerConfig
	Database  DatabaseConfig
	Auth      AuthConfig
	Scheduler SchedulerConfig
	MCP       MCPConfig
	SMTP      SMTPConfig
	Log       LogConfig
}

type ServerConfig struct {
	Host        string
	Port        int
	CorsOrigins []string
}

type DatabaseConfig struct {
	Driver string
	DSN    string
}

type AuthConfig struct {
	JWTSecret         string
	JWTExpiryHours    int
	LoginLockAttempts int
	LoginLockMinutes  int
}

type SchedulerConfig struct {
	NotificationScanInterval string
}

type MCPConfig struct {
	Enabled        bool
	Path           string
	ExternalURL    string
	ServerName     string
	ServerVersion  string
	TimeoutSeconds int
}

type SMTPConfig struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string
}

func (c *SMTPConfig) Enabled() bool {
	return c.Host != "" && c.From != ""
}

type LogConfig struct {
	Level  string
	Format string
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}

func envBool(key string, fallback bool) bool {
	if v := os.Getenv(key); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			return b
		}
	}
	return fallback
}

// Load 从 .env 文件与环境变量加载配置。
// 所有配置项通过 TASK_MANAGER_* 前缀的环境变量读取。
// 自动加载同目录下的 .env 文件（若存在），方便本地开发。
// 已在 shell 中导出的同名变量优先级高于 .env（godotenv 不覆盖已有环境变量）。
func Load() (*Config, error) {
	_ = godotenv.Load(".env")

	corsStr := env("TASK_MANAGER_SERVER_CORS_ORIGINS", "")
	var corsOrigins []string
	if corsStr != "" {
		for _, s := range strings.Split(corsStr, ",") {
			if t := strings.TrimSpace(s); t != "" {
				corsOrigins = append(corsOrigins, t)
			}
		}
	}

	cfg := &Config{
		Server: ServerConfig{
			Host:        env("TASK_MANAGER_SERVER_HOST", "0.0.0.0"),
			Port:        envInt("TASK_MANAGER_SERVER_PORT", 8080),
			CorsOrigins: corsOrigins,
		},
		Database: DatabaseConfig{
			Driver: env("TASK_MANAGER_DATABASE_DRIVER", "sqlite"),
			DSN:    env("TASK_MANAGER_DATABASE_DSN", "./data/task_manager.db"),
		},
		Auth: AuthConfig{
			JWTSecret:         env("TASK_MANAGER_AUTH_JWT_SECRET", ""),
			JWTExpiryHours:    envInt("TASK_MANAGER_AUTH_JWT_EXPIRY_HOURS", 24),
			LoginLockAttempts: envInt("TASK_MANAGER_AUTH_LOGIN_LOCK_ATTEMPTS", 5),
			LoginLockMinutes:  envInt("TASK_MANAGER_AUTH_LOGIN_LOCK_MINUTES", 15),
		},
		Scheduler: SchedulerConfig{
			NotificationScanInterval: env("TASK_MANAGER_SCHEDULER_NOTIFICATION_SCAN_INTERVAL", "10m"),
		},
		MCP: MCPConfig{
			Enabled:        envBool("TASK_MANAGER_MCP_ENABLED", true),
			Path:           env("TASK_MANAGER_MCP_PATH", "/mcp"),
			ExternalURL:    env("TASK_MANAGER_MCP_EXTERNAL_URL", ""),
			ServerName:     env("TASK_MANAGER_MCP_SERVER_NAME", "ops-timer-mcp"),
			ServerVersion:  env("TASK_MANAGER_MCP_SERVER_VERSION", "2.0.0"),
			TimeoutSeconds: envInt("TASK_MANAGER_MCP_TIMEOUT_SECONDS", 30),
		},
		SMTP: SMTPConfig{
			Host:     env("TASK_MANAGER_SMTP_HOST", ""),
			Port:     envInt("TASK_MANAGER_SMTP_PORT", 587),
			Username: env("TASK_MANAGER_SMTP_USERNAME", ""),
			Password: env("TASK_MANAGER_SMTP_PASSWORD", ""),
			From:     env("TASK_MANAGER_SMTP_FROM", ""),
		},
		Log: LogConfig{
			Level:  env("TASK_MANAGER_LOG_LEVEL", "info"),
			Format: env("TASK_MANAGER_LOG_FORMAT", "json"),
		},
	}

	return cfg, nil
}
