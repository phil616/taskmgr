package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
	"ops-timer-backend/internal/api/handler"
	"ops-timer-backend/internal/api/router"
	"ops-timer-backend/internal/config"
	"ops-timer-backend/internal/dto"
	"ops-timer-backend/internal/pkg/auth"
	"ops-timer-backend/internal/repository"
	"ops-timer-backend/internal/service"
)

type mcpTestFixture struct {
	server   *httptest.Server
	settings *service.MCPSettingsService
	token    string
	logs     *observer.ObservedLogs
}

func newMCPFixture(t *testing.T) *mcpTestFixture {
	t.Helper()
	db := testPostgres(t)
	jwt := auth.NewJWTManager(strings.Repeat("s", 32), 24, nil)
	users := repository.NewUserRepository(db)
	authSvc := service.NewAuthService(users, repository.NewBrowserSessionRepository(db), repository.NewLoginAttemptRepository(db), jwt, &config.AuthConfig{LoginLockAttempts: 5, LoginLockMinutes: 15})
	if err := authSvc.EnsureAdminExists("mcp-test", "MCPTestPassword123"); err != nil {
		t.Fatal(err)
	}
	user, err := users.FindByUsername("mcp-test")
	if err != nil {
		t.Fatal(err)
	}
	settings := service.NewMCPSettingsService(db, authSvc, strings.Repeat("s", 32))
	if err := settings.Initialize(context.Background(), &config.Config{MCP: config.MCPConfig{Enabled: true, Path: "/mcp", ServerName: "mcp-test", ServerVersion: "1", TimeoutSeconds: 5}}); err != nil {
		t.Fatal(err)
	}
	todos := service.NewTodoService(repository.NewTodoRepository(db), repository.NewTodoGroupRepository(db))
	if _, err := todos.Create(&dto.CreateTodoRequest{Title: "可被客户端读取的待办"}); err != nil {
		t.Fatal(err)
	}
	core, logs := observer.New(zap.InfoLevel)
	engine := router.NewRouter(&router.RouterConfig{
		AuthHandler: handler.NewAuthHandler(authSvc, nil), AuthService: authSvc, JWTManager: jwt,
		TodoHandler: handler.NewTodoHandler(todos), MCPSettings: settings,
		MCPHandler: handler.NewMCPHandler(handler.MCPHandlerConfig{AuthService: authSvc}), Logger: zap.New(core),
	}).Setup()
	server := httptest.NewServer(engine)
	t.Cleanup(server.Close)
	return &mcpTestFixture{server: server, settings: settings, token: user.APIToken, logs: logs}
}

type mcpAuthTransport struct {
	base  http.RoundTripper
	token string
}

func (t mcpAuthTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	r = r.Clone(r.Context())
	r.Header.Set("X-API-Token", t.token)
	return t.base.RoundTrip(r)
}

func exerciseMCPClient(t *testing.T, endpoint, token string, base *http.Client) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	transport := base.Transport
	if transport == nil {
		transport = http.DefaultTransport
	}
	clientHTTP := &http.Client{Transport: mcpAuthTransport{transport, token}, Timeout: 10 * time.Second}
	// Both explicitly selected legacy handshake versions and SDK-default discovery
	// must negotiate a supported version. Repeat to catch empty results/lifetime bugs.
	for _, version := range []string{"2025-11-25", "2025-06-18", "2025-03-26", ""} {
		client := mcp.NewClient(&mcp.Implementation{Name: "regression-client", Version: "1"}, nil)
		session, err := client.Connect(ctx, &mcp.StreamableClientTransport{Endpoint: endpoint, HTTPClient: clientHTTP}, &mcp.ClientSessionOptions{ProtocolVersion: version})
		if err != nil {
			t.Fatalf("connect version=%q: %v", version, err)
		}
		for n := 0; n < 5; n++ {
			list, err := session.ListTools(ctx, &mcp.ListToolsParams{})
			if err != nil {
				t.Fatal(err)
			}
			found := false
			for _, tool := range list.Tools {
				if tool.Name == "todo_list" {
					found = true
				}
				if tool.Name == "backend_request" || tool.Name == "auth_get_token" || tool.Name == "secret_get_value" {
					t.Fatal("disabled tool advertised", tool.Name)
				}
			}
			if !found {
				t.Fatal("missing todo_list")
			}
			result, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "todo_list", Arguments: map[string]any{}})
			if err != nil || result.IsError {
				t.Fatalf("call: %v %+v", err, result)
			}
			if len(result.Content) != 1 || !strings.Contains(result.Content[0].(*mcp.TextContent).Text, "可被客户端读取的待办") {
				t.Fatalf("missing business data: %+v", result)
			}
			raw, _ := json.Marshal(result.StructuredContent)
			if !strings.Contains(string(raw), "可被客户端读取的待办") {
				t.Fatal("missing structured data")
			}
		}
		if err := session.Close(); err != nil {
			t.Fatal(err)
		}
	}
}

