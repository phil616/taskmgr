package handler

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"testing"
)

//go:generate go test -run TestMCPToolDocumentation -update-mcp-docs
var updateMCPDocs = flag.Bool("update-mcp-docs", false, "regenerate docs/mcp-tools.md from the MCP registry")

func TestMCPToolDocumentation(t *testing.T) {
	tools := allTools()
	h := NewMCPHandler(MCPHandlerConfig{})
	var out strings.Builder
	fmt.Fprintf(&out, "# MCP 工具注册表\n\n由 `go generate ./internal/api/handler` 生成，测试自动校验。共 %d 个注册工具，默认启用 %d 个；OAuth 工具列表还受 scope 限制，通用代理不向 OAuth 客户端开放。\n\n| 工具 | OAuth scope | 默认启用 | 说明 |\n|---|---|---|---|\n", len(tools), len(h.visibleTools()))
	for _, tool := range tools {
		scope := toolScope(tool.Name)
		if scope == "" || tool.Name == mcpBackendRequest {
			scope = "不委托"
		}
		enabled := "是"
		if tool.Name == mcpBackendRequest || sensitiveTool(tool.Name) {
			enabled = "否"
		}
		fmt.Fprintf(&out, "| `%s` | %s | %s | %s |\n", tool.Name, scope, enabled, strings.ReplaceAll(tool.Description, "|", "\\|"))
	}
	const path = "../../../docs/mcp-tools.md"
	if *updateMCPDocs {
		if err := os.WriteFile(path, []byte(out.String()), 0644); err != nil {
			t.Fatal(err)
		}
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != out.String() {
		t.Fatal("MCP 工具文档与注册表不一致，运行 go generate ./internal/api/handler")
	}
}
