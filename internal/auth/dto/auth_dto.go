package dto

import "time"

// LoginRequest represents the login request
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
}

// RefreshTokenRequest represents the refresh token request
type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// VerifyTokenRequest represents the verify token request
type VerifyTokenRequest struct {
	Token string `json:"token" binding:"required"`
}

// SchoolInfo contiene informacion basica de una escuela
type SchoolInfo struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// LoginResponse represents the login response
type LoginResponse struct {
	AccessToken   string          `json:"access_token"`
	RefreshToken  string          `json:"refresh_token"`
	ExpiresIn     int64           `json:"expires_in"`
	TokenType     string          `json:"token_type"`
	User          *UserInfo       `json:"user"`
	Schools       []SchoolInfo    `json:"schools"`
	ActiveContext *UserContextDTO `json:"active_context,omitempty"`
}

// RefreshResponse represents the refresh token response
type RefreshResponse struct {
	AccessToken   string          `json:"access_token"`
	RefreshToken  string          `json:"refresh_token"`
	ExpiresIn     int64           `json:"expires_in"`
	TokenType     string          `json:"token_type"`
	ActiveContext *UserContextDTO `json:"active_context,omitempty"`
}

// VerifyTokenResponse represents the verify token response
type VerifyTokenResponse struct {
	Valid     bool       `json:"valid"`
	UserID    string     `json:"user_id,omitempty"`
	Email     string     `json:"email,omitempty"`
	SchoolID  string     `json:"school_id,omitempty"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
	Error     string     `json:"error,omitempty"`
}

// UserInfo represents basic user information
type UserInfo struct {
	ID        string `json:"id"`
	Email     string `json:"email"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	FullName  string `json:"full_name"`
	SchoolID  string `json:"school_id,omitempty"`
}

// UserContextDTO represents the active RBAC context
type UserContextDTO struct {
	RoleID           string   `json:"role_id"`
	RoleName         string   `json:"role_name"`
	SchoolID         string   `json:"school_id,omitempty"`
	SchoolName       string   `json:"school_name,omitempty"`
	AcademicUnitID   string   `json:"academic_unit_id,omitempty"`
	AcademicUnitName string   `json:"academic_unit_name,omitempty"`
	Permissions      []string `json:"permissions"`
}

// SwitchContextRequest represents the request to switch school context
type SwitchContextRequest struct {
	SchoolID string `json:"school_id" binding:"required,uuid"`
}

// SwitchContextResponse represents the response after switching context
type SwitchContextResponse struct {
	AccessToken  string       `json:"access_token"`
	RefreshToken string       `json:"refresh_token"`
	ExpiresIn    int64        `json:"expires_in"`
	TokenType    string       `json:"token_type"`
	Context      *ContextInfo `json:"context"`
}

// ContextInfo represents the current context info (school + role)
type ContextInfo struct {
	SchoolID string `json:"school_id"`
	Role     string `json:"role"`
	UserID   string `json:"user_id"`
	Email    string `json:"email"`
}

// AvailableContextsResponse represents available contexts for a user
type AvailableContextsResponse struct {
	Current   *UserContextDTO   `json:"current"`
	Available []*UserContextDTO `json:"available"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
	Code    string `json:"code,omitempty"`
}
