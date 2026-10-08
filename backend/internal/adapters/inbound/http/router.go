package httpinbound

import (
	"context"
	"errors"
	"net/http"

	"github.com/alexedwards/scs/v2"
	"github.com/go-chi/chi/v5"
	"github.com/gorilla/csrf"

	"github.com/Tamaiw/ludo-tournament-manager/backend/internal/adapters/outbound/sqlite"
	"github.com/Tamaiw/ludo-tournament-manager/backend/internal/core/domain"
	"github.com/Tamaiw/ludo-tournament-manager/backend/internal/core/ports"
	"github.com/Tamaiw/ludo-tournament-manager/backend/internal/core/services"
)

// Config is the dependency injection struct for the Server.
type Config struct {
	// Ports
	Users              ports.UserRepository
	Tournaments        ports.TournamentRepository
	TournamentManagers ports.TournamentManagerRepository
	TournamentPlayers  ports.TournamentPlayerRepository
	SpectatorTokens    ports.TournamentSpectatorTokenRepository
	Matches            ports.MatchRepository
	AuditLog           ports.AuditLogRepository
	AuthTokens         ports.AuthTokenRepository
	Email              ports.EmailSender
	Broadcaster        ports.Broadcaster
	Clock              ports.Clock
	Hasher             ports.PasswordHasher
	Tx                 ports.Transactional
	SessionStore       *sqlite.SessionStore

	// Public configuration
	CSRFKey      []byte
	CookieSecure bool
	PublicURL    string

	// Services
	Services *services.Bundle
}

// Server is the HTTP inbound adapter.
type Server struct {
	cfg            Config
	sessionMgr     *scs.SessionManager
	authHandler    *AuthHandler
	tourHandler    *TournamentHandler
	playerHandler  *PlayerHandler
	matchHandler   *MatchHandler
	bracketHandler *BracketHandler
	indexHandler   *IndexHandler
	dlHandler      *DownloadHandler
	inviteHandler  *InviteHandler
	auditHandler   *AuditHandler
	pwdHandler     *PasswordResetHandler
}

