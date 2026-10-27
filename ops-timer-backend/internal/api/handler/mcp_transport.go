package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/modelcontextprotocol/go-sdk/jsonrpc"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Handle delegates framing, negotiation, notifications and JSON-RPC validation to
// the official SDK. No session or authenticated identity survives an HTTP request.
func (h *MCPHandler) Handle(c *gin.Context) {
	// Gin enters dynamic NoRoute middleware with a pending 404. The SDK
	// may write a successful JSON body without explicitly writing status 200.
	c.Status(http.StatusOK)
	c.Header("Cache-Control", "no-store")
	if !h.validOrigin(c.Request) {
		c.AbortWithStatus(http.StatusForbidden)
		return
	}
	token, ok := h.validateAPIToken(c)
	if !ok {
		if h.oauthService != nil {
			c.Header("WWW-Authenticate", h.oauthService.Challenge("", "data:read"))
		}
		c.JSON(http.StatusUnauthorized, gin.H{"error": "MCP token 无效或缺失"})
		return
	}
	audit := &mcpAudit{remoteAddr: c.Request.RemoteAddr, requestID: c.GetString("request_id")}
	requestContext := context.WithValue(c.Request.Context(), mcpAuditKey{}, audit)
	scope, delegated := c.Get("oauth_scope")
	scopeText, _ := scope.(string)
	server := h.mcpServer(token, scopeText, delegated, requestContext)
	transport := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server { return server }, &mcp.StreamableHTTPOptions{Stateless: true, JSONResponse: true, MaxRequestBodyBytes: 4 << 20, DisableLocalhostProtection: h.trustedHost(c.Request.Host)})
	transport.ServeHTTP(c.Writer, c.Request)
	audit.mu.Lock()
	c.Set("mcp_method", audit.method)
	c.Set("mcp_tool", audit.tool)
	c.Set("mcp_error_category", audit.errorCategory)
	c.Set("mcp_backend_status", audit.status)
	c.Set("mcp_backend_latency", audit.latency)
	audit.mu.Unlock()
}

func (h *MCPHandler) mcpServer(token, scope string, delegated bool, requestContext context.Context) *mcp.Server {
	server := mcp.NewServer(&mcp.Implementation{Name: h.serverName, Version: h.serverVer}, &mcp.ServerOptions{
		SupportedProtocolVersions: []string{mcpProtocolVersion, "2025-06-18", "2025-03-26"}, SchemaCache: h.schemaCache,
		Capabilities: &mcp.ServerCapabilities{Tools: &mcp.ToolCapabilities{}},
	})
	server.AddReceivingMiddleware(func(next mcp.MethodHandler) mcp.MethodHandler {
		return func(ctx context.Context, method string, req mcp.Request) (mcp.Result, error) {
			if audit, ok := requestContext.Value(mcpAuditKey{}).(*mcpAudit); ok {
				audit.mu.Lock()
				audit.method = method
				audit.mu.Unlock()
			}
			if method == "initialize" {
				// The SDK uses an internal wire type for initialization. Inspect the
				// public Params representation rather than asserting a request alias.
				raw, marshalErr := json.Marshal(req.GetParams())
				var p mcp.InitializeParams
				decodeErr := json.Unmarshal(raw, &p)
				if marshalErr != nil || decodeErr != nil || p.ProtocolVersion == "" || p.Capabilities == nil || p.ClientInfo == nil || p.ClientInfo.Name == "" || p.ClientInfo.Version == "" {
					return nil, &jsonrpc.Error{Code: jsonrpc.CodeInvalidParams, Message: "initialize requires protocolVersion, capabilities and clientInfo (name/version)"}
				}
			}
			return next(ctx, method, req)
		}
	})
	for _, tool := range h.visibleTools() {
		if delegated && (tool.Name == mcpBackendRequest || !slices.Contains(strings.Fields(scope), toolScope(tool.Name))) {
			continue
		}
		mcp.AddTool(server, &mcp.Tool{Name: tool.Name, Description: tool.Description, InputSchema: tool.schema, OutputSchema: resolvedMCPOutputSchema},
			func(ctx context.Context, req *mcp.CallToolRequest, args map[string]any) (*mcp.CallToolResult, any, error) {
				ctx, cancel := context.WithCancel(ctx)
				defer cancel()
				stop := context.AfterFunc(requestContext, cancel)
				defer stop()
				if audit, ok := requestContext.Value(mcpAuditKey{}).(*mcpAudit); ok {
					ctx = context.WithValue(ctx, mcpAuditKey{}, audit)
				}
				start := time.Now()
				result := h.dispatchTool(ctx, token, tool.Name, args)
				if audit, ok := requestContext.Value(mcpAuditKey{}).(*mcpAudit); ok {
					audit.mu.Lock()
					audit.tool = tool.Name
					audit.latency = time.Since(start)
					if data, ok := result.StructuredContent.(gin.H); ok {
						audit.status, _ = data["status"].(int)
					}
					if result.IsError {
						audit.errorCategory = "tool_error"
						if len(result.Content) > 0 {
							if text, ok := result.Content[0].(*mcp.TextContent); ok {
								for _, category := range []string{"response_too_large", "invalid_backend_response", "empty_backend_response", "request_cancelled", "tool_disabled", "insufficient_scope"} {
									if strings.Contains(text.Text, category+":") {
										audit.errorCategory = category
										break
									}
								}
							}
						}
					}
					audit.mu.Unlock()
				}
				return result, result.StructuredContent, nil
			})
	}
	return server
}

