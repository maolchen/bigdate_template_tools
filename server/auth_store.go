package server

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

var usernamePattern = regexp.MustCompile(`^[a-zA-Z0-9_-]{3,64}$`)

type authContextKey struct{}

func nowRFC3339Now() string {
	return time.Now().Format(time.RFC3339)
}

func normalizeRole(raw string) (string, error) {
	role := strings.ToLower(strings.TrimSpace(raw))
	switch role {
	case roleAdmin, roleUser:
		return role, nil
	default:
		return "", errors.New("role must be admin or user")
	}
}

func validateUsername(raw string) (string, error) {
	username := strings.TrimSpace(raw)
	if !usernamePattern.MatchString(username) {
		return "", errors.New("username only allows [a-zA-Z0-9_-], length 3~64")
	}
	return username, nil
}

func ensurePasswordStrength(password string) error {
	if len(strings.TrimSpace(password)) < 8 {
		return errors.New("password length must be >= 8")
	}
	return nil
}

func hashPassword(password string) (string, error) {
	if err := ensurePasswordStrength(password); err != nil {
		return "", err
	}
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hashed), nil
}

func verifyPassword(hashValue, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hashValue), []byte(password)) == nil
}

func generateSessionToken() (string, error) {
	raw := make([]byte, 24)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return hex.EncodeToString(raw), nil
}

func shortToken(token string) string {
	token = strings.TrimSpace(token)
	if len(token) <= 8 {
		return token
	}
	return token[:8] + "..."
}

func (s *Server) initAuthStore() error {
	if err := os.MkdirAll(s.usersRootDir, 0755); err != nil {
		return err
	}
	if err := os.MkdirAll(s.authDir, 0755); err != nil {
		return err
	}

	if err := s.loadUsersFromFile(); err != nil {
		return err
	}
	if len(s.authUsers) == 0 {
		if err := s.bootstrapDefaultAdmin(); err != nil {
			return err
		}
	}

	if err := s.loadSessionsFromFile(); err != nil {
		return err
	}
	if s.purgeExpiredSessionsLocked() {
		if err := s.saveSessionsToFileLocked(); err != nil {
			return err
		}
	}
	fmt.Printf("[Auth] store initialized users=%d sessions=%d usersFile=%s sessionsFile=%s\n",
		len(s.authUsers), len(s.authSessions), s.usersFilePath, s.sessionsFilePath)

	return nil
}

func (s *Server) loadUsersFromFile() error {
	s.authMu.Lock()
	defer s.authMu.Unlock()

	s.authUsers = make(map[string]authUserRecord)
	data, err := os.ReadFile(s.usersFilePath)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if len(data) == 0 {
		return nil
	}

	var parsed authUsersFile
	if err := json.Unmarshal(data, &parsed); err != nil {
		return err
	}
	for _, user := range parsed.Users {
		username := strings.TrimSpace(user.Username)
		if username == "" {
			continue
		}
		s.authUsers[username] = user
	}
	return nil
}

func (s *Server) saveUsersToFileLocked() error {
	users := make([]authUserRecord, 0, len(s.authUsers))
	for _, item := range s.authUsers {
		users = append(users, item)
	}
	sort.SliceStable(users, func(i, j int) bool {
		return users[i].Username < users[j].Username
	})
	raw, err := json.MarshalIndent(authUsersFile{Users: users}, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.usersFilePath, raw, 0644)
}

func (s *Server) loadSessionsFromFile() error {
	s.authMu.Lock()
	defer s.authMu.Unlock()

	s.authSessions = make(map[string]authSessionRecord)
	data, err := os.ReadFile(s.sessionsFilePath)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if len(data) == 0 {
		return nil
	}
	var parsed authSessionsFile
	if err := json.Unmarshal(data, &parsed); err != nil {
		return err
	}
	for _, session := range parsed.Sessions {
		token := strings.TrimSpace(session.Token)
		if token == "" {
			continue
		}
		s.authSessions[token] = session
	}
	return nil
}

func (s *Server) saveSessionsToFileLocked() error {
	sessions := make([]authSessionRecord, 0, len(s.authSessions))
	for _, item := range s.authSessions {
		sessions = append(sessions, item)
	}
	sort.SliceStable(sessions, func(i, j int) bool {
		return sessions[i].CreatedAt > sessions[j].CreatedAt
	})
	raw, err := json.MarshalIndent(authSessionsFile{Sessions: sessions}, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.sessionsFilePath, raw, 0644)
}