// New constructs a Server.
func New(cfg Config) *Server {
	s := &Server{cfg: cfg}
	s.sessionMgr = scs.New()
	s.sessionMgr.Lifetime = 7 * 24 * 60 * 60 * 1e9 // 7 days
	s.sessionMgr.IdleTimeout = 24 * 60 * 60 * 1e9
	s.sessionMgr.Cookie.Name = "__Host-id"
	s.sessionMgr.Cookie.HttpOnly = true
	s.sessionMgr.Cookie.SameSite = http.SameSiteStrictMode
	s.sessionMgr.Cookie.Secure = cfg.CookieSecure
	s.sessionMgr.Cookie.Path = "/"
	s.sessionMgr.HashTokenInStore = true
	if cfg.SessionStore != nil {
		s.sessionMgr.Store = NewSCSStoreAdapter(cfg.SessionStore)
	}
	s.authHandler = &AuthHandler{
		Users:                cfg.Users,
		SignIn:               cfg.Services.SignIn,
		ChangePassword:       cfg.Services.ChangePassword,
		RequestPasswordReset: cfg.Services.RequestPasswordReset,
		RedeemPasswordReset:  cfg.Services.RedeemPasswordReset,
		Hasher:               cfg.Hasher,
		Clock:                cfg.Clock,
	}
	s.indexHandler = &IndexHandler{
		Tournaments: cfg.Tournaments,
		TPlayers:    cfg.TournamentPlayers,
		TManagers:   cfg.TournamentManagers,
	}
	s.tourHandler = &TournamentHandler{
		Tournaments:            cfg.Tournaments,
		TManagers:              cfg.TournamentManagers,
		TPlayers:               cfg.TournamentPlayers,
		SpectatorTokens:        cfg.SpectatorTokens,
		Matches:                cfg.Matches,
		AuditLog:               cfg.AuditLog,
		Users:                  cfg.Users,
		CreateTournament:       cfg.Services.CreateTournament,
		EditTournament:         cfg.Services.EditTournament,
		ChangeVisibility:       cfg.Services.ChangeVisibility,
		ChangeRegistrationMode: cfg.Services.ChangeRegistrationMode,
		OpenRegistration:       cfg.Services.OpenRegistration,
		CloseRegistration:      cfg.Services.CloseRegistration,
		StartTournament:        cfg.Services.StartTournament,
		CancelTournament:       cfg.Services.CancelTournament,
		ForceCloseTournament:   cfg.Services.ForceCloseTournament,
		PreviewBracket:         cfg.Services.PreviewBracket,
		AddManager:             cfg.Services.AddManager,
		RemoveManager:          cfg.Services.RemoveManager,
		IssueSpectatorToken:    cfg.Services.IssueSpectatorToken,
		RevokeSpectatorToken:   cfg.Services.RevokeSpectatorToken,
		InviteUser:             cfg.Services.InviteUser,
		Clock:                  cfg.Clock,
		PublicURL:              cfg.PublicURL,
	}
	s.playerHandler = &PlayerHandler{
		Tournaments:     cfg.Tournaments,
		TPlayers:        cfg.TournamentPlayers,
		Users:           cfg.Users,
		RegisterPlayer:  cfg.Services.RegisterPlayer,
		AddPlayerByUser: cfg.Services.AddPlayerByUser,
		WithdrawPlayer:  cfg.Services.WithdrawPlayer,
		EditSeed:        cfg.Services.EditSeed,
		InviteUser:      cfg.Services.InviteUser,
	}
	s.matchHandler = &MatchHandler{
		Tournaments:        cfg.Tournaments,
		Matches:            cfg.Matches,
		TPlayers:           cfg.TournamentPlayers,
		Users:              cfg.Users,
		MarkInProgress:     cfg.Services.MarkInProgress,
		RecordMatchResult:  cfg.Services.RecordMatchResult,
		CorrectMatchResult: cfg.Services.CorrectMatchResult,
	}
	s.bracketHandler = &BracketHandler{
		Tournaments: cfg.Tournaments,
		Matches:     cfg.Matches,
		TPlayers:    cfg.TournamentPlayers,
	}
	s.dlHandler = &DownloadHandler{
		Tournaments:      cfg.Tournaments,
		Matches:          cfg.Matches,
		TPlayers:         cfg.TournamentPlayers,
		Users:            cfg.Users,
		ExportBracketCSV: cfg.Services.ExportBracketCSV,
		ExportBracketPNG: cfg.Services.ExportBracketPNG,
	}
	s.inviteHandler = &InviteHandler{
		Users:        cfg.Users,
		AuthTokens:   cfg.AuthTokens,
		TPlayers:     cfg.TournamentPlayers,
		Clock:        cfg.Clock,
		AcceptInvite: cfg.Services.AcceptInvite,
	}
	s.auditHandler = &AuditHandler{
		AuditLog:     cfg.AuditLog,
		Users:        cfg.Users,
		ListAuditLog: cfg.Services.ListAuditLog,
	}
	s.pwdHandler = &PasswordResetHandler{
		Users:                cfg.Users,
		AuthTokens:           cfg.AuthTokens,
		RequestPasswordReset: cfg.Services.RequestPasswordReset,
		RedeemPasswordReset:  cfg.Services.RedeemPasswordReset,
	}
	return s
}

