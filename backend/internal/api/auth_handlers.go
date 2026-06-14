package api

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/morphatrix/campmenu/internal/auth"
	"github.com/morphatrix/campmenu/internal/models"
	"github.com/morphatrix/campmenu/internal/settings"
	"gorm.io/gorm"
)

// requestIsHTTPS reports whether the original client request used HTTPS,
// honoring the X-Forwarded-Proto header set by the ingress / nginx proxy.
func requestIsHTTPS(r *http.Request) bool {
	if r.TLS != nil {
		return true
	}
	return strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https")
}

func (s *Server) setAuthCookie(w http.ResponseWriter, r *http.Request, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     authCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		// Secure must match the actual scheme: a Secure cookie is dropped over
		// plain HTTP (e.g. a local port-forward), which would silently break login.
		Secure:   requestIsHTTPS(r),
		SameSite: http.SameSiteLaxMode,
		Expires:  time.Now().Add(s.Cfg.JWTExpiry),
	})
}

// ---- invitations ----

type createInviteReq struct {
	Email         string      `json:"email"`
	Role          models.Role `json:"role"`
	MaxUses       int         `json:"maxUses"`       // 0 = unlimited (reusable by many)
	ExpiresInDays int         `json:"expiresInDays"` // 0 = never expires
}

func (s *Server) handleCreateInvite(w http.ResponseWriter, r *http.Request) {
	var req createInviteReq
	if err := decode(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "corps de requête invalide")
		return
	}
	role := models.RoleUser
	switch req.Role {
	case models.RoleAdmin:
		if roleFrom(r) == models.RoleAdmin { // only admins can mint admin invites
			role = models.RoleAdmin
		}
	case models.RoleCollaborator:
		role = models.RoleCollaborator
	}
	var expPtr *time.Time
	if req.ExpiresInDays > 0 {
		exp := time.Now().Add(time.Duration(req.ExpiresInDays) * 24 * time.Hour)
		expPtr = &exp
	}
	if req.MaxUses < 0 {
		req.MaxUses = 0
	}
	inv := models.Invite{
		Code:      auth.RandomToken(16),
		Email:     strings.ToLower(strings.TrimSpace(req.Email)),
		Role:      role,
		CreatedBy: userIDFrom(r),
		MaxUses:   req.MaxUses,
		ExpiresAt: expPtr,
	}
	if err := s.DB.Create(&inv).Error; err != nil {
		writeError(w, http.StatusInternalServerError, "création de l'invitation impossible")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{
		"invite": inv,
		"link":   s.Settings.Get(settings.KeyAppURL) + "/invite/" + inv.Code,
	})
}

func (s *Server) handleListInvites(w http.ResponseWriter, r *http.Request) {
	var invites []models.Invite
	s.DB.Order("created_at desc").Find(&invites)
	writeJSON(w, http.StatusOK, invites)
}

// handleRevokeInvite invalidates an invite immediately (staff).
func (s *Server) handleRevokeInvite(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "id invalide")
		return
	}
	s.DB.Model(&models.Invite{}).Where("id = ?", id).Update("revoked", true)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// handleGetInvite is public: validates a code before showing the register form.
func (s *Server) handleGetInvite(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")
	var inv models.Invite
	if err := s.DB.Where("code = ?", code).First(&inv).Error; err != nil {
		writeError(w, http.StatusNotFound, "invitation introuvable")
		return
	}
	if inv.Exhausted() {
		writeError(w, http.StatusGone, "invitation expirée ou épuisée")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"email": inv.Email,
		"valid": true,
	})
}

// ---- register / login / confirm ----

type registerReq struct {
	Code      string `json:"code"`
	Email     string `json:"email"`
	Password  string `json:"password"`
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
}

