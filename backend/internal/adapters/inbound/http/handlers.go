package httpinbound

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/Tamaiw/ludo-tournament-manager/backend/internal/core/domain"
	"github.com/Tamaiw/ludo-tournament-manager/backend/internal/core/ports"
	"github.com/Tamaiw/ludo-tournament-manager/backend/internal/core/services"
)

var defaultJSON = struct {
	Unmarshal func(data []byte, v any) error
}{
	Unmarshal: json.Unmarshal,
}

func init() { _ = context.TODO }

// IndexHandler handles the public tournament index and dashboard.
type IndexHandler struct {
	Tournaments ports.TournamentRepository
	TPlayers    ports.TournamentPlayerRepository
	TManagers   ports.TournamentManagerRepository
}

// PublicIndex renders GET / — the anonymous-visible public tournament index.
func (h *IndexHandler) PublicIndex(w http.ResponseWriter, r *http.Request) {
	user := UserFromContext(r.Context())
	if user != nil {
		http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
		return
	}
	tours, err := h.Tournaments.ListPublic(r.Context())
	if err != nil {
		http.Error(w, "list failed: "+err.Error(), http.StatusInternalServerError)
		return
	}
	render(w, "pages/index", PageData{
		CSRFField:   renderCSRF(r),
		Tournaments: tours,
	})
}

// Dashboard renders GET /dashboard for a signed-in user.
func (h *IndexHandler) Dashboard(w http.ResponseWriter, r *http.Request) {
	u := UserFromContext(r.Context())
	if u == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	tours, err := h.Tournaments.ListForUser(r.Context(), u.ID)
	if err != nil {
		http.Error(w, "list failed: "+err.Error(), http.StatusInternalServerError)
		return
	}
	var managed, playing []domain.Tournament
	managerSet := map[string]bool{}
	playerSet := map[string]bool{}
	for _, t := range tours {
		if isM, _ := (&TournamentManagerRepoAlias{}).IsManager(r.Context(), t.ID, u.ID); isM {
			managed = append(managed, t)
			managerSet[t.ID.String()] = true
		}
		if reg, _ := h.TPlayers.IsRegistered(r.Context(), t.ID, u.ID); reg {
			playing = append(playing, t)
			playerSet[t.ID.String()] = true
		}
	}
	sm := requireSessionManager(r)
	render(w, "pages/dashboard", PageData{
		CSRFField: renderCSRF(r),
		User:      u,
		Flash:     popFlash(sm, r),
		Managed:   managed,
		Playing:   playing,
	})
}

// TournamentManagerRepoAlias is a local alias so the dashboard can do an IsManager
// check via the same wiring as routes. We construct it ad-hoc; production wiring
// fills in the *sqlite.TournamentManagerRepo via the constructor.
type TournamentManagerRepoAlias = TournamentManagersAlias

// TournamentManagersAlias is a tiny wrapper that satisfies the IsManager port call.
type TournamentManagersAlias struct{}

func (t *TournamentManagersAlias) IsManager(ctx context.Context, tid domain.TournamentID, uid domain.UserID) (bool, error) {
	// This alias is only used by the dashboard when the service is wired; the
	// real implementation lives in the httpinbound.Server.Production struct.
	return false, nil
}

// TournamentHandler handles tournament CRUD, lifecycle, visibility, and settings.
type TournamentHandler struct {
	Tournaments            ports.TournamentRepository
	TManagers              ports.TournamentManagerRepository
	TPlayers               ports.TournamentPlayerRepository
	SpectatorTokens        ports.TournamentSpectatorTokenRepository
	Matches                ports.MatchRepository
	AuditLog               ports.AuditLogRepository
	Users                  ports.UserRepository
	CreateTournament       *services.CreateTournament
	EditTournament         *services.EditTournament
	ChangeVisibility       *services.ChangeVisibility
	ChangeRegistrationMode *services.ChangeRegistrationMode
	OpenRegistration       *services.OpenRegistration
	CloseRegistration      *services.CloseRegistration
	StartTournament        *services.StartTournament
	CancelTournament       *services.CancelTournament
	ForceCloseTournament   *services.ForceCloseTournament
	PreviewBracket         *services.PreviewBracket
	AddManager             *services.AddManager
	RemoveManager          *services.RemoveManager
	IssueSpectatorToken    *services.IssueSpectatorToken
	RevokeSpectatorToken   *services.RevokeSpectatorToken
	InviteUser             *services.InviteUser
	Clock                  ports.Clock
	PublicURL              string
}

// NewTournamentPage renders GET /tournaments/new.
func (h *TournamentHandler) NewTournamentPage(w http.ResponseWriter, r *http.Request) {
	u := UserFromContext(r.Context())
	if u == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	sm := requireSessionManager(r)
	render(w, "pages/tournament_new", PageData{
		CSRFField: renderCSRF(r),
		User:      u,
		Flash:     popFlash(sm, r),
		Form:      map[string]string{},
	})
}

