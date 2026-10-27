package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"ops-timer-backend/internal/model"
	"ops-timer-backend/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const (
	mcpProtocolVersion  = "2025-11-25"
	mcpBackendRequest   = "backend_request"
	defaultMCPServer    = "ops-timer-mcp"
	defaultMCPVersion   = "2.0.0"
	defaultHTTPTimeoutS = 30
)

// ──────────────────────────────────────────────
//  MCPHandler 核心
// ──────────────────────────────────────────────

type MCPHandler struct {
	authService     *service.AuthService
	oauthService    *service.OAuthService
	httpClient      *http.Client
	baseURL         string
	externalURL     string
	mcpPath         string
	serverName      string
	serverVer       string
	disableAPIToken bool
	allowGeneric    bool
	allowSensitive  bool
	origins         []string
	schemaCache     *mcp.SchemaCache
}

type MCPHandlerConfig struct {
	AuthService     *service.AuthService
	OAuthService    *service.OAuthService
	BaseURL         string
	ExternalURL     string
	MCPPath         string
	ServerName      string
	ServerVersion   string
	TimeoutSeconds  int
	DisableAPIToken bool
	AllowGeneric    bool
	AllowSensitive  bool
	Origins         []string
}

func NewMCPHandler(cfg MCPHandlerConfig) *MCPHandler {
	timeout := cfg.TimeoutSeconds
	if timeout <= 0 {
		timeout = defaultHTTPTimeoutS
	}
	serverName := strings.TrimSpace(cfg.ServerName)
	if serverName == "" {
		serverName = defaultMCPServer
	}
	serverVer := strings.TrimSpace(cfg.ServerVersion)
	if serverVer == "" {
		serverVer = defaultMCPVersion
	}
	mcpPath := strings.TrimSpace(cfg.MCPPath)
	if mcpPath == "" {
		mcpPath = "/mcp"
	}
	return &MCPHandler{
		disableAPIToken: cfg.DisableAPIToken, schemaCache: &mcp.SchemaCache{},
		allowGeneric: cfg.AllowGeneric, allowSensitive: cfg.AllowSensitive, origins: append([]string(nil), cfg.Origins...),
		authService:  cfg.AuthService,
		oauthService: cfg.OAuthService,
		httpClient:   &http.Client{Timeout: time.Duration(timeout) * time.Second, CheckRedirect: func(req *http.Request, via []*http.Request) error { return http.ErrUseLastResponse }},
		baseURL:      strings.TrimRight(cfg.BaseURL, "/"),
		externalURL:  strings.TrimRight(strings.TrimSpace(cfg.ExternalURL), "/"),
		mcpPath:      mcpPath,
		serverName:   serverName,
		serverVer:    serverVer,
	}
}

// Configured returns an immutable request snapshot; shared handlers are never mutated.
func (h *MCPHandler) Configured(s *model.MCPSettings, oauth *service.OAuthService) *MCPHandler {
	copy := *h
	copy.oauthService = oauth
	copy.mcpPath, copy.externalURL = s.Path, s.ExternalURL
	copy.serverName, copy.serverVer = s.ServerName, s.ServerVersion
	copy.disableAPIToken = !s.APITokenEnabled
	copy.allowGeneric, copy.allowSensitive = s.GenericToolsEnabled, s.SensitiveToolsEnabled
	client := *h.httpClient
	client.Timeout = time.Duration(s.TimeoutSeconds) * time.Second
	copy.httpClient = &client
	return &copy
}