func (s *Server) handleRegister(w http.ResponseWriter, r *http.Request) {
	var req registerReq
	if err := decode(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "corps de requête invalide")
		return
	}
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	if req.Email == "" || len(req.Password) < 8 {
		writeError(w, http.StatusBadRequest, "email requis et mot de passe d'au moins 8 caractères")
		return
	}

	var inv models.Invite
	if err := s.DB.Where("code = ?", req.Code).First(&inv).Error; err != nil {
		writeError(w, http.StatusForbidden, "invitation invalide")
		return
	}
	if inv.Exhausted() {
		writeError(w, http.StatusForbidden, "invitation invalide, expirée ou épuisée")
		return
	}

	// Existing account? The invite then just confirms access; don't recreate.
	var existing models.User
	if err := s.DB.Where("email = ?", req.Email).First(&existing).Error; err == nil {
		writeError(w, http.StatusConflict, "un compte existe déjà pour cet email, connectez-vous")
		return
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		writeError(w, http.StatusInternalServerError, "erreur serveur")
		return
	}

	hash, err := auth.HashPassword(req.Password, s.Cfg.BcryptCost)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "erreur serveur")
		return
	}
	confirmRequired := s.Settings.Bool(settings.KeyEmailConfirmRequired)
	user := models.User{
		Email:             req.Email,
		PasswordHash:      hash,
		Role:              inv.Role,
		FirstName:         req.FirstName,
		LastName:          req.LastName,
		Theme:             s.Settings.Get(settings.KeyDefaultTheme),
		ColorPalette:      s.Settings.Get(settings.KeyDefaultPalette),
		Language:          "fr",
		ConfirmationToken: auth.RandomToken(20),
		EmailConfirmed:    !confirmRequired,
	}

	err = s.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&user).Error; err != nil {
			return err
		}
		updates := map[string]any{"use_count": gorm.Expr("use_count + 1")}
		if inv.UsedAt == nil {
			now := time.Now()
			updates["used_at"] = now
		}
		return tx.Model(&inv).Updates(updates).Error
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "création du compte impossible")
		return
	}

	if confirmRequired {
		_ = s.Mailer.SendConfirmation(user.Email, user.ConfirmationToken)
		writeJSON(w, http.StatusCreated, map[string]any{"emailConfirmRequired": true})
		return
	}
	token, _ := auth.GenerateJWT(s.Cfg.JWTSecret, s.Cfg.JWTExpiry, user.ID, user.Role)
	s.setAuthCookie(w, r, token)
	writeJSON(w, http.StatusCreated, map[string]any{"user": user})
}

func (s *Server) handleConfirm(w http.ResponseWriter, r *http.Request) {
	token := chi.URLParam(r, "token")
	var user models.User
	if err := s.DB.Where("confirmation_token = ?", token).First(&user).Error; err != nil {
		writeError(w, http.StatusNotFound, "jeton de confirmation invalide")
		return
	}
	s.DB.Model(&user).Updates(map[string]any{"email_confirmed": true, "confirmation_token": ""})
	writeJSON(w, http.StatusOK, map[string]any{"confirmed": true})
}

type loginReq struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req loginReq
	if err := decode(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "corps de requête invalide")
		return
	}
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	var user models.User
	if err := s.DB.Where("email = ?", req.Email).First(&user).Error; err != nil {
		writeError(w, http.StatusUnauthorized, "identifiants invalides")
		return
	}
	if !auth.CheckPassword(user.PasswordHash, req.Password) {
		writeError(w, http.StatusUnauthorized, "identifiants invalides")
		return
	}
	if s.Settings.Bool(settings.KeyEmailConfirmRequired) && !user.EmailConfirmed {
		writeError(w, http.StatusForbidden, "email non confirmé")
		return
	}
	token, _ := auth.GenerateJWT(s.Cfg.JWTSecret, s.Cfg.JWTExpiry, user.ID, user.Role)
	s.setAuthCookie(w, r, token)
	writeJSON(w, http.StatusOK, map[string]any{"user": user})
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name: authCookieName, Value: "", Path: "/", HttpOnly: true,
		Expires: time.Unix(0, 0), MaxAge: -1,
	})
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// ---- current user / profile ----