func TestMCPRealSDKClient(t *testing.T) {
	fixture := newMCPFixture(t)
	exerciseMCPClient(t, fixture.server.URL+"/mcp", fixture.token, fixture.server.Client())
	entries := fixture.logs.FilterMessage("request").All()
	found := false
	for _, entry := range entries {
		fields := entry.ContextMap()
		if fields["mcp_method"] == "tools/call" {
			found = true
			if fields["response_bytes"].(int64) <= 0 || fields["content_type"] != "application/json" || fields["mcp_tool"] != "todo_list" || fields["mcp_backend_status"] != int64(200) {
				t.Fatalf("incomplete telemetry: %v", fields)
			}
		}
		raw, _ := json.Marshal(fields)
		if strings.Contains(string(raw), fixture.token) || strings.Contains(string(raw), "可被客户端读取的待办") {
			t.Fatal("sensitive information in logs")
		}
	}
	if !found {
		t.Fatal("no MCP method telemetry")
	}
}

func TestMCPNginxHTTPS(t *testing.T) {
	image := os.Getenv("TASK_MANAGER_TEST_NGINX_IMAGE")
	if image == "" {
		t.Skip("set TASK_MANAGER_TEST_NGINX_IMAGE=nginx:alpine to run real Nginx HTTPS tests")
	}
	fixture := newMCPFixture(t)
	endpoint, client := startMCPNginx(t, fixture.server.URL, image)
	exerciseMCPClient(t, endpoint+"/mcp", fixture.token, client)
	settings, err := fixture.settings.Get(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	settings.Path = "/agents/mcp"
	if _, err := fixture.settings.Save(context.Background(), *settings, "test"); err != nil {
		t.Fatal(err)
	}
	exerciseMCPClient(t, endpoint+settings.Path, fixture.token, client)
	for _, tc := range []struct {
		method, path, accept, body string
		status                     int
		empty                      bool
	}{
		{"GET", settings.Path, "text/event-stream", "", 405, false},
		{"GET", settings.Path + "/info", "text/html", "", 200, false},
		{"GET", "/settings", "text/html", "", 200, false},
		{"GET", "/missing-mcp", "application/json, text/event-stream", "", 404, false},
		{"POST", settings.Path, "application/json, text/event-stream", `{"jsonrpc":"2.0","method":"notifications/initialized"}`, 202, true},
	} {
		req, _ := http.NewRequest(tc.method, endpoint+tc.path, strings.NewReader(tc.body))
		req.Header.Set("Accept", tc.accept)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-API-Token", fixture.token)
		resp, err := client.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if resp.StatusCode != tc.status || (tc.empty && len(body) != 0) {
			t.Fatalf("%s %s: status=%d bytes=%d body=%s", tc.method, tc.path, resp.StatusCode, len(body), body)
		}
	}
}

func startMCPNginx(t *testing.T, backend, image string) (string, *http.Client) {
	t.Helper()
	dir := t.TempDir()
	if err := os.Chmod(dir, 0755); err != nil {
		t.Fatal(err)
	}
	// Reuse httptest's trusted test certificate, keeping private material in temp files.
	certServer := httptest.NewTLSServer(http.NotFoundHandler())
	cert := certServer.TLS.Certificates[0]
	certServer.Close()
	key, err := x509.MarshalPKCS8PrivateKey(cert.PrivateKey)
	if err != nil {
		t.Fatal(err)
	}
	write := func(name string, data []byte) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, name), data, 0644); err != nil {
			t.Fatal(err)
		}
	}
	write("cert.pem", pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: cert.Certificate[0]}))
	write("key.pem", pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: key}))
	write("index.html", []byte("<html>SPA navigation</html>"))
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	listener.Close()
	raw, err := os.ReadFile("../../../ops-timer-frontend/nginx.conf")
	if err != nil {
		t.Fatal(err)
	}
	conf := strings.ReplaceAll(string(raw), "http://backend:8080", backend)
	conf = strings.Replace(conf, "listen 80;", fmt.Sprintf("listen 127.0.0.1:%d ssl;\n ssl_certificate /etc/nginx/test/cert.pem;\n ssl_certificate_key /etc/nginx/test/key.pem;", port), 1)
	write("default.conf", []byte(conf))
	cmd := exec.Command("docker", "run", "--rm", "-d", "--network", "host", "-v", filepath.Join(dir, "default.conf")+":/etc/nginx/conf.d/default.conf:ro", "-v", dir+":/etc/nginx/test:ro", "-v", dir+":/usr/share/nginx/html:ro", image)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("start nginx: %v %s", err, out)
	}
	container := strings.TrimSpace(string(out))
	t.Cleanup(func() {
		out, err := exec.Command("docker", "rm", "-f", container).CombinedOutput()
		if err != nil {
			t.Errorf("remove nginx: %v %s", err, out)
		}
	})
	roots := x509.NewCertPool()
	parsed, err := x509.ParseCertificate(cert.Certificate[0])
	if err != nil {
		t.Fatal(err)
	}
	roots.AddCert(parsed)
	client := &http.Client{Timeout: 10 * time.Second, Transport: &http.Transport{TLSClientConfig: &tls.Config{RootCAs: roots}}}
	t.Cleanup(client.CloseIdleConnections)
	endpoint := fmt.Sprintf("https://127.0.0.1:%d", port)
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		resp, err := client.Get(endpoint + "/health")
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == 200 {
				return endpoint, client
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
	logs, _ := exec.Command("docker", "logs", container).CombinedOutput()
	t.Fatalf("nginx readiness failed: %s", logs)
	return "", nil
}
