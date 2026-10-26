package handler

import (
	"encoding/json"
	"html"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestMCPInfoEscapesEditableSettings(t *testing.T) {
	name := `<script>alert("name")</script>`
	h := NewMCPHandler(MCPHandlerConfig{ServerName: name, ServerVersion: `<img src=x onerror=alert(1)>`, ExternalURL: "https://tasks.example"})
	r := gin.New()
	r.GET("/mcp", h.GetInfo)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/mcp", nil))
	body := w.Body.String()
	if strings.Contains(body, name) || strings.Contains(body, `<img src=x`) {
		t.Fatal("editable settings inserted into HTML without escaping")
	}
	match := regexp.MustCompile(`(?s)<pre id="config-json">(.*?)</pre>`).FindStringSubmatch(body)
	if len(match) != 2 {
		t.Fatal("missing client configuration")
	}
	var config struct {
		Servers map[string]any `json:"mcpServers"`
	}
	if err := json.Unmarshal([]byte(html.UnescapeString(match[1])), &config); err != nil {
		t.Fatal("copied configuration is not valid JSON:", err)
	}
	if config.Servers[name] == nil {
		t.Fatal("server name was not preserved in JSON")
	}
}
