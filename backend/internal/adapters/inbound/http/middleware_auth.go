package httpinbound

import (
	"net/http"
)

// RequireUser middleware: ensures the request has an authenticated user.
// Returns 401 otherwise.
func (h *Server) RequireUser(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u := UserFromContext(r.Context())
		if u == nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// RequireAnonymous middleware: ensures the request has no authenticated user.
// Returns 302 to /dashboard if a user is signed in.
func (h *Server) RequireAnonymous(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if u := UserFromContext(r.Context()); u != nil {
			http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
			return
		}
		next.ServeHTTP(w, r)
	})
}