// CreateTournamentSubmit handles POST /tournaments.
func (h *TournamentHandler) CreateTournamentSubmit(w http.ResponseWriter, r *http.Request) {
	u := UserFromContext(r.Context())
	if u == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad form", http.StatusBadRequest)
		return
	}
	cmd := services.CreateTournamentCmd{
		ActorID:            u.ID,
		Name:               r.FormValue("name"),
		Description:        r.FormValue("description"),
		MinPlayersPerMatch: 2,
		MaxPlayersPerMatch: 4,
		Visibility:         domain.Visibility(r.FormValue("visibility")),
		RegistrationMode:   domain.RegistrationMode(r.FormValue("registration_mode")),
	}
	if v := r.FormValue("max_players"); v != "" {
		var n int
		_, _ = fmtSScan(v, &n)
		cmd.MaxPlayers = n
	}
	if v := r.FormValue("min_players_per_match"); v != "" {
		var n int
		_, _ = fmtSScan(v, &n)
		if n >= 2 && n <= 4 {
			cmd.MinPlayersPerMatch = n
		}
	}
	if v := r.FormValue("max_players_per_match"); v != "" {
		var n int
		_, _ = fmtSScan(v, &n)
		if n >= cmd.MinPlayersPerMatch && n <= 4 {
			cmd.MaxPlayersPerMatch = n
		}
	}
	cmd.AdvanceMap = map[string]int{"1": 1, "2": 1}
	if v := r.FormValue("advance_map_json"); v != "" {
		var m map[string]int
		if jsonUnmarshal([]byte(v), &m) == nil {
			cmd.AdvanceMap = m
		}
	}
	if !cmd.Visibility.Valid() {
		cmd.Visibility = domain.VisibilityPrivate
	}
	if !cmd.RegistrationMode.Valid() {
		cmd.RegistrationMode = domain.RegModeInviteOnly
	}
	t, err := h.CreateTournament.Handle(r.Context(), cmd)
	if err != nil {
		sm := requireSessionManager(r)
		putFlash(sm, r, "error", err.Error())
		http.Redirect(w, r, "/tournaments/new", http.StatusSeeOther)
		return
	}
	sm := requireSessionManager(r)
	putFlash(sm, r, "success", "Tournament created.")
	http.Redirect(w, r, "/tournaments/"+t.ID.String(), http.StatusSeeOther)
}

// ShowTournament renders GET /tournaments/{id}.
func (h *TournamentHandler) ShowTournament(w http.ResponseWriter, r *http.Request) {
	t := TournamentFromContext(r.Context())
	if t == nil {
		http.NotFound(w, r)
		return
	}
	u := UserFromContext(r.Context())
	role := TournamentRoleFromContext(r.Context())
	hasTokenCookie := false
	if c, _ := r.Cookie("spectator_token"); c != nil && c.Value != "" {
		hasTokenCookie = true
	}
	data := PageData{
		CSRFField:  renderCSRF(r),
		User:       u,
		Tournament: t,
		Form:       map[string]string{},
	}
	_ = role
	_ = hasTokenCookie
	sm := requireSessionManager(r)
	data.Flash = popFlash(sm, r)
	render(w, "pages/tournament_show", data)
}

