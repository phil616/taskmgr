package handler

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"ops-timer-backend/internal/model"
	"ops-timer-backend/internal/pkg/response"
	"ops-timer-backend/internal/service"
)

type MCPSettingsHandler struct{ service *service.MCPSettingsService }

func NewMCPSettingsHandler(s *service.MCPSettingsService) *MCPSettingsHandler {
	return &MCPSettingsHandler{service: s}
}

func (h *MCPSettingsHandler) Get(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	settings, err := h.service.Get(c.Request.Context())
	if err != nil {
		response.InternalError(c, "读取 MCP 设置失败")
		return
	}
	response.Success(c, settings)
}

func (h *MCPSettingsHandler) Update(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	// Require a complete document so omitted booleans cannot silently disable auth.
	var fields map[string]json.RawMessage
	decoder := json.NewDecoder(http.MaxBytesReader(c.Writer, c.Request.Body, 16<<10))
	if decoder.Decode(&fields) != nil || decoder.Decode(new(any)) != io.EOF {
		response.BadRequest(c, "设置必须为完整 JSON 对象", nil)
		return
	}
	for _, key := range []string{"enabled", "path", "external_url", "server_name", "server_version", "timeout_seconds", "api_token_enabled", "oauth_enabled", "oauth_issuer", "oauth_dynamic_registration"} {
		if raw, ok := fields[key]; !ok || string(raw) == "null" {
			response.BadRequest(c, "缺少设置字段: "+key, nil)
			return
		}
	}
	raw, _ := json.Marshal(fields)
	var settings model.MCPSettings
	if json.Unmarshal(raw, &settings) != nil {
		response.BadRequest(c, "设置字段类型错误", nil)
		return
	}
	if err := service.ValidateMCPSettings(&settings); err != nil {
		response.BadRequest(c, err.Error(), nil)
		return
	}
	saved, err := h.service.Save(c.Request.Context(), settings, c.GetString("user_id"))
	if err != nil {
		response.InternalError(c, "保存 MCP 设置失败")
		return
	}
	response.Success(c, saved)
}
