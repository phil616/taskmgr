package handler

import (
	"bytes"
	"encoding/json"
	"errors"
	"html/template"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"ops-timer-backend/internal/service"
)

type OAuthHandler struct{ Service *service.OAuthService }

func NewOAuthHandler(s *service.OAuthService) *OAuthHandler { return &OAuthHandler{Service: s} }

func oauthHeaders(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	c.Header("Pragma", "no-cache")
	c.Header("Referrer-Policy", "no-referrer")
}

func oauthFailure(c *gin.Context, err error) {
	oauthHeaders(c)
	status, code := http.StatusBadRequest, "server_error"
	switch {
	case errors.Is(err, service.ErrOAuthRequest), errors.Is(err, service.ErrOAuthGrant), errors.Is(err, service.ErrOAuthScope):
		code = err.Error()
	case errors.Is(err, service.ErrOAuthClient):
		code = "invalid_client"
	case errors.Is(err, service.ErrOAuthTarget):
		code = "invalid_target"
	case errors.Is(err, service.ErrOAuthRateLimit):
		status, code = http.StatusTooManyRequests, "temporarily_unavailable"
	default:
		status = http.StatusInternalServerError
	}
	c.JSON(status, gin.H{"error": code})
}

func (h *OAuthHandler) Metadata(c *gin.Context) {
	s := h.Service
	metadata := gin.H{
		"issuer":                                         s.Issuer,
		"authorization_endpoint":                         s.Issuer + "/oauth/authorize",
		"token_endpoint":                                 s.Issuer + "/oauth/token",
		"revocation_endpoint":                            s.Issuer + "/oauth/revoke",
		"response_types_supported":                       []string{"code"},
		"grant_types_supported":                          []string{"authorization_code", "refresh_token"},
		"token_endpoint_auth_methods_supported":          []string{"none"},
		"revocation_endpoint_auth_methods_supported":     []string{"none"},
		"code_challenge_methods_supported":               []string{"S256"},
		"scopes_supported":                               service.OAuthScopes,
		"authorization_response_iss_parameter_supported": true,
		"client_id_metadata_document_supported":          false,
	}
	if s.DynamicRegistration {
		metadata["registration_endpoint"] = s.Issuer + "/oauth/register"
	}
	c.JSON(200, metadata)
}

func (h *OAuthHandler) ResourceMetadata(c *gin.Context) {
	c.JSON(200, gin.H{"resource": h.Service.Resource, "authorization_servers": []string{h.Service.Issuer}, "scopes_supported": service.OAuthScopes, "bearer_methods_supported": []string{"header"}, "resource_name": "Task Manager API and MCP"})
}

func (h *OAuthHandler) Register(c *gin.Context) {
	oauthHeaders(c)
	if c.Request.URL.Path == "/oauth/register" && !h.Service.DynamicRegistration {
		c.JSON(403, gin.H{"error": "access_denied"})
		return
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 16<<10)
	var req struct {
		Name       string   `json:"client_name"`
		Redirects  []string `json:"redirect_uris"`
		AuthMethod string   `json:"token_endpoint_auth_method"`
		Grants     []string `json:"grant_types"`
		Responses  []string `json:"response_types"`
	}
	if c.ShouldBindJSON(&req) != nil {
		oauthFailure(c, service.ErrOAuthRequest)
		return
	}
	if req.AuthMethod != "" && req.AuthMethod != "none" {
		c.JSON(400, gin.H{"error": "invalid_client_metadata"})
		return
	}
	for _, grant := range req.Grants {
		if grant != "authorization_code" && grant != "refresh_token" {
			c.JSON(400, gin.H{"error": "invalid_client_metadata"})
			return
		}
	}
	for _, response := range req.Responses {
		if response != "code" {
			c.JSON(400, gin.H{"error": "invalid_client_metadata"})
			return
		}
	}
	if req.Name == "" {
		req.Name = "External agent"
	}
	// Use the actual network peer; untrusted forwarding headers cannot bypass limits.
	client, err := h.Service.RegisterClient(req.Name, req.Redirects, c.RemoteIP())
	if err != nil {
		oauthFailure(c, err)
		return
	}
	c.JSON(201, gin.H{"client_id": client.ID, "client_name": client.Name, "redirect_uris": client.RedirectURIs, "client_id_issued_at": client.CreatedAt.Unix(), "token_endpoint_auth_method": "none", "grant_types": []string{"authorization_code", "refresh_token"}, "response_types": []string{"code"}})
}