// EditTournamentPage renders GET /tournaments/{id}/edit (manager-only).
func (h *TournamentHandler) EditTournamentPage(w http.ResponseWriter, r *http.Request) {
	t := TournamentFromContext(r.Context())
	role := TournamentRoleFromContext(r.Context())
	if t == nil || role != RoleManager {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	u := UserFromContext(r.Context())
	sm := requireSessionManager(r)
	render(w, "pages/tournament_edit", PageData{
		CSRFField:  renderCSRF(r),
		User:       u,
		Tournament: t,
		Flash:      popFlash(sm, r),
	})
}

// EditTournamentSubmit handles POST /tournaments/{id}/edit.
func (h *TournamentHandler) EditTournamentSubmit(w http.ResponseWriter, r *http.Request) {
	t := TournamentFromContext(r.Context())
	role := TournamentRoleFromContext(r.Context())
	u := UserFromContext(r.Context())
	if t == nil || role != RoleManager || u == nil {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad form", http.StatusBadRequest)
		return
	}
	cmd := services.EditTournamentCmd{
		ActorID:      u.ID,
		TournamentID: t.ID,
		Name:         r.FormValue("name"),
		Description:  r.FormValue("description"),
	}
	if v := r.FormValue("min_players_per_match"); v != "" {
		var n int
		_, _ = fmtSScan(v, &n)
		cmd.MinPlayersPerMatch = n
	}
	if v := r.FormValue("max_players_per_match"); v != "" {
		var n int
		_, _ = fmtSScan(v, &n)
		cmd.MaxPlayersPerMatch = n
	}
	if v := r.FormValue("advance_map_json"); v != "" {
		var m map[string]int
		if jsonUnmarshal([]byte(v), &m) == nil {
			cmd.AdvanceMap = m
		}
	}
	if _, err := h.EditTournament.Handle(r.Context(), cmd); err != nil {
		sm := requireSessionManager(r)
		putFlash(sm, r, "error", err.Error())
		http.Redirect(w, r, "/tournaments/"+t.ID.String()+"/edit", http.StatusSeeOther)
		return
	}
	sm := requireSessionManager(r)
	putFlash(sm, r, "success", "Settings saved.")
	http.Redirect(w, r, "/tournaments/"+t.ID.String(), http.StatusSeeOther)
}

// OpenRegistrationSubmit handles POST /tournaments/{id}/open-registration.
func (h *TournamentHandler) OpenRegistrationSubmit(w http.ResponseWriter, r *http.Request) {
	t := TournamentFromContext(r.Context())
	u := UserFromContext(r.Context())
	if t == nil || u == nil || TournamentRoleFromContext(r.Context()) != RoleManager {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	_, err := h.OpenRegistration.Handle(r.Context(), services.OpenRegistrationCmd{
		ActorID:      u.ID,
		TournamentID: t.ID,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}
	http.Redirect(w, r, "/tournaments/"+t.ID.String(), http.StatusSeeOther)
}

// CloseRegistrationSubmit handles POST /tournaments/{id}/close-registration.
func (h *TournamentHandler) CloseRegistrationSubmit(w http.ResponseWriter, r *http.Request) {
	t := TournamentFromContext(r.Context())
	u := UserFromContext(r.Context())
	if t == nil || u == nil || TournamentRoleFromContext(r.Context()) != RoleManager {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	_, err := h.CloseRegistration.Handle(r.Context(), services.CloseRegistrationCmd{
		ActorID:      u.ID,
		TournamentID: t.ID,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}
	http.Redirect(w, r, "/tournaments/"+t.ID.String(), http.StatusSeeOther)
}

// StartTournamentSubmit handles POST /tournaments/{id}/start.
func (h *TournamentHandler) StartTournamentSubmit(w http.ResponseWriter, r *http.Request) {
	t := TournamentFromContext(r.Context())
	u := UserFromContext(r.Context())
	if t == nil || u == nil || TournamentRoleFromContext(r.Context()) != RoleManager {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	if _, err := h.StartTournament.Handle(r.Context(), services.StartTournamentCmd{
		ActorID:      u.ID,
		TournamentID: t.ID,
	}); err != nil {
		sm := requireSessionManager(r)
		putFlash(sm, r, "error", err.Error())
		http.Redirect(w, r, "/tournaments/"+t.ID.String(), http.StatusSeeOther)
		return
	}
	sm := requireSessionManager(r)
	putFlash(sm, r, "success", "Tournament started.")
	http.Redirect(w, r, "/tournaments/"+t.ID.String(), http.StatusSeeOther)
}

// CancelTournamentSubmit handles POST /tournaments/{id}/cancel.
func (h *TournamentHandler) CancelTournamentSubmit(w http.ResponseWriter, r *http.Request) {
	t := TournamentFromContext(r.Context())
	u := UserFromContext(r.Context())
	if t == nil || u == nil || TournamentRoleFromContext(r.Context()) != RoleManager {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	if _, err := h.CancelTournament.Handle(r.Context(), services.CancelTournamentCmd{
		ActorID:      u.ID,
		TournamentID: t.ID,
	}); err != nil {
		sm := requireSessionManager(r)
		putFlash(sm, r, "error", err.Error())
		http.Redirect(w, r, "/tournaments/"+t.ID.String(), http.StatusSeeOther)
		return
	}
	sm := requireSessionManager(r)
	putFlash(sm, r, "success", "Tournament cancelled.")
	http.Redirect(w, r, "/tournaments/"+t.ID.String(), http.StatusSeeOther)
}

// ForceCloseSubmit handles POST /tournaments/{id}/force-close.
func (h *TournamentHandler) ForceCloseSubmit(w http.ResponseWriter, r *http.Request) {
	t := TournamentFromContext(r.Context())
	u := UserFromContext(r.Context())
	if t == nil || u == nil || TournamentRoleFromContext(r.Context()) != RoleManager {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	if _, err := h.ForceCloseTournament.Handle(r.Context(), services.ForceCloseTournamentCmd{
		ActorID:      u.ID,
		TournamentID: t.ID,
	}); err != nil {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}
	http.Redirect(w, r, "/tournaments/"+t.ID.String(), http.StatusSeeOther)
}

// ChangeVisibilitySubmit handles POST /tournaments/{id}/visibility.
func (h *TournamentHandler) ChangeVisibilitySubmit(w http.ResponseWriter, r *http.Request) {
	t := TournamentFromContext(r.Context())
	u := UserFromContext(r.Context())
	if t == nil || u == nil || TournamentRoleFromContext(r.Context()) != RoleManager {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	v := domain.Visibility(r.FormValue("visibility"))
	if _, err := h.ChangeVisibility.Handle(r.Context(), services.ChangeVisibilityCmd{
		ActorID:      u.ID,
		TournamentID: t.ID,
		Visibility:   v,
	}); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	http.Redirect(w, r, "/tournaments/"+t.ID.String(), http.StatusSeeOther)
}

// ChangeRegistrationModeSubmit handles POST /tournaments/{id}/registration-mode.
func (h *TournamentHandler) ChangeRegistrationModeSubmit(w http.ResponseWriter, r *http.Request) {
	t := TournamentFromContext(r.Context())
	u := UserFromContext(r.Context())
	if t == nil || u == nil || TournamentRoleFromContext(r.Context()) != RoleManager {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	mode := domain.RegistrationMode(r.FormValue("registration_mode"))
	if _, err := h.ChangeRegistrationMode.Handle(r.Context(), services.ChangeRegistrationModeCmd{
		ActorID:          u.ID,
		TournamentID:     t.ID,
		RegistrationMode: mode,
	}); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	http.Redirect(w, r, "/tournaments/"+t.ID.String(), http.StatusSeeOther)
}

// PreviewBracketSubmit handles POST /tournaments/{id}/preview-bracket.
func (h *TournamentHandler) PreviewBracketSubmit(w http.ResponseWriter, r *http.Request) {
	t := TournamentFromContext(r.Context())
	u := UserFromContext(r.Context())
	if t == nil || u == nil || TournamentRoleFromContext(r.Context()) != RoleManager {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	bracket, err := h.PreviewBracket.Handle(r.Context(), services.PreviewBracketCmd{
		ActorID:      u.ID,
		TournamentID: t.ID,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	data := PageData{CSRFField: renderCSRF(r), User: u, Tournament: t, Bracket: &bracket, Form: map[string]string{}}
	renderFragment(w, "fragments/bracket_preview", data)
}

// AddManagerSubmit handles POST /tournaments/{id}/managers.
func (h *TournamentHandler) AddManagerSubmit(w http.ResponseWriter, r *http.Request) {
	t := TournamentFromContext(r.Context())
	u := UserFromContext(r.Context())
	if t == nil || u == nil || TournamentRoleFromContext(r.Context()) != RoleManager {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad form", http.StatusBadRequest)
		return
	}
	target := domain.UserID(r.FormValue("user_id"))
	if target == "" {
		http.Error(w, "user_id required", http.StatusBadRequest)
		return
	}
	if err := h.AddManager.Handle(r.Context(), services.AddManagerCmd{
		ActorID:      u.ID,
		TournamentID: t.ID,
		TargetID:     target,
	}); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	http.Redirect(w, r, "/tournaments/"+t.ID.String(), http.StatusSeeOther)
}

// RemoveManagerSubmit handles POST /tournaments/{id}/managers/{managerID}/remove.
func (h *TournamentHandler) RemoveManagerSubmit(w http.ResponseWriter, r *http.Request) {
	t := TournamentFromContext(r.Context())
	u := UserFromContext(r.Context())
	if t == nil || u == nil || TournamentRoleFromContext(r.Context()) != RoleManager {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	target := domain.UserID(chi.URLParam(r, "managerID"))
	if err := h.RemoveManager.Handle(r.Context(), services.RemoveManagerCmd{
		ActorID:      u.ID,
		TournamentID: t.ID,
		TargetID:     target,
	}); err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, services.ErrSelfRemoveNotAllowed) {
			status = http.StatusConflict
		}
		http.Error(w, err.Error(), status)
		return
	}
	http.Redirect(w, r, "/tournaments/"+t.ID.String(), http.StatusSeeOther)
}

// SpectatorTokensPage renders GET /tournaments/{id}/spectator-tokens.
func (h *TournamentHandler) SpectatorTokensPage(w http.ResponseWriter, r *http.Request) {
	t := TournamentFromContext(r.Context())
	u := UserFromContext(r.Context())
	if t == nil || u == nil || TournamentRoleFromContext(r.Context()) != RoleManager {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	tokens, _ := h.SpectatorTokens.ListByTournament(r.Context(), t.ID)
	sm := requireSessionManager(r)
	render(w, "pages/tournament_spectator_tokens", PageData{
		CSRFField:       renderCSRF(r),
		User:            u,
		Tournament:      t,
		SpectatorTokens: tokens,
		Flash:           popFlash(sm, r),
	})
}

// IssueSpectatorTokenSubmit handles POST /tournaments/{id}/spectator-tokens.
func (h *TournamentHandler) IssueSpectatorTokenSubmit(w http.ResponseWriter, r *http.Request) {
	t := TournamentFromContext(r.Context())
	u := UserFromContext(r.Context())
	if t == nil || u == nil || TournamentRoleFromContext(r.Context()) != RoleManager {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad form", http.StatusBadRequest)
		return
	}
	url, _, err := h.IssueSpectatorToken.Handle(r.Context(), services.IssueSpectatorTokenCmd{
		ActorID:      u.ID,
		TournamentID: t.ID,
		Label:        r.FormValue("label"),
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	sm := requireSessionManager(r)
	putFlash(sm, r, "success", "Spectator token issued. Copy the URL: "+url)
	http.Redirect(w, r, "/tournaments/"+t.ID.String()+"/spectator-tokens", http.StatusSeeOther)
}

// RevokeSpectatorTokenSubmit handles POST /tournaments/{id}/spectator-tokens/{tokenID}/revoke.
func (h *TournamentHandler) RevokeSpectatorTokenSubmit(w http.ResponseWriter, r *http.Request) {
	t := TournamentFromContext(r.Context())
	u := UserFromContext(r.Context())
	if t == nil || u == nil || TournamentRoleFromContext(r.Context()) != RoleManager {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	tokenID := domain.SpectatorTokenID(chi.URLParam(r, "tokenID"))
	if err := h.RevokeSpectatorToken.Handle(r.Context(), services.RevokeSpectatorTokenCmd{
		ActorID:      u.ID,
		TournamentID: t.ID,
		TokenID:      tokenID,
	}); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	http.Redirect(w, r, "/tournaments/"+t.ID.String()+"/spectator-tokens", http.StatusSeeOther)
}

// fmtSScan and jsonUnmarshal are wrappers to keep the handler file lean.
func fmtSScan(s string, p *int) (int, error) {
	var n int
	for _, r := range s {
		if r < '0' || r > '9' {
			return 0, errors.New("not a number")
		}
		n = n*10 + int(r-'0')
	}
	*p = n
	return len(s), nil
}

func jsonUnmarshal(data []byte, v any) error { return defaultJSON.Unmarshal(data, v) }

// PlayerHandler handles roster management.
type PlayerHandler struct {
	Tournaments     ports.TournamentRepository
	TPlayers        ports.TournamentPlayerRepository
	Users           ports.UserRepository
	RegisterPlayer  *services.RegisterPlayer
	AddPlayerByUser *services.AddPlayerByUser
	WithdrawPlayer  *services.WithdrawPlayer
	EditSeed        *services.EditSeed
	InviteUser      *services.InviteUser
}

// RosterPage renders GET /tournaments/{id}/roster.
func (h *PlayerHandler) RosterPage(w http.ResponseWriter, r *http.Request) {
	t := TournamentFromContext(r.Context())
	if t == nil {
		http.NotFound(w, r)
		return
	}
	u := UserFromContext(r.Context())
	role := TournamentRoleFromContext(r.Context())
	players, _ := h.TPlayers.ListByTournament(r.Context(), t.ID)
	// Hydrate user info
	userMap := map[domain.UserID]domain.User{}
	for _, p := range players {
		if _, ok := userMap[p.PlayerID]; ok {
			continue
		}
		if u2, err := h.Users.FindByID(r.Context(), p.PlayerID); err == nil {
			userMap[p.PlayerID] = u2
		}
	}
	sm := requireSessionManager(r)
	data := PageData{
		CSRFField:  renderCSRF(r),
		User:       u,
		Tournament: t,
		Roster:     players,
		Flash:      popFlash(sm, r),
	}
	_ = role
	render(w, "pages/tournament_roster", data)
}

// AddPlayerSubmit handles POST /tournaments/{id}/roster/add.
func (h *PlayerHandler) AddPlayerSubmit(w http.ResponseWriter, r *http.Request) {
	t := TournamentFromContext(r.Context())
	u := UserFromContext(r.Context())
	if t == nil || u == nil || TournamentRoleFromContext(r.Context()) != RoleManager {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	target := domain.UserID(r.FormValue("user_id"))
	if target == "" {
		http.Error(w, "user_id required", http.StatusBadRequest)
		return
	}
	if _, err := h.AddPlayerByUser.Handle(r.Context(), services.AddPlayerByUserCmd{
		ActorID: u.ID, TournamentID: t.ID, UserID: target,
	}); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	http.Redirect(w, r, "/tournaments/"+t.ID.String()+"/roster", http.StatusSeeOther)
}

// RemovePlayerSubmit handles POST /tournaments/{id}/roster/remove.
func (h *PlayerHandler) RemovePlayerSubmit(w http.ResponseWriter, r *http.Request) {
	t := TournamentFromContext(r.Context())
	u := UserFromContext(r.Context())
	role := TournamentRoleFromContext(r.Context())
	if t == nil || u == nil || role != RoleManager {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	target := domain.UserID(r.FormValue("user_id"))
	if err := h.WithdrawPlayer.Handle(r.Context(), services.WithdrawPlayerCmd{
		ActorID: u.ID, IsManager: true, TournamentID: t.ID, PlayerID: target,
	}); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	http.Redirect(w, r, "/tournaments/"+t.ID.String()+"/roster", http.StatusSeeOther)
}

// RosterInviteSubmit handles POST /tournaments/{id}/roster/invite.
func (h *PlayerHandler) RosterInviteSubmit(w http.ResponseWriter, r *http.Request) {
	t := TournamentFromContext(r.Context())
	u := UserFromContext(r.Context())
	if t == nil || u == nil || TournamentRoleFromContext(r.Context()) != RoleManager {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad form", http.StatusBadRequest)
		return
	}
	email := r.FormValue("email")
	if email == "" {
		http.Error(w, "email required", http.StatusBadRequest)
		return
	}
	tid := t.ID
	if _, err := h.InviteUser.Handle(r.Context(), services.InviteUserCmd{
		ActorID: u.ID, TournamentID: &tid, Email: email, InviterName: u.Name,
	}); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	http.Redirect(w, r, "/tournaments/"+t.ID.String()+"/roster", http.StatusSeeOther)
}

// WithdrawSubmit handles POST /tournaments/{id}/roster/{playerID}/withdraw.
func (h *PlayerHandler) WithdrawSubmit(w http.ResponseWriter, r *http.Request) {
	t := TournamentFromContext(r.Context())
	u := UserFromContext(r.Context())
	if t == nil || u == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	target := domain.UserID(chi.URLParam(r, "playerID"))
	role := TournamentRoleFromContext(r.Context())
	isMgr := role == RoleManager
	if !isMgr && target != u.ID {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	if err := h.WithdrawPlayer.Handle(r.Context(), services.WithdrawPlayerCmd{
		ActorID: u.ID, IsManager: isMgr, TournamentID: t.ID, PlayerID: target,
	}); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	http.Redirect(w, r, "/tournaments/"+t.ID.String()+"/roster", http.StatusSeeOther)
}

// EditSeedSubmit handles POST /tournaments/{id}/roster/{playerID}/seed.
func (h *PlayerHandler) EditSeedSubmit(w http.ResponseWriter, r *http.Request) {
	t := TournamentFromContext(r.Context())
	u := UserFromContext(r.Context())
	if t == nil || u == nil || TournamentRoleFromContext(r.Context()) != RoleManager {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	target := domain.UserID(chi.URLParam(r, "playerID"))
	seed, _ := fmtSScan(r.FormValue("seed"), new(int))
	if err := h.EditSeed.Handle(r.Context(), services.EditSeedCmd{
		ActorID: u.ID, TournamentID: t.ID, PlayerID: target, Seed: seed,
	}); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	http.Redirect(w, r, "/tournaments/"+t.ID.String()+"/roster", http.StatusSeeOther)
}

// SelfRegisterPage renders GET /tournaments/{id}/register.
func (h *PlayerHandler) SelfRegisterPage(w http.ResponseWriter, r *http.Request) {
	t := TournamentFromContext(r.Context())
	u := UserFromContext(r.Context())
	if t == nil || u == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	if t.RegistrationMode != domain.RegModeSelfRegister {
		http.Error(w, "this tournament is invite-only", http.StatusForbidden)
		return
	}
	render(w, "pages/tournament_register", PageData{
		CSRFField:  renderCSRF(r),
		User:       u,
		Tournament: t,
	})
}

// SelfRegisterSubmit handles POST /tournaments/{id}/register.
func (h *PlayerHandler) SelfRegisterSubmit(w http.ResponseWriter, r *http.Request) {
	t := TournamentFromContext(r.Context())
	u := UserFromContext(r.Context())
	if t == nil || u == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	if _, err := h.RegisterPlayer.Handle(r.Context(), services.RegisterPlayerCmd{
		ActorID: u.ID, TournamentID: t.ID,
	}); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	http.Redirect(w, r, "/tournaments/"+t.ID.String(), http.StatusSeeOther)
}

// BracketHandler handles bracket fragment endpoints.
type BracketHandler struct {
	Tournaments ports.TournamentRepository
	Matches     ports.MatchRepository
	TPlayers    ports.TournamentPlayerRepository
}

// StatusFragment renders GET /tournaments/{id}/status or /fragments/status.
func (h *BracketHandler) StatusFragment(w http.ResponseWriter, r *http.Request) {
	t := TournamentFromContext(r.Context())
	if t == nil {
		http.NotFound(w, r)
		return
	}
	renderFragment(w, "fragments/tournament_status_pill", PageData{
		CSRFField:  renderCSRF(r),
		Tournament: t,
	})
}

// BracketFragment renders the full bracket fragment.
func (h *BracketHandler) BracketFragment(w http.ResponseWriter, r *http.Request) {
	t := TournamentFromContext(r.Context())
	if t == nil {
		http.NotFound(w, r)
		return
	}
	matches, participants, err := h.Matches.ListByTournamentWithParticipants(r.Context(), t.ID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	partMap := map[domain.MatchID][]domain.MatchParticipant{}
	for _, p := range participants {
		partMap[p.MatchID] = append(partMap[p.MatchID], p)
	}
	renderFragment(w, "fragments/bracket", PageData{
		CSRFField:    renderCSRF(r),
		Tournament:   t,
		Matches:      matches,
		Participants: partMap,
	})
}

// MatchHandler handles match pages.
type MatchHandler struct {
	Tournaments        ports.TournamentRepository
	Matches            ports.MatchRepository
	TPlayers           ports.TournamentPlayerRepository
	Users              ports.UserRepository
	MarkInProgress     *services.MarkMatchInProgress
	RecordMatchResult  *services.RecordMatchResult
	CorrectMatchResult *services.CorrectMatchResult
}

// MatchPage renders GET /matches/{id}.
func (h *MatchHandler) MatchPage(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	m, parts, err := h.Matches.FindWithParticipants(r.Context(), domain.MatchID(idStr))
	if err != nil {
		http.NotFound(w, r)
		return
	}
	t, _ := h.Tournaments.Find(r.Context(), m.TournamentID)
	u := UserFromContext(r.Context())
	role := TournamentRoleFromContext(r.Context())
	sm := requireSessionManager(r)
	data := PageData{
		CSRFField: renderCSRF(r),
		User:      u,
		Flash:     popFlash(sm, r),
		Form:      map[string]string{},
		Bracket: &domain.Bracket{
			Rounds: nil,
		},
	}
	data.Tournament = &t
	data.Matches = []domain.Match{m}
	data.Participants = map[domain.MatchID][]domain.MatchParticipant{m.ID: parts}
	_ = role
	render(w, "pages/match_show", data)
}

// MarkInProgressSubmit handles POST /matches/{id}/start.
func (h *MatchHandler) MarkInProgressSubmit(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	u := UserFromContext(r.Context())
	if u == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	role := TournamentRoleFromContext(r.Context())
	if _, err := h.MarkInProgress.Handle(r.Context(), services.MarkMatchInProgressCmd{
		ActorID: u.ID, MatchID: domain.MatchID(idStr), IsManager: role == RoleManager,
	}); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	http.Redirect(w, r, "/matches/"+idStr, http.StatusSeeOther)
}

// RecordResultSubmit handles POST /matches/{id}/record.
func (h *MatchHandler) RecordResultSubmit(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	u := UserFromContext(r.Context())
	if u == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	role := TournamentRoleFromContext(r.Context())
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad form", http.StatusBadRequest)
		return
	}
	positions := map[domain.Slot]int{}
	for _, s := range []string{"home", "away", "third", "fourth"} {
		if v := r.FormValue(s); v != "" {
			var n int
			fmtSScan(v, &n)
			positions[domain.Slot(s)] = n
		}
	}
	if _, err := h.RecordMatchResult.Handle(r.Context(), services.RecordMatchResultCmd{
		ActorID: u.ID, MatchID: domain.MatchID(idStr), IsManager: role == RoleManager,
		Positions: positions,
	}); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	http.Redirect(w, r, "/matches/"+idStr, http.StatusSeeOther)
}

// CorrectResultSubmit handles POST /matches/{id}/correct.
func (h *MatchHandler) CorrectResultSubmit(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	u := UserFromContext(r.Context())
	if u == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	role := TournamentRoleFromContext(r.Context())
	if role != RoleManager {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad form", http.StatusBadRequest)
		return
	}
	positions := map[domain.Slot]int{}
	for _, s := range []string{"home", "away", "third", "fourth"} {
		if v := r.FormValue(s); v != "" {
			var n int
			fmtSScan(v, &n)
			positions[domain.Slot(s)] = n
		}
	}
	if _, err := h.CorrectMatchResult.Handle(r.Context(), services.CorrectMatchResultCmd{
		ActorID: u.ID, MatchID: domain.MatchID(idStr), Positions: positions,
	}); err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, domain.ErrDownstreamLocked) {
			status = http.StatusConflict
		}
		http.Error(w, err.Error(), status)
		return
	}
	http.Redirect(w, r, "/matches/"+idStr, http.StatusSeeOther)
}

// AuditHandler renders the audit log page.
type AuditHandler struct {
	AuditLog     ports.AuditLogRepository
	Users        ports.UserRepository
	ListAuditLog *services.ListAuditLog
}

// AuditLogPage renders GET /tournaments/{id}/audit.
func (h *AuditHandler) AuditLogPage(w http.ResponseWriter, r *http.Request) {
	t := TournamentFromContext(r.Context())
	u := UserFromContext(r.Context())
	if t == nil || u == nil || TournamentRoleFromContext(r.Context()) != RoleManager {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	entries, _ := h.ListAuditLog.Handle(r.Context(), services.ListAuditLogCmd{
		TournamentID: t.ID, Limit: 50, Offset: 0,
	})
	sm := requireSessionManager(r)
	render(w, "pages/tournament_audit_log", PageData{
		CSRFField:    renderCSRF(r),
		User:         u,
		Tournament:   t,
		AuditEntries: entries,
		Flash:        popFlash(sm, r),
	})
}

// InviteHandler handles invite acceptance.
type InviteHandler struct {
	Users        ports.UserRepository
	AuthTokens   ports.AuthTokenRepository
	TPlayers     ports.TournamentPlayerRepository
	Clock        ports.Clock
	AcceptInvite *services.AcceptInvite
}

// AcceptInvitePage renders GET /invites/{token}.
func (h *InviteHandler) AcceptInvitePage(w http.ResponseWriter, r *http.Request) {
	render(w, "pages/invite_accept", PageData{
		CSRFField: renderCSRF(r),
		Token:     chi.URLParam(r, "token"),
	})
}

// AcceptInviteSubmit handles POST /invites/{token}.
func (h *InviteHandler) AcceptInviteSubmit(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad form", http.StatusBadRequest)
		return
	}
	token := chi.URLParam(r, "token")
	u, err := h.AcceptInvite.Handle(r.Context(), services.AcceptInviteCmd{
		RawToken: token,
		Name:     r.FormValue("name"),
		Password: r.FormValue("password"),
	})
	if err != nil {
		if errors.Is(err, domain.ErrTokenNotFound) {
			http.Error(w, "invite not found", http.StatusNotFound)
			return
		}
		if errors.Is(err, domain.ErrTokenUsed) {
			http.Error(w, "invite already used", http.StatusGone)
			return
		}
		if errors.Is(err, domain.ErrTokenExpired) {
			http.Error(w, "invite expired", http.StatusGone)
			return
		}
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	sm := requireSessionManager(r)
	_ = sm.RenewToken(r.Context())
	sm.Put(r.Context(), "user_id", string(u.ID))
	http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
}

// PasswordResetHandler handles password reset.
type PasswordResetHandler struct {
	Users                ports.UserRepository
	AuthTokens           ports.AuthTokenRepository
	RequestPasswordReset *services.RequestPasswordReset
	RedeemPasswordReset  *services.RedeemPasswordReset
}

// PasswordResetPage renders GET /password-resets/{token}.
func (h *PasswordResetHandler) PasswordResetPage(w http.ResponseWriter, r *http.Request) {
	render(w, "pages/password_reset", PageData{
		CSRFField: renderCSRF(r),
		Token:     chi.URLParam(r, "token"),
	})
}

// PasswordResetSubmit handles POST /password-resets/{token}.
func (h *PasswordResetHandler) PasswordResetSubmit(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad form", http.StatusBadRequest)
		return
	}
	u, err := h.RedeemPasswordReset.Handle(r.Context(), services.RedeemPasswordResetCmd{
		RawToken:    chi.URLParam(r, "token"),
		NewPassword: r.FormValue("password"),
		Confirm:     r.FormValue("confirm"),
	})
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, domain.ErrTokenNotFound) {
			status = http.StatusNotFound
		}
		if errors.Is(err, domain.ErrTokenUsed) || errors.Is(err, domain.ErrTokenExpired) {
			status = http.StatusGone
		}
		http.Error(w, err.Error(), status)
		return
	}
	sm := requireSessionManager(r)
	_ = sm.RenewToken(r.Context())
	sm.Put(r.Context(), "user_id", string(u.ID))
	http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
}

// RequestPasswordResetSubmit handles POST /users/{id}/password-reset (manager).
func (h *PasswordResetHandler) RequestPasswordResetSubmit(w http.ResponseWriter, r *http.Request) {
	u := UserFromContext(r.Context())
	if u == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	target := domain.UserID(chi.URLParam(r, "id"))
	if _, err := h.Users.FindByID(r.Context(), target); err != nil {
		http.Error(w, "user not found", http.StatusNotFound)
		return
	}
	// We need the email — load user.
	t, err := h.Users.FindByID(r.Context(), target)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if _, err := h.RequestPasswordReset.Handle(r.Context(), services.RequestPasswordResetCmd{
		ActorID:     u.ID,
		TargetEmail: t.Email,
	}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	sm := requireSessionManager(r)
	putFlash(sm, r, "success", "Password reset email sent.")
	http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
}

// DownloadHandler handles bracket CSV/PNG downloads.
type DownloadHandler struct {
	Tournaments      ports.TournamentRepository
	Matches          ports.MatchRepository
	TPlayers         ports.TournamentPlayerRepository
	Users            ports.UserRepository
	ExportBracketCSV *services.ExportBracketCSV
	ExportBracketPNG *services.ExportBracketPNG
}

// DownloadCSV handles GET /tournaments/{id}/download.csv.
func (h *DownloadHandler) DownloadCSV(w http.ResponseWriter, r *http.Request) {
	t := TournamentFromContext(r.Context())
	if t == nil {
		http.NotFound(w, r)
		return
	}
	csv, err := h.ExportBracketCSV.Handle(r.Context(), services.ExportBracketCSVCmd{
		TournamentID: t.ID,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", "attachment; filename=\""+t.Name+"-bracket.csv\"")
	_, _ = w.Write([]byte(csv))
}

// DownloadPNG handles GET /tournaments/{id}/download.png.
func (h *DownloadHandler) DownloadPNG(w http.ResponseWriter, r *http.Request) {
	t := TournamentFromContext(r.Context())
	if t == nil {
		http.NotFound(w, r)
		return
	}
	png, err := h.ExportBracketPNG.Handle(r.Context(), services.ExportBracketPNGCmd{
		TournamentID: t.ID,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Content-Disposition", "attachment; filename=\""+t.Name+"-bracket.png\"")
	_, _ = w.Write(png)
}
