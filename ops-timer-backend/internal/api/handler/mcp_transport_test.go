package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"ops-timer-backend/internal/service"
)

const testInitialize = `{"jsonrpc":"2.0","id":0,"method":"initialize","params":{"protocolVersion":"2025-11-25","capabilities":{},"clientInfo":{"name":"regression","version":"1"}}}`

func TestMCPHTTPProtocol(t *testing.T) {
	h := NewMCPHandler(MCPHandlerConfig{})
	var calls atomic.Int32
	h.UseBackend(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"code":0,"data":[{"title":"真实待办"}]}`)
	}), nil)
	transport := mcp.NewStreamableHTTPHandler(func(r *http.Request) *mcp.Server { return h.mcpServer("test", "", false, r.Context()) }, &mcp.StreamableHTTPOptions{Stateless: true, JSONResponse: true, MaxRequestBodyBytes: 1024})
	for _, tc := range []struct {
		name, method, body, accept, content, version string
		status                                       int
		empty                                        bool
	}{
		{"initialize", "POST", testInitialize, "application/json, text/event-stream", "application/json", "", 200, false},
		{"negotiate", "POST", strings.Replace(testInitialize, "2025-11-25", "unknown", 1), "application/json, text/event-stream", "application/json", "", 200, false},
		{"initialized", "POST", `{"jsonrpc":"2.0","method":"notifications/initialized"}`, "application/json, text/event-stream", "application/json", "2025-11-25", 202, true},
		{"ping", "POST", `{"jsonrpc":"2.0","id":"ping","method":"ping"}`, "application/json, text/event-stream", "application/json", "2025-11-25", 200, false},
		{"list", "POST", `{"jsonrpc":"2.0","id":2,"method":"tools/list"}`, "application/json, text/event-stream", "application/json", "2025-11-25", 200, false},
		{"call", "POST", `{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"todo_list","arguments":{}}}`, "application/json, text/event-stream", "application/json", "2025-11-25", 200, false},
		{"no GET stream", "GET", "", "text/event-stream", "", "2025-11-25", 405, false},
		{"no DELETE session", "DELETE", "", "", "", "", 405, false},
		{"invalid JSON", "POST", "{", "application/json, text/event-stream", "application/json", "", 400, false},
		{"invalid rpc version", "POST", `{"jsonrpc":"1.0","id":1,"method":"ping"}`, "application/json, text/event-stream", "application/json", "", 400, false},
		{"invalid id", "POST", `{"jsonrpc":"2.0","id":{},"method":"ping"}`, "application/json, text/event-stream", "application/json", "", 400, false},
		{"missing initialize params", "POST", `{"jsonrpc":"2.0","id":1,"method":"initialize"}`, "application/json, text/event-stream", "application/json", "", 400, false},
		{"unknown method", "POST", `{"jsonrpc":"2.0","id":"unknown","method":"unknown"}`, "application/json, text/event-stream", "application/json", "2025-11-25", 400, false},
		{"invalid initialize fields", "POST", `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-11-25","capabilities":{}}}`, "application/json, text/event-stream", "application/json", "", 200, false},
		{"invalid tool arguments", "POST", `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"todo_get","arguments":{}}}`, "application/json, text/event-stream", "application/json", "2025-11-25", 200, false},

		{"client response", "POST", `{"jsonrpc":"2.0","id":77,"result":{}}`, "application/json, text/event-stream", "application/json", "2025-11-25", 202, true},
		{"null id", "POST", `{"jsonrpc":"2.0","id":null,"method":"ping"}`, "application/json, text/event-stream", "application/json", "2025-11-25", 400, false},
		{"batch", "POST", "[]", "application/json, text/event-stream", "application/json", "", 400, false},
		{"missing accept", "POST", testInitialize, "", "application/json", "", 400, false},
		{"missing stream accept", "POST", testInitialize, "application/json", "application/json", "", 400, false},
		{"wrong content type", "POST", testInitialize, "application/json, text/event-stream", "text/plain", "", 415, false},
		{"bad version header", "POST", testInitialize, "application/json, text/event-stream", "application/json", "2024-11-05", 400, false},
		{"oversized", "POST", strings.Repeat("x", 1025), "application/json, text/event-stream", "application/json", "", 413, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, "/mcp", strings.NewReader(tc.body))
			req.Header.Set("Accept", tc.accept)
			req.Header.Set("Content-Type", tc.content)
			if tc.version != "" {
				req.Header.Set("MCP-Protocol-Version", tc.version)
			}
			w := httptest.NewRecorder()
			transport.ServeHTTP(w, req)
			if w.Code != tc.status {
				t.Fatalf("status %d want %d: %s", w.Code, tc.status, w.Body.String())
			}
			if tc.empty && w.Body.Len() != 0 {
				t.Fatalf("notification body: %s", w.Body.String())
			}
			if !tc.empty && w.Body.Len() == 0 {
				t.Fatal("unexpected zero-byte body")
			}
			if tc.status == 200 {
				if !strings.HasPrefix(w.Header().Get("Content-Type"), "application/json") {
					t.Fatal(w.Header())
				}
				var data map[string]any
				if err := json.Unmarshal(w.Body.Bytes(), &data); err != nil {
					t.Fatal(err)
				}
				if _, ok := data["id"]; !ok {
					t.Fatal("missing response id")
				}
				if tc.name == "invalid initialize fields" {
					if data["error"] == nil {
						t.Fatalf("expected JSON-RPC error: %s", w.Body.String())
					}
				}
				if tc.name == "invalid tool arguments" && !strings.Contains(w.Body.String(), `"isError":true`) {
					t.Fatalf("missing validation failure: %s", w.Body.String())
				}
				if tc.name == "call" && !strings.Contains(w.Body.String(), "真实待办") {
					t.Fatal(w.Body.String())
				}
				if tc.name == "initialize" || tc.name == "negotiate" {
					if !strings.Contains(w.Body.String(), mcpProtocolVersion) {
						t.Fatal(w.Body.String())
					}
				}
			}
		})
	}
	if calls.Load() != 1 {
		t.Fatalf("backend calls=%d", calls.Load())
	}
}