// GetConfig 返回可直接粘贴到 MCP 客户端的 JSON 配置（GET /mcp/config）
func (h *MCPHandler) GetConfig(c *gin.Context) {
	if h.oauthService != nil {
		c.JSON(200, gin.H{"mcpServers": gin.H{h.serverName: gin.H{"url": h.oauthService.Resource}}, "oauth": gin.H{"issuer": h.oauthService.Issuer, "resource": h.oauthService.Resource, "authorization_server_metadata": h.oauthService.Issuer + "/.well-known/oauth-authorization-server", "scopes_supported": service.OAuthScopes}})
		return
	}
	mcpURL := h.externalURL + h.mcpPath
	if h.externalURL == "" {
		scheme := "http"
		if c.Request.TLS != nil || c.GetHeader("X-Forwarded-Proto") == "https" {
			scheme = "https"
		}
		mcpURL = scheme + "://" + c.Request.Host + h.mcpPath
	}

	c.JSON(http.StatusOK, gin.H{
		"mcpServers": gin.H{
			h.serverName: gin.H{
				"url": mcpURL,
				"headers": gin.H{
					"X-API-Token": "<your-api-token>",
				},
			},
		},
		"_usage": gin.H{
			"steps": []string{
				"1. 登录后在「设置 → API Token」获取你的 Token",
				"2. 将上方 JSON 中的 <your-api-token> 替换为你的真实 Token",
				"3. 将 mcpServers 部分粘贴到你的 MCP 客户端配置文件中（如 Cursor Settings → MCP、Claude Desktop mcp_config.json 等）",
			},
			"total_tools":      len(h.visibleTools()),
			"server_version":   h.serverVer,
			"protocol_version": mcpProtocolVersion,
		},
	})
}