// Routes returns the http.Handler with the full route table.
func (s *Server) Routes() http.Handler {
	r := chi.NewRouter()
	r.Use(RequestID)
	r.Use(RealIP)
	r.Use(Recoverer)
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			next.ServeHTTP(w, WithSessionManager(req, s.sessionMgr))
		})
	})
	r.Use(s.sessionMgr.LoadAndSave)

	// CSRF: in dev we mark every request as plaintext so the Origin scheme check
	// doesn't reject http requests. In prod we use https scheme checks.
	csrfMiddleware := csrf.Protect(
		s.cfg.CSRFKey,
		csrf.Secure(s.cfg.CookieSecure),
		csrf.SameSite(csrf.SameSiteStrictMode),
		csrf.MaxAge(3600),
		csrf.Path("/"),
	)
	if !s.cfg.CookieSecure {
		// `__Host-` prefix requires Secure (RFC 6265bis). Drop the prefix in
		// dev so the cookie survives an http:// origin.
		s.sessionMgr.Cookie.Name = "id"
		csrfMiddleware = csrf.Protect(
			s.cfg.CSRFKey,
			csrf.Secure(false),
			csrf.SameSite(csrf.SameSiteStrictMode),
			csrf.MaxAge(3600),
			csrf.Path("/"),
		)
		r.Use(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				next.ServeHTTP(w, csrf.PlaintextHTTPRequest(r))
			})
		})
	}
	_ = csrfMiddleware

	r.Use(s.loadUserFromSession)

	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok\n"))
	})

	// Public tournament index.
	r.Get("/", s.indexHandler.PublicIndex)

	// Sign-in (anonymous only).
	r.Group(func(r chi.Router) {
		r.Use(s.RequireAnonymous)
		r.Use(csrfMiddleware)
		r.Get("/sign-in", s.authHandler.SignInPage)
		r.Post("/sign-in", s.authHandler.SignInSubmit)
	})

	r.With(csrfMiddleware).Post("/sign-out", s.authHandler.SignOutSubmit)

	// Authenticated routes.
	r.Group(func(r chi.Router) {
		r.Use(s.RequireUser)
		r.Use(csrfMiddleware)

		r.Get("/dashboard", s.indexHandler.Dashboard)

		r.Get("/change-password", s.authHandler.ChangePasswordPage)
		r.Post("/change-password", s.authHandler.ChangePasswordSubmit)

		r.Get("/tournaments/new", s.tourHandler.NewTournamentPage)
		r.Post("/tournaments", s.tourHandler.CreateTournamentSubmit)

		r.Route("/tournaments/{id}", func(r chi.Router) {
			r.Use(s.loadTournamentRoute)
			r.Get("/", s.tourHandler.ShowTournament)
			r.Get("/edit", s.tourHandler.EditTournamentPage)
			r.Post("/edit", s.tourHandler.EditTournamentSubmit)
			r.Post("/open-registration", s.tourHandler.OpenRegistrationSubmit)
			r.Post("/close-registration", s.tourHandler.CloseRegistrationSubmit)
			r.Post("/start", s.tourHandler.StartTournamentSubmit)
			r.Post("/cancel", s.tourHandler.CancelTournamentSubmit)
			r.Post("/force-close", s.tourHandler.ForceCloseSubmit)
			r.Post("/visibility", s.tourHandler.ChangeVisibilitySubmit)
			r.Post("/registration-mode", s.tourHandler.ChangeRegistrationModeSubmit)
			r.Post("/preview-bracket", s.tourHandler.PreviewBracketSubmit)

			r.Get("/roster", s.playerHandler.RosterPage)
			r.Post("/roster/add", s.playerHandler.AddPlayerSubmit)
			r.Post("/roster/remove", s.playerHandler.RemovePlayerSubmit)
			r.Post("/roster/invite", s.playerHandler.RosterInviteSubmit)
			r.Post("/roster/{playerID}/withdraw", s.playerHandler.WithdrawSubmit)
			r.Post("/roster/{playerID}/seed", s.playerHandler.EditSeedSubmit)

			r.Get("/register", s.playerHandler.SelfRegisterPage)
			r.Post("/register", s.playerHandler.SelfRegisterSubmit)

			r.Post("/managers", s.tourHandler.AddManagerSubmit)
			r.Post("/managers/{managerID}/remove", s.tourHandler.RemoveManagerSubmit)

			r.Get("/spectator-tokens", s.tourHandler.SpectatorTokensPage)
			r.Post("/spectator-tokens", s.tourHandler.IssueSpectatorTokenSubmit)
			r.Post("/spectator-tokens/{tokenID}/revoke", s.tourHandler.RevokeSpectatorTokenSubmit)

			r.Get("/audit", s.auditHandler.AuditLogPage)

			r.Get("/download.csv", s.dlHandler.DownloadCSV)
			r.Get("/download.png", s.dlHandler.DownloadPNG)

			r.Get("/bracket", s.bracketHandler.BracketFragment)
			r.Get("/status", s.bracketHandler.StatusFragment)
			r.Get("/fragments/bracket", s.bracketHandler.BracketFragment)
			r.Get("/fragments/status", s.bracketHandler.StatusFragment)
		})

		r.Get("/matches/{id}", s.matchHandler.MatchPage)
		r.Post("/matches/{id}/start", s.matchHandler.MarkInProgressSubmit)
		r.Post("/matches/{id}/record", s.matchHandler.RecordResultSubmit)
		r.Post("/matches/{id}/correct", s.matchHandler.CorrectResultSubmit)

		r.Route("/invites/{token}", func(r chi.Router) {
			r.Get("/", s.inviteHandler.AcceptInvitePage)
			r.Post("/", s.inviteHandler.AcceptInviteSubmit)
		})
		r.Route("/password-resets/{token}", func(r chi.Router) {
			r.Get("/", s.pwdHandler.PasswordResetPage)
			r.Post("/", s.pwdHandler.PasswordResetSubmit)
		})

		r.Post("/users/{id}/password-reset", s.pwdHandler.RequestPasswordResetSubmit)

		// WebSocket (no CSRF — WS upgrade).
		r.Get("/ws/tournaments/{id}", s.WSSubscribe)
	})

	return r
}

