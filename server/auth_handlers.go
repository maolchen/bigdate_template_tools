package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// handleAuthLogin validates username/password and creates session cookie.
func (s *Server) handleAuthLogin(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	fmt.Printf("[Auth] login api method=%s\n", r.Method)
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req authLoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	principal, err := s.login(req.Username, req.Password)
	if err != nil {
		fmt.Printf("[Auth] login api failed username=%s err=%v\n", strings.TrimSpace(req.Username), err)
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	activeTemplateID, syncLog, err := s.ensureUserConfigForLogin(principal.Username)
	if err != nil {
		_ = s.logout(principal.Token)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	fmt.Printf("[Auth] login success user=%s role=%s activeTemplate=%s\n", principal.Username, principal.Role, activeTemplateID)

	setAuthCookie(w, principal.Token, time.Now().Add(defaultSessionTTL))
	json.NewEncoder(w).Encode(authLoginResponse{
		Success: true,
		User:    principal,
		Sync:    syncLog,
	})
}

// handleAuthLogout clears current auth session.
func (s *Server) handleAuthLogout(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	fmt.Printf("[Auth] logout api method=%s\n", r.Method)
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	token := authTokenFromRequest(r)
	if err := s.logout(token); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	clearAuthCookie(w)
	json.NewEncoder(w).Encode(map[string]any{
		"success": true,
	})
}

// handleAuthMe returns current authenticated principal and active template info.
func (s *Server) handleAuthMe(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	principal, ok := authPrincipalFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	fmt.Printf("[Auth] me user=%s role=%s\n", principal.Username, principal.Role)
	json.NewEncoder(w).Encode(map[string]any{
		"user":             principal,
		"activeTemplateId": userMainTemplateID,
	})
}

// handleAuthChangePassword updates current user's password.
func (s *Server) handleAuthChangePassword(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	fmt.Printf("[Auth] change-password api method=%s\n", r.Method)
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	principal, ok := authPrincipalFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var req authChangePasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := s.changePassword(principal, req); err != nil {
		fmt.Printf("[Auth] change-password failed user=%s err=%v\n", principal.Username, err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	json.NewEncoder(w).Encode(map[string]any{
		"success": true,
		"message": "密码修改成功",
	})
}

// handleUsersCollection manages /api/users list/create operations.
func (s *Server) handleUsersCollection(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	fmt.Printf("[Auth] users collection method=%s\n", r.Method)
	switch r.Method {
	case http.MethodGet:
		json.NewEncoder(w).Encode(s.listUsers())
	case http.MethodPost:
		var req authCreateUserRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		user, err := s.createUser(req)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if _, _, err := s.ensureUserConfigForLogin(user.Username); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(user)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleUsersDetail manages /api/users/:username update operation.
func (s *Server) handleUsersDetail(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	fmt.Printf("[Auth] users detail method=%s path=%s\n", r.Method, r.URL.Path)
	if r.Method != http.MethodPut {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	username := strings.TrimSpace(strings.TrimPrefix(r.URL.Path, "/api/users/"))
	if username == "" {
		http.Error(w, "username is required", http.StatusBadRequest)
		return
	}

	var req authUpdateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	user, err := s.updateUser(username, req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	json.NewEncoder(w).Encode(user)
}