// GetInfo 返回面向人类的 MCP 端点说明页（GET /mcp/info）
func (h *MCPHandler) GetInfo(c *gin.Context) {
	if h.oauthService != nil {
		h.writeOAuthInfo(c)
		return
	}
	mcpURL := h.externalURL + h.mcpPath
	if h.externalURL == "" {
		scheme := "http"
		if c.Request.TLS != nil || c.GetHeader("X-Forwarded-Proto") == "https" {
			scheme = "https"
		}
		mcpURL = scheme + "://" + c.Request.Host + h.mcpPath
	}

	tools := h.visibleTools()
	groups := []struct {
		Label string
		Names []string
	}{
		{"认证", nil}, {"计时单元", nil}, {"项目", nil}, {"待办事项", nil},
		{"待办分组", nil}, {"笔记", nil}, {"笔记分组", nil}, {"通知", nil}, {"日程", nil}, {"钱包", nil},
		{"收支分类", nil}, {"收支记录", nil}, {"统计", nil}, {"密钥管理", nil}, {"通用", nil},
	}
	prefix := []string{
		"auth_", "unit_", "project_", "todo_",
		"todo_group_", "note_", "note_group_", "notification_", "schedule_", "wallet_",
		"budget_category_", "transaction_", "budget_stats", "secret_", "backend_request",
	}

	for _, t := range tools {
		placed := false
		for i := len(prefix) - 1; i >= 0; i-- {
			if strings.HasPrefix(t.Name, prefix[i]) || t.Name == prefix[i] {
				groups[i].Names = append(groups[i].Names, t.Name)
				placed = true
				break
			}
		}
		if !placed {
			groups[len(groups)-1].Names = append(groups[len(groups)-1].Names, t.Name)
		}
	}

	var toolListHTML strings.Builder
	for _, g := range groups {
		if len(g.Names) == 0 {
			continue
		}
		toolListHTML.WriteString(`<div class="tool-group"><h3>`)
		toolListHTML.WriteString(g.Label)
		toolListHTML.WriteString(fmt.Sprintf(` <span class="badge">%d</span></h3><ul>`, len(g.Names)))
		for _, n := range g.Names {
			toolListHTML.WriteString(`<li><code>`)
			toolListHTML.WriteString(n)
			toolListHTML.WriteString(`</code></li>`)
		}
		toolListHTML.WriteString(`</ul></div>`)
	}

	configJSON, _ := json.MarshalIndent(gin.H{"mcpServers": gin.H{h.serverName: gin.H{
		"url": mcpURL, "headers": gin.H{"X-API-Token": "<your-api-token>"},
	}}}, "", "  ")

	html := fmt.Sprintf(`<!DOCTYPE html>
<html lang="zh-CN">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>MCP Endpoint — %s</title>
<style>
*{box-sizing:border-box;margin:0;padding:0}
body{font-family:-apple-system,BlinkMacSystemFont,"Segoe UI",Roboto,"Helvetica Neue",Arial,sans-serif;
background:#0f172a;color:#e2e8f0;min-height:100vh;display:flex;justify-content:center;padding:2rem 1rem}
.container{max-width:720px;width:100%%}
.hero{text-align:center;margin-bottom:2.5rem}
.hero h1{font-size:1.8rem;background:linear-gradient(135deg,#38bdf8,#818cf8);-webkit-background-clip:text;
-webkit-text-fill-color:transparent;margin-bottom:.5rem}
.hero .subtitle{color:#94a3b8;font-size:1rem}
.card{background:#1e293b;border:1px solid #334155;border-radius:12px;padding:1.5rem;margin-bottom:1.5rem}
.card h2{font-size:1.15rem;color:#f1f5f9;margin-bottom:1rem;display:flex;align-items:center;gap:.5rem}
.card h2 .icon{font-size:1.3rem}
.alert{background:#1e3a5f;border:1px solid #2563eb;border-radius:8px;padding:1rem;
font-size:.9rem;color:#93c5fd;line-height:1.6}
.config-wrap{position:relative}
pre{background:#0f172a;border:1px solid #334155;border-radius:8px;padding:1rem;
font-size:.85rem;line-height:1.5;overflow-x:auto;color:#a5f3fc;font-family:"Fira Code",Consolas,monospace}
.copy-btn{position:absolute;top:.5rem;right:.5rem;background:#334155;color:#e2e8f0;border:none;
border-radius:6px;padding:.35rem .75rem;cursor:pointer;font-size:.8rem;transition:all .2s}
.copy-btn:hover{background:#475569}
.copy-btn.copied{background:#059669;color:#fff}
.steps{list-style:none;counter-reset:s}
.steps li{counter-increment:s;padding:.6rem 0 .6rem 2.2rem;position:relative;font-size:.9rem;
color:#cbd5e1;border-bottom:1px solid #1e293b}
.steps li:last-child{border-bottom:none}
.steps li::before{content:counter(s);position:absolute;left:0;top:.55rem;width:1.6rem;height:1.6rem;
border-radius:50%%;background:#334155;color:#38bdf8;font-size:.8rem;display:flex;align-items:center;
justify-content:center;font-weight:600}
.steps code{background:#334155;padding:.15rem .4rem;border-radius:4px;font-size:.82rem;color:#a5f3fc}
.tool-group{margin-bottom:1rem}
.tool-group h3{font-size:.95rem;color:#94a3b8;margin-bottom:.4rem;display:flex;align-items:center;gap:.4rem}
.badge{background:#334155;color:#38bdf8;font-size:.75rem;padding:.1rem .45rem;border-radius:10px}
.tool-group ul{display:flex;flex-wrap:wrap;gap:.4rem;list-style:none}
.tool-group li code{background:#0f172a;border:1px solid #334155;padding:.2rem .5rem;border-radius:4px;
font-size:.78rem;color:#67e8f9;display:inline-block}
.meta{text-align:center;color:#475569;font-size:.8rem;margin-top:1.5rem}
</style>
</head>
<body>
<div class="container">

<div class="hero">
  <h1>🤖 MCP Endpoint</h1>
  <p class="subtitle">Model Context Protocol — %s v%s</p>
</div>

<div class="card">
  <h2><span class="icon">⚠️</span> 这不是普通网页</h2>
  <div class="alert">
    此端点是 <strong>MCP（Model Context Protocol）</strong> 服务接口，专为 AI 助手设计。<br>
    AI 客户端（如 Cursor、Claude Desktop）通过 <strong>POST</strong> 请求发送 JSON-RPC 调用来使用此接口。<br>
    你正在通过浏览器 GET 请求访问，所以看到了此说明页面。
  </div>
</div>

<div class="card">
  <h2><span class="icon">⚙️</span> 快速配置</h2>
  <div class="config-wrap">
    <button class="copy-btn" onclick="copyConfig(this)">复制</button>
    <pre id="config-json">%s</pre>
  </div>
  <ol class="steps" style="margin-top:1rem">
    <li>登录系统后，在 <strong>设置 → API Token</strong> 获取你的 Token</li>
    <li>将上方配置中的 <code>&lt;your-api-token&gt;</code> 替换为真实 Token</li>
    <li>粘贴到 MCP 客户端配置中：<br>
      <strong>Cursor</strong> → Settings → MCP<br>
      <strong>Claude Desktop</strong> → <code>mcp_config.json</code><br>
      <strong>其他客户端</strong> → 参照各自文档</li>
  </ol>
</div>

<div class="card">
  <h2><span class="icon">🛠️</span> 可用工具（共 %d 个）</h2>
  %s
</div>

<p class="meta">MCP Protocol %s · <a href="%s/config" style="color:#475569">JSON 配置接口</a></p>

</div>
<script>
function copyConfig(btn){
  var t=document.getElementById('config-json').textContent;
  navigator.clipboard.writeText(t).then(function(){
    btn.textContent='已复制 ✓';btn.classList.add('copied');
    setTimeout(function(){btn.textContent='复制';btn.classList.remove('copied')},2000);
  });
}
</script>
</body>
</html>`,
		html.EscapeString(h.serverName),
		html.EscapeString(h.serverName), html.EscapeString(h.serverVer),
		html.EscapeString(string(configJSON)),
		len(tools), toolListHTML.String(),
		mcpProtocolVersion, html.EscapeString(mcpURL),
	)

	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(html))
}