func (s *Server) bootstrapDefaultAdmin() error {
	hashed, err := hashPassword("Admin@123")
	if err != nil {
		return err
	}
	now := nowRFC3339Now()
	s.authUsers["admin"] = authUserRecord{
		Username:           "admin",
		PasswordHash:       hashed,
		Role:               roleAdmin,
		Enabled:            true,
		MustChangePassword: true,
		CreatedAt:          now,
		UpdatedAt:          now,
	}
	if err := s.saveUsersToFileLocked(); err != nil {
		return err
	}
	fmt.Printf("[Auth] bootstrap default admin user created: username=admin\n")
	return nil
}

func (s *Server) purgeExpiredSessionsLocked() bool {
	now := time.Now()
	changed := false
	for token, session := range s.authSessions {
		expiresAt, err := time.Parse(time.RFC3339, session.ExpiresAt)
		if err != nil || now.After(expiresAt) {
			delete(s.authSessions, token)
			changed = true
		}
	}
	return changed
}

func (s *Server) activeAdminSessionExistsLocked() bool {
	now := time.Now()
	for _, session := range s.authSessions {
		if session.Role != roleAdmin {
			continue
		}
		expiresAt, err := time.Parse(time.RFC3339, session.ExpiresAt)
		if err != nil {
			continue
		}
		if now.Before(expiresAt) {
			return true
		}
	}
	return false
}

// findActiveAdminSessionLocked returns any currently active admin session.
// Caller must hold s.authMu.
func (s *Server) findActiveAdminSessionLocked() (string, authSessionRecord, bool) {
	now := time.Now()
	for token, session := range s.authSessions {
		if session.Role != roleAdmin {
			continue
		}
		expiresAt, err := time.Parse(time.RFC3339, session.ExpiresAt)
		if err != nil {
			continue
		}
		if now.Before(expiresAt) {
			return token, session, true
		}
	}
	return "", authSessionRecord{}, false
}

// revokeActiveAdminSessionsByUsernameLocked removes all active admin sessions for the given username.
// Caller must hold s.authMu.
func (s *Server) revokeActiveAdminSessionsByUsernameLocked(username string) int {
	now := time.Now()
	removed := 0
	for token, session := range s.authSessions {
		if session.Role != roleAdmin || session.Username != username {
			continue
		}
		expiresAt, err := time.Parse(time.RFC3339, session.ExpiresAt)
		if err != nil || now.After(expiresAt) {
			continue
		}
		delete(s.authSessions, token)
		removed++
	}
	return removed
}

func (s *Server) login(username, password string) (authPrincipal, error) {
	username, err := validateUsername(username)
	if err != nil {
		fmt.Printf("[Auth] login invalid username raw=%q err=%v\n", username, err)
		return authPrincipal{}, err
	}

	s.authMu.Lock()
	defer s.authMu.Unlock()

	user, ok := s.authUsers[username]
	if !ok || !user.Enabled {
		fmt.Printf("[Auth] login denied username=%s reason=user_not_found_or_disabled\n", username)
		return authPrincipal{}, errors.New("invalid username or password")
	}
	if !verifyPassword(user.PasswordHash, password) {
		fmt.Printf("[Auth] login denied username=%s reason=password_mismatch\n", username)
		return authPrincipal{}, errors.New("invalid username or password")
	}
	if user.Role == roleAdmin {
		if conflictToken, conflictSession, ok := s.findActiveAdminSessionLocked(); ok {
			if conflictSession.Username == user.Username {
				// Same admin account re-login is treated as session takeover to avoid stale lock after tab/browser close.
				removed := s.revokeActiveAdminSessionsByUsernameLocked(user.Username)
				fmt.Printf("[Auth] admin relogin takeover user=%s removedSessions=%d conflictToken=%s\n", user.Username, removed, conflictToken)
			} else {
				fmt.Printf("[Auth] admin login blocked user=%s activeAdmin=%s token=%s\n", user.Username, conflictSession.Username, conflictToken)
				return authPrincipal{}, errors.New("another admin session is active")
			}
		}
	}

	token, err := generateSessionToken()
	if err != nil {
		return authPrincipal{}, err
	}
	now := time.Now()
	record := authSessionRecord{
		Token:     token,
		Username:  user.Username,
		Role:      user.Role,
		CreatedAt: now.Format(time.RFC3339),
		ExpiresAt: now.Add(defaultSessionTTL).Format(time.RFC3339),
	}
	s.authSessions[token] = record
	if err := s.saveSessionsToFileLocked(); err != nil {
		return authPrincipal{}, err
	}
	fmt.Printf("[Auth] login success username=%s role=%s token=%s expiresAt=%s\n", user.Username, user.Role, shortToken(token), record.ExpiresAt)
	return authPrincipal{
		Token:              token,
		Username:           user.Username,
		Role:               user.Role,
		MustChangePassword: user.MustChangePassword,
	}, nil
}