func TestMCPToolResultsAndBounds(t *testing.T) {
	for _, tc := range []struct {
		name, body, kind string
		status           int
		isError          bool
	}{
		{"nonempty", `{"code":0,"data":[{"title":"真实数据"}]}`, "json", 200, false},
		{"empty array", `[]`, "json", 200, false}, {"empty object", `{}`, "json", 200, false}, {"null", `null`, "json", 200, false},
		{"no content", "", "no_content", 204, false}, {"empty success", "", "", 200, true},
		{"bad request", `{"message":"bad request"}`, "json", 400, true}, {"backend failure", `{"message":"failure"}`, "json", 503, true},
		{"business error", `{"code":42201,"message":"failure"}`, "json", 200, true},
		{"HTML", "<html>SPA</html>", "", 200, true}, {"truncated JSON", `{"code":0`, "", 200, true},
		{"too large", `"` + strings.Repeat("x", maxMCPResponse) + `"`, "", 200, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h := NewMCPHandler(MCPHandlerConfig{})
			h.UseBackend(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(tc.status); fmt.Fprint(w, tc.body) }), nil)
			result := h.dispatchTool(context.Background(), "test", "todo_list", nil)
			if result.IsError != tc.isError {
				t.Fatalf("error=%v result=%+v", result.IsError, result)
			}
			if len(result.Content) != 1 {
				t.Fatal("missing text")
			}
			if tc.kind != "" {
				var text map[string]any
				if err := json.Unmarshal([]byte(result.Content[0].(*mcp.TextContent).Text), &text); err != nil {
					t.Fatal(err)
				}
				encoded, _ := json.Marshal(result.StructuredContent)
				var structured map[string]any
				json.Unmarshal(encoded, &structured)
				if !reflect.DeepEqual(text, structured) || text["kind"] != tc.kind {
					t.Fatal("content/structuredContent diverged")
				}
			}
		})
	}
	h := NewMCPHandler(MCPHandlerConfig{})
	h.httpClient.Timeout = 20 * time.Millisecond
	h.UseBackend(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { <-r.Context().Done() }), nil)
	if !h.dispatchTool(context.Background(), "test", "todo_list", nil).IsError {
		t.Fatal("deadline swallowed")
	}
}