// dispatchTool 统一调度：具名工具 or 通用代理
func (h *MCPHandler) dispatchTool(ctx context.Context, apiToken, toolName string, args map[string]any) *mcp.CallToolResult {
	if args == nil {
		args = map[string]any{}
	}

	if toolName == mcpBackendRequest && !h.allowGeneric {
		return toolError("tool_disabled: 通用代理工具未启用")
	}
	if sensitiveTool(toolName) && !h.allowSensitive {
		return toolError("tool_disabled: 敏感工具未启用")
	}
	if toolName == mcpBackendRequest {
		return h.callBackend(ctx, apiToken, args)
	}

	call, err := toolToHTTP(toolName, args)
	if err != nil {
		return toolError(err.Error())
	}

	proxyArgs := map[string]any{
		"method": call.method,
		"path":   call.path,
	}
	if len(call.query) > 0 {
		proxyArgs["query"] = call.query
	}
	if len(call.body) > 0 {
		proxyArgs["body"] = call.body
	}

	return h.callBackend(ctx, apiToken, proxyArgs)
}

func (h *MCPHandler) validateAPIToken(c *gin.Context) (string, bool) {
	parts := strings.Fields(c.GetHeader("Authorization"))
	if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") && strings.HasPrefix(parts[1], "ot_") {
		if h.oauthService == nil {
			return "", false
		}
		identity, err := h.oauthService.ValidateAccess(parts[1])
		if err != nil {
			return "", false
		}
		c.Set("oauth_scope", identity.Scope)
		c.Set("oauth_client_id", identity.ClientID)
		c.Set("oauth_grant_id", identity.GrantID)
		c.Header("Cache-Control", "no-store")
		return parts[1], true
	}
	if h.disableAPIToken {
		return "", false
	}
	token := strings.TrimSpace(c.GetHeader("X-API-Token"))
	if token == "" {
		authHeader := strings.TrimSpace(c.GetHeader("Authorization"))
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
			token = strings.TrimSpace(parts[1])
		}
	}
	if token == "" {
		return "", false
	}
	if h.authService == nil {
		return "", false
	}
	if _, err := h.authService.FindByAPIToken(token); err != nil {
		return "", false
	}
	return token, true
}

