package httpinbound

import (
	"errors"
	"net/http"

	"github.com/alexedwards/scs/v2"

	"github.com/Tamaiw/ludo-tournament-manager/backend/internal/core/ports"
	"github.com/Tamaiw/ludo-tournament-manager/backend/internal/core/services"
)

// _ ensures scs is referenced (its constructor is used in dev/main wiring elsewhere).
var _ = scs.New

// AuthHandler wires the sign-in / sign-out / change-password / password-reset endpoints.
type AuthHandler struct {
	Users         ports.UserRepository
	SignIn        *services.SignIn
	ChangePassword *services.ChangePassword
	RequestPasswordReset *services.RequestPasswordReset
	RedeemPasswordReset *services.RedeemPasswordReset
	Hasher        ports.PasswordHasher
	Clock         ports.Clock
}

// SignInPage renders GET /sign-in.
func (h *AuthHandler) SignInPage(w http.ResponseWriter, r *http.Request) {
	render(w, "pages/sign_in", PageData{CSRFField: renderCSRF(r)})
}

// SignInSubmit handles POST /sign-in.
func (h *AuthHandler) SignInSubmit(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad form", http.StatusBadRequest)
		return
	}
	email := r.FormValue("email")
	password := r.FormValue("password")

	sm := requireSessionManager(r)
	u, err := h.SignIn.Handle(r.Context(), services.SignInCmd{Email: email, Password: password})
	if err != nil {
		if errors.Is(err, services.ErrInvalidCredentials) {
			render(w, "pages/sign_in", PageData{
				CSRFField: renderCSRF(r),
				Error:     "Invalid email or password.",
			})
			return
		}
		http.Error(w, "sign-in failed: "+err.Error(), http.StatusInternalServerError)
		return
	}
	// OWASP session fixation defence: renew token before privilege change.
	_ = sm.RenewToken(r.Context())
	sm.Put(r.Context(), "user_id", string(u.ID))
	putFlash(sm, r, "success", "Signed in.")
	// Force the session commit before we redirect, so the new Set-Cookie is
	// in the response. scs's LoadAndSave middleware will still commit, but we
	// want this to be visible immediately to browsers and curl.
	w.Header().Set("Location", "/dashboard")
	w.WriteHeader(http.StatusSeeOther)
}

// SignOutSubmit handles POST /sign-out.
func (h *AuthHandler) SignOutSubmit(w http.ResponseWriter, r *http.Request) {
	sm := requireSessionManager(r)
	_ = sm.RenewToken(r.Context())
	sm.Destroy(r.Context())
	http.Redirect(w, r, "/sign-in", http.StatusSeeOther)
}

// ChangePasswordPage renders GET /change-password.
func (h *AuthHandler) ChangePasswordPage(w http.ResponseWriter, r *http.Request) {
	u := UserFromContext(r.Context())
	if u == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	sm := requireSessionManager(r)
	render(w, "pages/change_password", PageData{
		CSRFField: renderCSRF(r),
		User:      u,
		Flash:     popFlash(sm, r),
	})
}

// ChangePasswordSubmit handles POST /change-password.
func (h *AuthHandler) ChangePasswordSubmit(w http.ResponseWriter, r *http.Request) {
	u := UserFromContext(r.Context())
	if u == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad form", http.StatusBadRequest)
		return
	}
	sm := requireSessionManager(r)
	current := r.FormValue("current_password")
	new := r.FormValue("new_password")
	confirm := r.FormValue("confirm_password")
	_, err := h.ChangePassword.Handle(r.Context(), services.ChangePasswordCmd{
		UserID:      u.ID,
		Current:     current,
		NewPassword: new,
		Confirm:     confirm,
	})
	if err != nil {
		render(w, "pages/change_password", PageData{
			CSRFField: renderCSRF(r),
			User:      u,
			Error:     err.Error(),
		})
		return
	}
	putFlash(sm, r, "success", "Password changed.")
	http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
}