func TestMCPRegistryPermissionsAndSchemas(t *testing.T) {
	names := map[string]bool{}
	for _, tool := range allTools() {
		if names[tool.Name] {
			t.Fatal("duplicate tool", tool.Name)
		}
		names[tool.Name] = true
		if tool.Name == mcpBackendRequest {
			continue
		}
		call, err := toolToHTTP(tool.Name, map[string]any{"id": "resource", "resource_id": "resource"})
		if err != nil {
			t.Fatal(tool.Name, err)
		}
		if got, want := toolScope(tool.Name), service.RequiredOAuthScope(call.method, call.path); got != want {
			t.Fatalf("%s scope=%s want=%s", tool.Name, got, want)
		}
	}
	h := NewMCPHandler(MCPHandlerConfig{AllowGeneric: true})
	var calls atomic.Int32
	h.UseBackend(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { calls.Add(1); fmt.Fprint(w, `{}`) }), nil)
	for _, path := range []string{"/api/v1/auth/token", "/api/v1/auth/token/regenerate", "/api/v1/backup/export", "/api/v1/secrets/key", "/api/v1/secrets/key/value", "/api/v1/%73ecrets/key", "//evil.example/api/v1/todos", "/api/v1/todos/../../auth/token"} {
		if !h.dispatchTool(context.Background(), "test", mcpBackendRequest, map[string]any{"method": "GET", "path": path}).IsError {
			t.Fatal("accepted sensitive/invalid path", path)
		}
	}
	if calls.Load() != 0 {
		t.Fatal("blocked request reached backend")
	}
	// Registration resolves and validates every declared input/output schema.
	_ = h.mcpServer("test", "", false, context.Background())
	for _, id := range []string{"../secrets", "x?query=1", "x/y", "x%2Fy", "x#fragment"} {
		if _, err := toolToHTTP("todo_get", map[string]any{"id": id}); err == nil {
			t.Fatal("unsafe id", id)
		}
	}
}

func TestMCPOrigin(t *testing.T) {
	h := NewMCPHandler(MCPHandlerConfig{ExternalURL: "https://tasks.example", Origins: []string{"https://ui.example"}})
	for _, origin := range []string{"", "https://tasks.example", "https://ui.example", "https://evil.example", "null", "https://tasks.example/", "https://tasks.example#", "https://tasks.example?"} {
		req := httptest.NewRequest("POST", "https://tasks.example/mcp", nil)
		if origin != "" {
			req.Header.Set("Origin", origin)
		}
		allowed := origin == "" || origin == "https://tasks.example" || origin == "https://ui.example"
		if h.validOrigin(req) != allowed {
			t.Fatal("origin policy", origin)
		}
	}
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("POST", "/mcp", nil)
	c.Request.Header.Set("Origin", "https://evil.example")
	h.Handle(c)
	if c.Writer.Status() != 403 {
		t.Fatal("invalid origin not rejected before auth")
	}
}

func TestMCPSecretWriteCannotReadExistingValue(t *testing.T) {
	for _, allowed := range []bool{false, true} {
		h := NewMCPHandler(MCPHandlerConfig{AllowSensitive: allowed})
		h.UseBackend(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprint(w, `{"code":0,"data":{"id":"key","name":"renamed","value":"existing-secret-value"}}`)
		}), nil)
		result := h.dispatchTool(context.Background(), "test", "secret_update", map[string]any{"id": "key", "name": "renamed"})
		if result.IsError {
			t.Fatal(result)
		}
		raw, _ := json.Marshal(result)
		if strings.Contains(string(raw), "existing-secret-value") != allowed {
			t.Fatal("secret output policy bypass", allowed)
		}
	}
}