func (h *MCPHandler) callBackend(ctx context.Context, apiToken string, args map[string]any) *mcp.CallToolResult {
	ctx, cancel := context.WithTimeout(ctx, h.httpClient.Timeout)
	defer cancel()
	method := strings.ToUpper(strings.TrimSpace(toString(args["method"])))
	path := strings.TrimSpace(toString(args["path"]))

	if method == "" || path == "" {
		return toolError("参数缺失: method 与 path 为必填")
	}
	if !isAllowedMethod(method) {
		return toolError("不支持的 HTTP 方法: " + method)
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	if !isAllowedPath(path) {
		return toolError("非法路径，仅允许 /api/v1/* 或 /health")
	}

	safeURL, err := h.buildSafeURL(path, args)
	if err != nil {
		return toolError(err.Error())
	}
	target, err := url.Parse(safeURL)
	if err != nil {
		return toolError("invalid_path")
	}
	if !h.allowSensitive && h.sensitivePath(method, target.Path) {
		return toolError("tool_disabled: 敏感操作未启用")
	}
	allowSecretValues := h.allowSensitive
	if strings.HasPrefix(apiToken, "ot_") {
		if h.oauthService == nil {
			return toolError("OAuth 未启用")
		}
		identity, err := h.oauthService.ValidateAccess(apiToken)
		target, parseErr := url.Parse(safeURL)
		if err != nil || parseErr != nil || !service.OAuthAllows(identity.Scope, method, target.Path) {
			return toolError("insufficient_scope: 当前授权不允许此操作")
		}
		allowSecretValues = allowSecretValues && service.OAuthAllows(identity.Scope, http.MethodGet, "/api/v1/secrets/key")
	}

	var bodyReader io.Reader
	if body, exists := args["body"]; exists {
		raw, err := json.Marshal(body)
		if err != nil {
			return toolError("body 不是合法 JSON: " + err.Error())
		}
		bodyReader = bytes.NewReader(raw)
	}

	req, err := http.NewRequestWithContext(ctx, method, safeURL, bodyReader)
	if err != nil {
		return toolError("创建请求失败: " + err.Error())
	}
	if strings.HasPrefix(apiToken, "ot_") {
		req.Header.Set("Authorization", "Bearer "+apiToken)
	} else {
		req.Header.Set("X-API-Token", apiToken)
	}
	if audit, ok := ctx.Value(mcpAuditKey{}).(*mcpAudit); ok {
		req.RemoteAddr = audit.remoteAddr
		req.Header.Set("X-Request-ID", audit.requestID)
	}
	req.Header.Set("Accept", "application/json")
	if bodyReader != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	if headers, ok := args["headers"].(map[string]any); ok {
		for k, v := range headers {
			k = strings.TrimSpace(k)
			if !strings.EqualFold(k, "Accept") && !strings.EqualFold(k, "Content-Type") && !strings.EqualFold(k, "If-Match") && !strings.EqualFold(k, "If-None-Match") {
				continue
			}
			req.Header.Set(k, toString(v))
		}
	}
	// Production uses an in-process transport sharing REST authentication and services.
	resp, err := h.httpClient.Do(req)
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return toolError("request_cancelled: " + err.Error())
		}
		return toolError("调用后端失败: " + err.Error())
	}
	defer resp.Body.Close()
	if audit, ok := ctx.Value(mcpAuditKey{}).(*mcpAudit); ok {
		audit.mu.Lock()
		audit.status = resp.StatusCode
		audit.mu.Unlock()
	}

	const maxRespBody = 10 << 20 // 10 MiB
	respBody, err := io.ReadAll(io.LimitReader(resp.Body, maxRespBody+1))
	if err != nil {
		return toolError("读取后端响应失败: " + err.Error())
	}

	if len(respBody) > maxRespBody {
		return toolError("response_too_large: 业务响应超过 10 MiB，请使用分页")
	}
	if err := ctx.Err(); err != nil {
		return toolError("request_cancelled: " + err.Error())
	}
	var parsed any
	kind := "empty"
	if resp.StatusCode == http.StatusNoContent {
		kind = "no_content"
	} else if len(respBody) > 0 {
		if err := json.Unmarshal(respBody, &parsed); err != nil {
			return toolError(fmt.Sprintf("invalid_backend_response: %s %s -> %d，业务响应不是 JSON", method, path, resp.StatusCode))
		}
		kind = "json"
	} else if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return toolError("empty_backend_response: 业务接口返回成功但没有正文")
	}
	if !allowSecretValues && (target.Path == "/api/v1/secrets" || strings.HasPrefix(target.Path, "/api/v1/secrets/")) {
		redactMCPSecretValue(parsed)
	}
	data := gin.H{"status": resp.StatusCode, "method": method, "path": req.URL.Path, "kind": kind, "data": parsed}
	raw, err := json.Marshal(data)
	if err != nil {
		return toolError("invalid_backend_response: 结果无法序列化")
	}
	isErr := resp.StatusCode < 200 || resp.StatusCode >= 300
	if obj, ok := parsed.(map[string]any); ok {
		if code, ok := obj["code"].(float64); ok && code != 0 {
			isErr = true
		}
	}
	return &mcp.CallToolResult{IsError: isErr, Content: []mcp.Content{&mcp.TextContent{Text: string(raw)}}, StructuredContent: data}
}