// loadUserFromSession loads the user from the session manager's user_id and puts
// it on the request context.
func (s *Server) loadUserFromSession(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		uid := s.sessionMgr.GetString(r.Context(), "user_id")
		if uid != "" {
			if u, err := s.cfg.Users.FindByID(r.Context(), domain.UserID(uid)); err == nil {
				r = putUser(r, &u)
			}
		}
		next.ServeHTTP(w, r)
	})
}

// loadTournamentRoute is the per-tournament route middleware: enforces visibility
// and stores the loaded tournament + role on the request context.
func (s *Server) loadTournamentRoute(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		idStr := chi.URLParam(r, "id")
		t, err := s.cfg.Tournaments.Find(r.Context(), domain.TournamentID(idStr))
		if err != nil {
			if errors.Is(err, domain.ErrTournamentNotFound) {
				http.NotFound(w, r)
				return
			}
			http.Error(w, "lookup failed: "+err.Error(), http.StatusInternalServerError)
			return
		}
		user := UserFromContext(r.Context())
		hasToken := false
		if c, _ := r.Cookie("spectator_token"); c != nil && c.Value != "" {
			hasToken = true
		}
		if r.URL.Query().Get("spectator_token") != "" {
			hasToken = true
		}
		if !visibilityAllowed(t, user, hasToken) {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		if user != nil {
			if isM, _ := s.cfg.TournamentManagers.IsManager(r.Context(), t.ID, user.ID); isM {
				r = putTournamentRole(r, RoleManager)
			} else if reg, _ := s.cfg.TournamentPlayers.IsRegistered(r.Context(), t.ID, user.ID); reg {
				r = putTournamentRole(r, RolePlayer)
			}
		}
		r = PutTournament(r, &t)
		next.ServeHTTP(w, r)
	})
}

// sessionManager returns the scs manager (used by templates/flash helpers).
func (s *Server) sessionManager(_ context.Context) *scs.SessionManager {
	return s.sessionMgr
}

// WSSubscribe upgrades the HTTP request to WebSocket and joins the
// broadcaster's per-tournament room.
func (s *Server) WSSubscribe(w http.ResponseWriter, r *http.Request) {
	t := TournamentFromContext(r.Context())
	if t == nil {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	hub, ok := s.cfg.Broadcaster.(interface {
		Upgrade(w http.ResponseWriter, r *http.Request, tournamentID string) (Subscription, error)
	})
	if !ok {
		http.Error(w, "broadcaster does not support upgrade", http.StatusInternalServerError)
		return
	}
	sub, err := hub.Upgrade(w, r, t.ID.String())
	if err != nil {
		return
	}
	s.cfg.Broadcaster.Register(t.ID.String(), sub)
}

// Subscription re-exports ports.Subscription for use by Server.WSSubscribe.
type Subscription = ports.Subscription

var _ = context.TODO
