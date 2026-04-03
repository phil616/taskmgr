package handler

import (
	"ops-timer-backend/internal/dto"
	"ops-timer-backend/internal/pkg/email"
	"ops-timer-backend/internal/pkg/response"
	"ops-timer-backend/internal/service"

	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	authService  *service.AuthService
	emailService *email.Service
}

func NewAuthHandler(authService *service.AuthService, emailSvc *email.Service) *AuthHandler {
	return &AuthHandler{authService: authService, emailService: emailSvc}
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req dto.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数校验失败", nil)
		return
	}

	resp, err := h.authService.Login(&req)
	if err != nil {
		switch err {
		case service.ErrAccountLocked:
			response.TooManyRequests(c, err.Error())
		default:
			response.Unauthorized(c, response.CodeCredentialError, err.Error())
		}
		return
	}

	response.Success(c, resp)
}

func (h *AuthHandler) Logout(c *gin.Context) {
	token, exists := c.Get("token")
	if exists {
		h.authService.Logout(token.(string))
	}
	response.Success(c, nil)
}

func (h *AuthHandler) GetProfile(c *gin.Context) {
	userID := c.GetString("user_id")
	profile, err := h.authService.GetProfile(userID)
	if err != nil {
		response.NotFound(c, err.Error())
		return
	}
	response.Success(c, profile)
}

func (h *AuthHandler) UpdateProfile(c *gin.Context) {
	var req dto.UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数校验失败", nil)
		return
	}

	userID := c.GetString("user_id")
	profile, err := h.authService.UpdateProfile(userID, &req)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.Success(c, profile)
}

func (h *AuthHandler) ChangePassword(c *gin.Context) {
	var req dto.ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数校验失败", nil)
		return
	}

	userID := c.GetString("user_id")
	if err := h.authService.ChangePassword(userID, &req); err != nil {
		if err == service.ErrOldPasswordWrong {
			response.Unauthorized(c, response.CodeCredentialError, err.Error())
		} else {
			response.InternalError(c, err.Error())
		}
		return
	}

	// 密码修改成功后撤销当前 JWT，强制重新登录
	if token, exists := c.Get("token"); exists {
		h.authService.Logout(token.(string))
	}
	response.Success(c, nil)
}

func (h *AuthHandler) GetToken(c *gin.Context) {
	userID := c.GetString("user_id")
	token, err := h.authService.GetAPIToken(userID)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.Success(c, dto.TokenResponse{APIToken: token})
}

func (h *AuthHandler) RegenerateToken(c *gin.Context) {
	userID := c.GetString("user_id")
	token, err := h.authService.RegenerateAPIToken(userID)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.Success(c, dto.TokenResponse{APIToken: token})
}

// TestEmail 发送测试邮件到当前用户的邮箱
func (h *AuthHandler) TestEmail(c *gin.Context) {
	if !h.emailService.Enabled() {
		response.BusinessError(c, "SMTP 邮件通知未配置，请通过 TASK_MANAGER_SMTP_* 环境变量配置")
		return
	}
	userID := c.GetString("user_id")
	profile, err := h.authService.GetProfile(userID)
	if err != nil || profile.Email == "" {
		response.BusinessError(c, "请先在个人设置中填写通知邮箱")
		return
	}
	if err := h.emailService.SendTest(profile.Email); err != nil {
		response.BusinessError(c, "邮件发送失败: "+err.Error())
		return
	}
	response.Success(c, gin.H{"message": "测试邮件已发送至 " + profile.Email})
}

// SMTPStatus 返回当前 SMTP 是否已配置
func (h *AuthHandler) SMTPStatus(c *gin.Context) {
	response.Success(c, gin.H{"enabled": h.emailService.Enabled()})
}