// ──────────────────────────────────────────────
//  辅助函数
// ──────────────────────────────────────────────

// buildSafeURL constructs a request URL from validated path and query args.
// The result is always anchored to the internal dispatcher with a whitelisted
// path prefix, so it is not controllable by external input beyond the
// already-validated path segment and query parameters.
func (h *MCPHandler) buildSafeURL(validatedPath string, args map[string]any) (string, error) {
	base, err := url.Parse(h.baseURL)
	if err != nil {
		return "", fmt.Errorf("baseURL 解析失败: %w", err)
	}

	ref, err := url.Parse(validatedPath)
	if err != nil {
		return "", fmt.Errorf("路径解析失败: %w", err)
	}

	resolved := base.ResolveReference(ref)

	// Post-parse safety: ensure the resolved host has not been altered
	// by a crafted path (e.g. "//evil.com/api/v1/...").
	if resolved.Host != base.Host || resolved.Scheme != base.Scheme {
		return "", fmt.Errorf("非法路径: 目标主机不符")
	}
	if !isAllowedPath(resolved.Path) {
		return "", fmt.Errorf("非法路径: 解析后路径不在白名单内")
	}

	if queryObj, ok := args["query"].(map[string]any); ok {
		q := resolved.Query()
		for k, v := range queryObj {
			if vs := toString(v); vs != "" {
				q.Set(k, vs)
			}
		}
		resolved.RawQuery = q.Encode()
	}

	return resolved.String(), nil
}

func isAllowedMethod(method string) bool {
	switch method {
	case http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
		return true
	default:
		return false
	}
}

func isAllowedPath(path string) bool {
	if strings.ContainsAny(path, "\\%?#") {
		return false
	}
	cleaned := path
	if strings.Contains(cleaned, "..") {
		return false
	}
	return cleaned == "/health" || strings.HasPrefix(cleaned, "/api/v1/")
}

func toString(v any) string {
	switch val := v.(type) {
	case string:
		return val
	case nil:
		return ""
	default:
		return fmt.Sprint(val)
	}
}

func toolError(msg string) *mcp.CallToolResult {
	return &mcp.CallToolResult{IsError: true, Content: []mcp.Content{&mcp.TextContent{Text: msg}}}
}

func pickStrings(args map[string]any, keys ...string) map[string]any {
	result := map[string]any{}
	for _, k := range keys {
		if v, ok := args[k]; ok {
			if s := toString(v); s != "" {
				result[k] = s
			}
		}
	}
	return result
}

func pickKeys(args map[string]any, keys ...string) map[string]any {
	result := map[string]any{}
	for _, k := range keys {
		if v, ok := args[k]; ok {
			result[k] = v
		}
	}
	return result
}

func omitKeys(args map[string]any, keys ...string) map[string]any {
	exclude := make(map[string]struct{}, len(keys))
	for _, k := range keys {
		exclude[k] = struct{}{}
	}
	result := map[string]any{}
	for k, v := range args {
		if _, skip := exclude[k]; !skip {
			result[k] = v
		}
	}
	return result
}
