package server

import "time"

const (
	roleAdmin = "admin"
	roleUser  = "user"
)

const (
	authCookieName = "config_generator_token"
)

var (
	defaultSessionTTL = 24 * time.Hour
)

type authUserRecord struct {
	Username           string `json:"username"`
	PasswordHash       string `json:"passwordHash"`
	Role               string `json:"role"`
	Enabled            bool   `json:"enabled"`
	MustChangePassword bool   `json:"mustChangePassword"`
	CreatedAt          string `json:"createdAt"`
	UpdatedAt          string `json:"updatedAt"`
}

type authUsersFile struct {
	Users []authUserRecord `json:"users"`
}

type authSessionRecord struct {
	Token     string `json:"token"`
	Username  string `json:"username"`
	Role      string `json:"role"`
	CreatedAt string `json:"createdAt"`
	ExpiresAt string `json:"expiresAt"`
}

type authSessionsFile struct {
	Sessions []authSessionRecord `json:"sessions"`
}

type authPrincipal struct {
	Token              string `json:"-"`
	Username           string `json:"username"`
	Role               string `json:"role"`
	MustChangePassword bool   `json:"mustChangePassword"`
}

type authLoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type authLoginResponse struct {
	Success bool              `json:"success"`
	User    authPrincipal     `json:"user"`
	Sync    userConfigSyncLog `json:"sync"`
}

type authChangePasswordRequest struct {
	OldPassword string `json:"oldPassword"`
	NewPassword string `json:"newPassword"`
}

type authCreateUserRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Role     string `json:"role"`
	Enabled  *bool  `json:"enabled"`
}

type authUpdateUserRequest struct {
	Role               string `json:"role"`
	Enabled            *bool  `json:"enabled"`
	ResetPassword      string `json:"resetPassword"`
	MustChangePassword *bool  `json:"mustChangePassword"`
}

type authUserSummary struct {
	Username           string `json:"username"`
	Role               string `json:"role"`
	Enabled            bool   `json:"enabled"`
	MustChangePassword bool   `json:"mustChangePassword"`
	CreatedAt          string `json:"createdAt"`
	UpdatedAt          string `json:"updatedAt"`
}