var consentPage = template.Must(template.New("consent").Parse(`<!doctype html>
<html lang="zh-CN"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1"><title>任务管理器 · 授权智能体</title></head>
<body><main><h1>授权外部智能体</h1><p>应用 <strong>{{.Name}}</strong> 请求操作此任务管理器。</p>
<p>应用名称由注册者提供，请核对回调地址：<code>{{.Redirect}}</code></p>
<p>授权资源：<code>{{.Resource}}</code></p><p>将授予以下权限：</p><ul>{{range .Scopes}}<li>{{.}}</li>{{end}}</ul>
<p>授权有效期为 30 天，可提前撤销。只有确认信任此应用时才授权。</p>
{{if .Error}}<p role="alert">{{.Error}}</p>{{end}}
<form action="/oauth/authorize" method="post"><input type="hidden" name="request" value="{{.Request}}">
<p><label>用户名 <input name="username" autocomplete="username" maxlength="32"></label></p>
<p><label>密码 <input type="password" name="password" autocomplete="current-password" maxlength="72"></label></p>
<button type="submit" name="decision" value="approve">登录并允许授权</button> <button type="submit" name="decision" value="deny">拒绝</button>
</form></main></body></html>`))

var oauthMCPInfo = template.Must(template.New("mcp-oauth").Parse(`<!doctype html>
<html lang="zh-CN"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1"><title>任务管理器 MCP · OAuth</title></head>
<body><main><h1>任务管理器 MCP</h1><p>已启用 OAuth。将以下配置加入支持 OAuth 的 MCP 客户端，连接时在浏览器中登录并确认权限。</p>
<pre>{{.Config}}</pre><p>默认权限为只读；写入操作需申请 data:write，密钥读写需单独授权。</p>
<p><a href="{{.ConfigURL}}">JSON 配置</a> · <a href="{{.MetadataURL}}">OAuth 授权服务器元数据</a></p>
</main></body></html>`))

func (h *MCPHandler) writeOAuthInfo(c *gin.Context) {
	raw, err := json.MarshalIndent(gin.H{"mcpServers": gin.H{h.serverName: gin.H{"url": h.oauthService.Resource}}}, "", "  ")
	if err != nil {
		oauthFailure(c, err)
		return
	}
	var body bytes.Buffer
	if err := oauthMCPInfo.Execute(&body, gin.H{"Config": string(raw), "ConfigURL": h.oauthService.Resource + "/config", "MetadataURL": h.oauthService.Issuer + "/.well-known/oauth-authorization-server"}); err != nil {
		oauthFailure(c, err)
		return
	}
	c.Data(200, "text/html; charset=utf-8", body.Bytes())
}

func (h *OAuthHandler) renderConsent(c *gin.Context, a *service.OAuthAuthorization, form, name, message string, status int) {
	labels := map[string]string{"data:read": "读取任务、项目、笔记、日程和财务数据及个人资料", "data:write": "创建、修改和删除任务、项目、笔记、日程和财务数据", "secrets:read": "读取密钥（包括明文）和密钥审计日志", "secrets:write": "创建、修改和删除密钥"}
	var scopes []string
	for _, scope := range strings.Fields(a.Scope) {
		scopes = append(scopes, labels[scope])
	}
	var body bytes.Buffer
	if err := consentPage.Execute(&body, gin.H{"Name": name, "Redirect": a.RedirectURI, "Resource": a.Resource, "Scopes": scopes, "Request": form, "Error": message}); err != nil {
		oauthFailure(c, err)
		return
	}
	c.Header("Content-Security-Policy", "default-src 'none'; form-action 'self'; frame-ancestors 'none'; base-uri 'none'")
	c.Header("X-Frame-Options", "DENY")
	// no-referrer makes browsers send Origin: null on native form POSTs,
	// which our consent endpoint correctly rejects. Preserve same-origin form
	// identity while withholding the authorization URL from external origins.
	c.Header("Referrer-Policy", "same-origin")
	c.Data(status, "text/html; charset=utf-8", body.Bytes())
}