func (s *Server) logout(token string) error {
	token = strings.TrimSpace(token)
	if token == "" {
		return nil
	}
	s.authMu.Lock()
	defer s.authMu.Unlock()

	if _, exists := s.authSessions[token]; exists {
		delete(s.authSessions, token)
		fmt.Printf("[Auth] logout token=%s\n", shortToken(token))
		return s.saveSessionsToFileLocked()
	}
	fmt.Printf("[Auth] logout ignored token=%s reason=not_found\n", shortToken(token))
	return nil
}

func authTokenFromRequest(r *http.Request) string {
	if authHeader := strings.TrimSpace(r.Header.Get("Authorization")); authHeader != "" {
		if strings.HasPrefix(strings.ToLower(authHeader), "bearer ") {
			return strings.TrimSpace(authHeader[7:])
		}
	}
	if cookie, err := r.Cookie(authCookieName); err == nil {
		return strings.TrimSpace(cookie.Value)
	}
	return ""
}

func (s *Server) principalFromToken(token string) (authPrincipal, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return authPrincipal{}, errors.New("missing auth token")
	}

	s.authMu.Lock()
	defer s.authMu.Unlock()

	session, ok := s.authSessions[token]
	if !ok {
		return authPrincipal{}, errors.New("invalid session")
	}
	expiresAt, err := time.Parse(time.RFC3339, session.ExpiresAt)
	if err != nil || time.Now().After(expiresAt) {
		delete(s.authSessions, token)
		_ = s.saveSessionsToFileLocked()
		return authPrincipal{}, errors.New("session expired")
	}

	user, ok := s.authUsers[session.Username]
	if !ok || !user.Enabled {
		delete(s.authSessions, token)
		_ = s.saveSessionsToFileLocked()
		return authPrincipal{}, errors.New("user disabled")
	}

	return authPrincipal{
		Token:              token,
		Username:           user.Username,
		Role:               user.Role,
		MustChangePassword: user.MustChangePassword,
	}, nil
}

func (s *Server) withAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := authTokenFromRequest(r)
		principal, err := s.principalFromToken(token)
		if err != nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		ctx := context.WithValue(r.Context(), authContextKey{}, principal)
		next(w, r.WithContext(ctx))
	}
}

