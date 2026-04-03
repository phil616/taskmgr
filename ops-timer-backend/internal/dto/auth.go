package dto

type LoginRequest struct {
	Username string `json:"username" binding:"required,min=3,max=32"`
	Password string `json:"password" binding:"required,min=6"`
}

type LoginResponse struct {
	Token string       `json:"token"`
	User  UserResponse `json:"user"`
}

type UserResponse struct {
	ID          string `json:"id"`
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
	Email       string `json:"email"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

type UpdateProfileRequest struct {
	// alphanum 限制用户名仅含字母和数字，防止特殊字符滥用
	Username    string `json:"username" binding:"omitempty,min=3,max=32,alphanum"`
	DisplayName string `json:"display_name" binding:"omitempty,min=1,max=64"`
	Email       string `json:"email" binding:"omitempty,email,max=128"`
}

type ChangePasswordRequest struct {
	OldPassword string `json:"old_password" binding:"required"`
	// 新密码至少 8 位，最多 72 位（bcrypt 截断限制）
	NewPassword string `json:"new_password" binding:"required,min=8,max=72"`
}

type TokenResponse struct {
	APIToken string `json:"api_token"`
}