func (h *OAuthHandler) Authorize(c *gin.Context) {
	oauthHeaders(c)
	if c.Request.Method == http.MethodGet {
		q, err := url.ParseQuery(c.Request.URL.RawQuery)
		if err != nil {
			oauthFailure(c, service.ErrOAuthRequest)
			return
		}
		for _, values := range q {
			if len(values) != 1 {
				oauthFailure(c, service.ErrOAuthRequest)
				return
			}
		}
		a := &service.OAuthAuthorization{ClientID: q.Get("client_id"), RedirectURI: q.Get("redirect_uri"), ResponseType: q.Get("response_type"), Challenge: q.Get("code_challenge"), ChallengeMethod: q.Get("code_challenge_method"), Scope: q.Get("scope"), State: q.Get("state"), Resource: q.Get("resource")}
		client, err := h.Service.ValidateAuthorization(a)
		// Invalid requests are never redirected to an untrusted callback.
		if err != nil {
			oauthFailure(c, err)
			return
		}
		form, csrf, err := h.Service.SignConsent(a)
		if err != nil {
			oauthFailure(c, err)
			return
		}
		http.SetCookie(c.Writer, &http.Cookie{Name: "taskmgr_oauth_csrf", Value: csrf, Path: "/oauth/authorize", HttpOnly: true, Secure: strings.HasPrefix(h.Service.Issuer, "https://"), SameSite: http.SameSiteLaxMode, MaxAge: 600})
		h.renderConsent(c, a, form, client.Name, "", 200)
		return
	}
	if !oauthForm(c) {
		return
	}
	if origin := c.GetHeader("Origin"); (origin != "" && origin != h.Service.Issuer) || c.GetHeader("Sec-Fetch-Site") == "cross-site" {
		c.AbortWithStatus(403)
		return
	}
	csrf, _ := c.Cookie("taskmgr_oauth_csrf")
	a, err := h.Service.VerifyConsent(c.PostForm("request"), csrf)
	if err != nil {
		oauthFailure(c, err)
		return
	}
	client, err := h.Service.ValidateAuthorization(a)
	if err != nil {
		oauthFailure(c, err)
		return
	}
	callback, _ := url.Parse(a.RedirectURI)
	q := callback.Query()
	if a.State != "" {
		q.Set("state", a.State)
	}
	q.Set("iss", h.Service.Issuer)
	switch c.PostForm("decision") {
	case "deny":
		q.Set("error", "access_denied")
	case "approve":
		code, err := h.Service.Authorize(a, c.PostForm("username"), c.PostForm("password"))
		if err != nil {
			status, message := 401, "用户名或密码错误"
			if errors.Is(err, service.ErrAccountLocked) {
				status, message = 429, "账户已锁定，请稍后重试"
			} else if !errors.Is(err, service.ErrInvalidCredentials) {
				oauthFailure(c, err)
				return
			}
			h.renderConsent(c, a, c.PostForm("request"), client.Name, message, status)
			return
		}
		q.Set("code", code)
	default:
		oauthFailure(c, service.ErrOAuthRequest)
		return
	}
	http.SetCookie(c.Writer, &http.Cookie{Name: "taskmgr_oauth_csrf", Value: "", Path: "/oauth/authorize", HttpOnly: true, Secure: strings.HasPrefix(h.Service.Issuer, "https://"), SameSite: http.SameSiteLaxMode, MaxAge: -1, Expires: time.Unix(1, 0)})
	callback.RawQuery = q.Encode()
	c.Redirect(http.StatusSeeOther, callback.String())
}

func oauthForm(c *gin.Context) bool {
	oauthHeaders(c)
	if c.ContentType() != "application/x-www-form-urlencoded" {
		oauthFailure(c, service.ErrOAuthRequest)
		return false
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 32<<10)
	if c.Request.ParseForm() != nil {
		oauthFailure(c, service.ErrOAuthRequest)
		return false
	}
	for _, values := range c.Request.PostForm {
		if len(values) != 1 {
			oauthFailure(c, service.ErrOAuthRequest)
			return false
		}
	}
	return true
}

func (h *OAuthHandler) Token(c *gin.Context) {
	if !oauthForm(c) {
		return
	}
	if c.GetHeader("Authorization") != "" || c.PostForm("client_secret") != "" {
		oauthFailure(c, service.ErrOAuthClient)
		return
	}
	var result *service.OAuthTokenResponse
	var err error
	switch c.PostForm("grant_type") {
	case "authorization_code":
		result, err = h.Service.ExchangeCode(c.PostForm("client_id"), c.PostForm("code"), c.PostForm("redirect_uri"), c.PostForm("code_verifier"), c.PostForm("resource"))
	case "refresh_token":
		result, err = h.Service.Refresh(c.PostForm("client_id"), c.PostForm("refresh_token"), c.PostForm("scope"), c.PostForm("resource"))
	default:
		c.JSON(400, gin.H{"error": "unsupported_grant_type"})
		return
	}
	if err != nil {
		oauthFailure(c, err)
		return
	}
	c.JSON(200, result)
}

func (h *OAuthHandler) Revoke(c *gin.Context) {
	if !oauthForm(c) {
		return
	}
	if c.PostForm("client_id") == "" || len(c.PostForm("token")) > 128 {
		oauthFailure(c, service.ErrOAuthRequest)
		return
	}
	if err := h.Service.RevokeToken(c.PostForm("client_id"), c.PostForm("token")); err != nil {
		oauthFailure(c, err)
		return
	}
	c.Status(200)
}

func (h *OAuthHandler) ListGrants(c *gin.Context) {
	oauthHeaders(c)
	grants, err := h.Service.ListGrants(c.GetString("user_id"))
	if err != nil {
		oauthFailure(c, err)
		return
	}
	c.JSON(200, gin.H{"grants": grants})
}

func (h *OAuthHandler) RevokeGrant(c *gin.Context) {
	oauthHeaders(c)
	if err := h.Service.RevokeGrant(c.GetString("user_id"), c.Param("id")); err != nil {
		oauthFailure(c, err)
		return
	}
	c.Status(204)
}
