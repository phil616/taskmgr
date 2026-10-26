package service

import (
	"context"
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"ops-timer-backend/internal/config"
	"ops-timer-backend/internal/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type MCPSettingsService struct {
	db     *gorm.DB
	auth   *AuthService
	secret string
}

func NewMCPSettingsService(db *gorm.DB, auth *AuthService, secret string) *MCPSettingsService {
	return &MCPSettingsService{db: db, auth: auth, secret: secret}
}

// Initialize imports legacy environment configuration only when no database row exists.
func (s *MCPSettingsService) Initialize(ctx context.Context, cfg *config.Config) error {
	var count int64
	if err := s.db.WithContext(ctx).Model(&model.MCPSettings{}).Where("id = 1").Count(&count).Error; err != nil {
		return err
	}
	if count != 0 {
		return nil
	}
	initial := model.MCPSettings{
		ID: 1, Enabled: cfg.MCP.Enabled, Path: cfg.MCP.Path,
		ExternalURL: cfg.MCP.ExternalURL, ServerName: cfg.MCP.ServerName,
		ServerVersion: cfg.MCP.ServerVersion, TimeoutSeconds: cfg.MCP.TimeoutSeconds,
		APITokenEnabled: true, OAuthEnabled: cfg.OAuth.Issuer != "",
		OAuthIssuer: cfg.OAuth.Issuer, OAuthDynamicRegistration: cfg.OAuth.DynamicRegistration,
	}
	if err := ValidateMCPSettings(&initial); err != nil {
		return err
	}
	return s.db.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(&initial).Error
}

func (s *MCPSettingsService) Get(ctx context.Context) (*model.MCPSettings, error) {
	var settings model.MCPSettings
	if err := s.db.WithContext(ctx).First(&settings, 1).Error; err != nil {
		return nil, err
	}
	return &settings, nil
}

func (s *MCPSettingsService) Save(ctx context.Context, settings model.MCPSettings, userID string) (*model.MCPSettings, error) {
	if err := ValidateMCPSettings(&settings); err != nil {
		return nil, err
	}
	settings.ID = 1
	settings.UpdatedBy = userID
	// Save writes false and empty values as well as non-zero fields.
	if err := s.db.WithContext(ctx).Save(&settings).Error; err != nil {
		return nil, err
	}
	return &settings, nil
}

func (s *MCPSettingsService) OAuth(settings *model.MCPSettings) (*OAuthService, error) {
	if !settings.OAuthEnabled {
		return nil, nil
	}
	return NewOAuthService(s.db, s.auth, settings.OAuthIssuer, settings.Path, s.secret, settings.OAuthDynamicRegistration)
}

var mcpPathPattern = regexp.MustCompile(`^/[A-Za-z0-9_-]+(/[A-Za-z0-9_-]+)*$`)

func ValidateMCPSettings(s *model.MCPSettings) error {
	s.Path = strings.TrimSpace(s.Path)
	s.ServerName = strings.TrimSpace(s.ServerName)
	s.ServerVersion = strings.TrimSpace(s.ServerVersion)
	s.ExternalURL = strings.TrimRight(strings.TrimSpace(s.ExternalURL), "/")
	s.OAuthIssuer = strings.TrimRight(strings.TrimSpace(s.OAuthIssuer), "/")
	if len(s.Path) > 200 || !mcpPathPattern.MatchString(s.Path) {
		return fmt.Errorf("MCP 路径须为 /mcp 或 /agents/mcp 形式，不含查询、转义或特殊字符")
	}
	first := strings.Split(strings.TrimPrefix(s.Path, "/"), "/")[0]
	if first == "api" || first == "oauth" || first == "health" {
		return fmt.Errorf("MCP 路径不能占用 api、oauth 或 health 路由")
	}
	if s.ServerName == "" || len(s.ServerName) > 100 || strings.ContainsAny(s.ServerName, "\r\n\t") {
		return fmt.Errorf("服务名称须为 1–100 字节且不含控制字符")
	}
	if s.ServerVersion == "" || len(s.ServerVersion) > 50 || strings.ContainsAny(s.ServerVersion, "\r\n\t") {
		return fmt.Errorf("服务版本须为 1–50 字节且不含控制字符")
	}
	if s.TimeoutSeconds < 1 || s.TimeoutSeconds > 300 {
		return fmt.Errorf("MCP 请求超时须为 1–300 秒")
	}
	for _, value := range []string{s.ExternalURL, s.OAuthIssuer} {
		if value == "" {
			continue
		}
		u, err := url.Parse(value)
		if err != nil || len(value) > 2048 || !safeOAuthURL(u) || u.RawQuery != "" || u.ForceQuery || u.Path != "" {
			return fmt.Errorf("外部地址和 OAuth issuer 须为不带路径的 HTTPS 来源；HTTP 仅允许本机开发地址")
		}
	}
	if s.OAuthEnabled && s.OAuthIssuer == "" {
		return fmt.Errorf("启用 OAuth 时必须设置授权服务器地址")
	}
	if s.Enabled && !s.APITokenEnabled && !s.OAuthEnabled {
		return fmt.Errorf("启用 MCP 时至少需要一种认证方式")
	}
	return nil
}