func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	var user models.User
	if err := s.DB.First(&user, "id = ?", userIDFrom(r)).Error; err != nil {
		writeError(w, http.StatusNotFound, "utilisateur introuvable")
		return
	}
	user.Impersonating = impFrom(r) != uuid.Nil
	writeJSON(w, http.StatusOK, user)
}

// handleImpersonate (admin) starts a test session "as" another user, keeping
// the admin id so they can switch back.
func (s *Server) handleImpersonate(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "id invalide")
		return
	}
	var target models.User
	if err := s.DB.First(&target, "id = ?", id).Error; err != nil {
		writeError(w, http.StatusNotFound, "utilisateur introuvable")
		return
	}
	admin := userIDFrom(r)
	token, err := auth.GenerateImpersonationJWT(s.Cfg.JWTSecret, s.Cfg.JWTExpiry, target.ID, target.Role, admin)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "erreur serveur")
		return
	}
	s.setAuthCookie(w, r, token)
	target.Impersonating = true
	writeJSON(w, http.StatusOK, target)
}

// handleStopImpersonate ends a test session and restores the admin account.
func (s *Server) handleStopImpersonate(w http.ResponseWriter, r *http.Request) {
	imp := impFrom(r)
	if imp == uuid.Nil {
		writeError(w, http.StatusBadRequest, "pas en mode test")
		return
	}
	var admin models.User
	if err := s.DB.First(&admin, "id = ?", imp).Error; err != nil {
		writeError(w, http.StatusNotFound, "compte administrateur introuvable")
		return
	}
	if admin.Role != models.RoleAdmin {
		writeError(w, http.StatusForbidden, "restauration non autorisée")
		return
	}
	token, _ := auth.GenerateJWT(s.Cfg.JWTSecret, s.Cfg.JWTExpiry, admin.ID, admin.Role)
	s.setAuthCookie(w, r, token)
	writeJSON(w, http.StatusOK, admin)
}

type updateProfileReq struct {
	FirstName      *string    `json:"firstName"`
	LastName       *string    `json:"lastName"`
	Nickname       *string    `json:"nickname"`
	IBAN           *string    `json:"iban"`
	BirthDate      *time.Time `json:"birthDate"`
	ShoeSize       *float64   `json:"shoeSize"`
	Weight         *float64   `json:"weight"`
	PhotoURL       *string    `json:"photoUrl"`
	Theme          *string    `json:"theme"`
	ColorPalette   *string    `json:"colorPalette"`
	ColorblindMode *bool      `json:"colorblindMode"`
	Language       *string    `json:"language"`
}

// handleUpdateMe lets a user edit only their own profile/preferences.
func (s *Server) handleUpdateMe(w http.ResponseWriter, r *http.Request) {
	var req updateProfileReq
	if err := decode(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "corps de requête invalide")
		return
	}
	updates := map[string]any{}
	if req.FirstName != nil {
		updates["first_name"] = *req.FirstName
	}
	if req.LastName != nil {
		updates["last_name"] = *req.LastName
	}
	if req.Nickname != nil {
		updates["nickname"] = *req.Nickname
	}
	if req.IBAN != nil {
		updates["iban"] = *req.IBAN
	}
	if req.BirthDate != nil {
		updates["birth_date"] = *req.BirthDate
	}
	if req.ShoeSize != nil {
		updates["shoe_size"] = *req.ShoeSize
	}
	if req.Weight != nil {
		updates["weight"] = *req.Weight
	}
	if req.PhotoURL != nil {
		updates["photo_url"] = *req.PhotoURL
	}
	if req.Theme != nil {
		updates["theme"] = *req.Theme
	}
	if req.ColorPalette != nil {
		updates["color_palette"] = *req.ColorPalette
	}
	if req.ColorblindMode != nil {
		updates["colorblind_mode"] = *req.ColorblindMode
	}
	if req.Language != nil {
		updates["language"] = *req.Language
	}
	if len(updates) > 0 {
		s.DB.Model(&models.User{}).Where("id = ?", userIDFrom(r)).Updates(updates)
	}
	var user models.User
	s.DB.First(&user, "id = ?", userIDFrom(r))
	writeJSON(w, http.StatusOK, user)
}