var toolOutputSchema = gin.H{
	"type": "object", "required": []string{"status", "method", "path", "kind", "data"},
	"properties": gin.H{
		"status": gin.H{"type": "integer"}, "method": gin.H{"type": "string"}, "path": gin.H{"type": "string"},
		"kind": gin.H{"type": "string", "enum": []string{"json", "no_content", "empty"}}, "data": gin.H{},
	}, "additionalProperties": false,
}

var resolvedMCPOutputSchema = mustMCPSchema(toolOutputSchema)

func (h *MCPHandler) validOrigin(r *http.Request) bool {
	values := r.Header.Values("Origin")
	if len(values) == 0 {
		return true
	}
	if len(values) != 1 || values[0] == "" {
		return false
	}
	origin := values[0]
	u, err := url.Parse(origin)
	if err != nil || (u.Scheme != "https" && u.Scheme != "http") || u.Host == "" || u.User != nil || u.Path != "" || u.RawQuery != "" || u.Fragment != "" || strings.ContainsAny(origin, "?#") {
		return false
	}
	// Only explicitly configured origins are trusted, never an arbitrary Host or
	// X-Forwarded-* header supplied by the caller.
	return slices.Contains(h.origins, origin) || (h.externalURL != "" && origin == h.externalURL) || (h.oauthService != nil && origin == h.oauthService.Issuer)
}

func (h *MCPHandler) visibleTools() []mcpToolDef {
	result := make([]mcpToolDef, 0)
	for _, t := range allTools() {
		if t.Name == mcpBackendRequest && !h.allowGeneric {
			continue
		}
		if sensitiveTool(t.Name) && !h.allowSensitive {
			continue
		}
		result = append(result, t)
	}
	return result
}

func sensitiveTool(name string) bool {
	return slices.Contains([]string{"auth_get_token", "auth_regenerate_token", "auth_change_password", "secret_get", "secret_get_value"}, name)
}

// Explicit scope metadata; no fabricated arguments or business dispatch during discovery.
func toolScope(name string) string {
	if strings.HasPrefix(name, "auth_") && name != "auth_get_profile" {
		return ""
	}
	read := slices.Contains([]string{
		"auth_get_profile", "unit_list", "unit_get", "unit_logs", "unit_summary", "project_list", "project_get", "project_units", "project_budget_stats",
		"todo_list", "todo_get", "todo_group_list", "note_list", "note_search", "note_get", "note_group_list",
		"notification_list", "notification_unread_count", "schedule_list", "schedule_get", "wallet_list", "wallet_get",
		"budget_category_list", "transaction_list", "transaction_get", "budget_stats", "secret_list", "secret_get", "secret_get_value", "secret_audit_logs",
	}, name)
	if strings.HasPrefix(name, "secret_") {
		if read {
			return "secrets:read"
		}
		return "secrets:write"
	}
	if read {
		return "data:read"
	}
	return "data:write"
}

func (h *MCPHandler) sensitivePath(method, path string) bool {
	if strings.HasPrefix(path, "/api/v1/backup") || strings.HasPrefix(path, "/api/v1/oauth/") || (strings.HasPrefix(path, "/api/v1/auth/") && path != "/api/v1/auth/profile" && path != "/api/v1/auth/smtp-status" && path != "/api/v1/auth/test-email") {
		return true
	}
	if method == http.MethodGet && strings.HasPrefix(path, "/api/v1/secrets/") {
		parts := strings.Split(strings.TrimPrefix(path, "/api/v1/secrets/"), "/")
		return len(parts) == 1 || (len(parts) == 2 && parts[1] == "value")
	}
	return false
}

// A reverse proxy may connect on loopback while preserving a public Host. Only
// configured hosts bypass the SDK's localhost Host check; arbitrary hosts do not.
func (h *MCPHandler) trustedHost(host string) bool {
	origins := append([]string{h.externalURL}, h.origins...)
	if h.oauthService != nil {
		origins = append(origins, h.oauthService.Issuer)
	}
	for _, origin := range origins {
		if u, err := url.Parse(origin); err == nil && u.Host != "" && u.Host == host {
			return true
		}
	}
	return false
}

type mcpAuditKey struct{}
type mcpAudit struct {
	mu                          sync.Mutex
	method, tool, errorCategory string
	remoteAddr, requestID       string
	status                      int
	latency                     time.Duration
}

// Secret writes return SecretResponse in REST, including an existing value when
// only metadata changed. Restrict that field at the MCP output boundary too.
func redactMCPSecretValue(value any) {
	obj, ok := value.(map[string]any)
	if !ok {
		return
	}
	delete(obj, "value")
	if data, ok := obj["data"].(map[string]any); ok {
		delete(data, "value")
	}
}