func (s *Server) withAdmin(next http.HandlerFunc) http.HandlerFunc {
	return s.withAuth(func(w http.ResponseWriter, r *http.Request) {
		principal, ok := authPrincipalFromContext(r.Context())
		if !ok || principal.Role != roleAdmin {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		next(w, r)
	})
}

func authPrincipalFromContext(ctx context.Context) (authPrincipal, bool) {
	value := ctx.Value(authContextKey{})
	if value == nil {
		return authPrincipal{}, false
	}
	principal, ok := value.(authPrincipal)
	return principal, ok
}

func (s *Server) listUsers() []authUserSummary {
	s.authMu.Lock()
	defer s.authMu.Unlock()
	items := make([]authUserSummary, 0, len(s.authUsers))
	for _, user := range s.authUsers {
		items = append(items, authUserSummary{
			Username:           user.Username,
			Role:               user.Role,
			Enabled:            user.Enabled,
			MustChangePassword: user.MustChangePassword,
			CreatedAt:          user.CreatedAt,
			UpdatedAt:          user.UpdatedAt,
		})
	}
	sort.SliceStable(items, func(i, j int) bool {
		return items[i].Username < items[j].Username
	})
	return items
}

func (s *Server) createUser(req authCreateUserRequest) (authUserSummary, error) {
	username, err := validateUsername(req.Username)
	if err != nil {
		return authUserSummary{}, err
	}
	role, err := normalizeRole(req.Role)
	if err != nil {
		return authUserSummary{}, err
	}
	hashed, err := hashPassword(req.Password)
	if err != nil {
		return authUserSummary{}, err
	}

	s.authMu.Lock()
	defer s.authMu.Unlock()

	if _, exists := s.authUsers[username]; exists {
		return authUserSummary{}, errors.New("username already exists")
	}
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	now := nowRFC3339Now()
	record := authUserRecord{
		Username:           username,
		PasswordHash:       hashed,
		Role:               role,
		Enabled:            enabled,
		MustChangePassword: false,
		CreatedAt:          now,
		UpdatedAt:          now,
	}
	s.authUsers[username] = record
	if err := s.saveUsersToFileLocked(); err != nil {
		return authUserSummary{}, err
	}
	fmt.Printf("[Auth] user created username=%s role=%s enabled=%t\n", record.Username, record.Role, record.Enabled)
	return authUserSummary{
		Username:           record.Username,
		Role:               record.Role,
		Enabled:            record.Enabled,
		MustChangePassword: record.MustChangePassword,
		CreatedAt:          record.CreatedAt,
		UpdatedAt:          record.UpdatedAt,
	}, nil
}

func (s *Server) enabledAdminCountLocked() int {
	count := 0
	for _, user := range s.authUsers {
		if user.Role == roleAdmin && user.Enabled {
			count++
		}
	}
	return count
}

func (s *Server) updateUser(username string, req authUpdateUserRequest) (authUserSummary, error) {
	username, err := validateUsername(username)
	if err != nil {
		return authUserSummary{}, err
	}

	s.authMu.Lock()
	defer s.authMu.Unlock()

	user, exists := s.authUsers[username]
	if !exists {
		return authUserSummary{}, errors.New("user not found")
	}

	nextRole := user.Role
	if strings.TrimSpace(req.Role) != "" {
		role, err := normalizeRole(req.Role)
		if err != nil {
			return authUserSummary{}, err
		}
		nextRole = role
	}
	nextEnabled := user.Enabled
	if req.Enabled != nil {
		nextEnabled = *req.Enabled
	}
	if (user.Role == roleAdmin && user.Enabled) && (nextRole != roleAdmin || !nextEnabled) && s.enabledAdminCountLocked() <= 1 {
		return authUserSummary{}, errors.New("at least one enabled admin is required")
	}

	user.Role = nextRole
	user.Enabled = nextEnabled
	if req.MustChangePassword != nil {
		user.MustChangePassword = *req.MustChangePassword
	}
	if strings.TrimSpace(req.ResetPassword) != "" {
		hashed, err := hashPassword(req.ResetPassword)
		if err != nil {
			return authUserSummary{}, err
		}
		user.PasswordHash = hashed
		user.MustChangePassword = true
	}
	user.UpdatedAt = nowRFC3339Now()
	s.authUsers[username] = user

	if err := s.saveUsersToFileLocked(); err != nil {
		return authUserSummary{}, err
	}
	fmt.Printf("[Auth] user updated username=%s role=%s enabled=%t mustChangePassword=%t resetPassword=%t\n",
		user.Username, user.Role, user.Enabled, user.MustChangePassword, strings.TrimSpace(req.ResetPassword) != "")
	return authUserSummary{
		Username:           user.Username,
		Role:               user.Role,
		Enabled:            user.Enabled,
		MustChangePassword: user.MustChangePassword,
		CreatedAt:          user.CreatedAt,
		UpdatedAt:          user.UpdatedAt,
	}, nil
}

func (s *Server) changePassword(principal authPrincipal, req authChangePasswordRequest) error {
	if err := ensurePasswordStrength(req.NewPassword); err != nil {
		return err
	}

	s.authMu.Lock()
	defer s.authMu.Unlock()

	user, exists := s.authUsers[principal.Username]
	if !exists {
		return errors.New("user not found")
	}
	if !verifyPassword(user.PasswordHash, req.OldPassword) {
		return errors.New("oldPassword is invalid")
	}
	hashed, err := hashPassword(req.NewPassword)
	if err != nil {
		return err
	}
	user.PasswordHash = hashed
	user.MustChangePassword = false
	user.UpdatedAt = nowRFC3339Now()
	s.authUsers[user.Username] = user
	if err := s.saveUsersToFileLocked(); err != nil {
		return err
	}
	fmt.Printf("[Auth] password changed username=%s\n", user.Username)
	return nil
}

func setAuthCookie(w http.ResponseWriter, token string, expiresAt time.Time) {
	http.SetCookie(w, &http.Cookie{
		Name:     authCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Expires:  expiresAt,
	})
}

func clearAuthCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     authCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
	})
}