// ---- admin user management ----

func (s *Server) handleListUsers(w http.ResponseWriter, r *http.Request) {
	var users []models.User
	s.DB.Order("first_name asc").Find(&users)
	// Apply IBAN visibility (staff aren't exempt — an IBAN is the owner's to share).
	self := userIDFrom(r)
	ptrs := make([]*models.User, len(users))
	for i := range users {
		ptrs[i] = &users[i]
	}
	s.redactIBANs(self, ptrs)
	writeJSON(w, http.StatusOK, users)
}

// ---- forgotten password (public) ----

type forgotPasswordReq struct {
	Email string `json:"email"`
}

// handleForgotPassword emails a reset link. Always returns 200 so attackers
// can't probe which emails exist.
func (s *Server) handleForgotPassword(w http.ResponseWriter, r *http.Request) {
	var req forgotPasswordReq
	if err := decode(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "corps de requête invalide")
		return
	}
	email := strings.ToLower(strings.TrimSpace(req.Email))
	var user models.User
	if err := s.DB.Where("email = ?", email).First(&user).Error; err == nil {
		token := auth.RandomToken(20)
		exp := time.Now().Add(time.Hour)
		s.DB.Model(&user).Updates(map[string]any{"reset_token": token, "reset_token_expiry": exp})
		_ = s.Mailer.SendPasswordReset(user.Email, token)
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

type resetPasswordPublicReq struct {
	Token    string `json:"token"`
	Password string `json:"password"`
}

func (s *Server) handleResetPasswordPublic(w http.ResponseWriter, r *http.Request) {
	var req resetPasswordPublicReq
	if err := decode(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "corps de requête invalide")
		return
	}
	if len(req.Password) < 8 {
		writeError(w, http.StatusBadRequest, "mot de passe d'au moins 8 caractères")
		return
	}
	var user models.User
	if err := s.DB.Where("reset_token = ?", req.Token).First(&user).Error; err != nil || req.Token == "" {
		writeError(w, http.StatusBadRequest, "lien de réinitialisation invalide")
		return
	}
	if user.ResetTokenExpiry == nil || user.ResetTokenExpiry.Before(time.Now()) {
		writeError(w, http.StatusBadRequest, "lien de réinitialisation expiré")
		return
	}
	hash, err := auth.HashPassword(req.Password, s.Cfg.BcryptCost)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "erreur serveur")
		return
	}
	s.DB.Model(&user).Updates(map[string]any{
		"password_hash":      hash,
		"reset_token":        "",
		"reset_token_expiry": nil,
		"email_confirmed":    true, // proving email access also confirms it
	})
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// handleResendConfirmation (admin) re-sends the confirmation email to an
// unconfirmed user.
func (s *Server) handleResendConfirmation(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "id invalide")
		return
	}
	var user models.User
	if err := s.DB.First(&user, "id = ?", id).Error; err != nil {
		writeError(w, http.StatusNotFound, "utilisateur introuvable")
		return
	}
	if user.EmailConfirmed {
		writeJSON(w, http.StatusOK, map[string]any{"alreadyConfirmed": true})
		return
	}
	if user.ConfirmationToken == "" {
		user.ConfirmationToken = auth.RandomToken(20)
		s.DB.Model(&user).Update("confirmation_token", user.ConfirmationToken)
	}
	_ = s.Mailer.SendConfirmation(user.Email, user.ConfirmationToken)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) handlePromoteUser(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "id invalide")
		return
	}
	if err := s.DB.Model(&models.User{}).Where("id = ?", id).Update("role", models.RoleAdmin).Error; err != nil {
		writeError(w, http.StatusInternalServerError, "promotion impossible")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}
