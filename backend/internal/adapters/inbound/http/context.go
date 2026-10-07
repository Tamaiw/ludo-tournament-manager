package httpinbound

import (
	"context"
	"net/http"

	"github.com/Tamaiw/ludo-tournament-manager/backend/internal/core/domain"
)

type ctxKey int

const (
	ctxUserKey ctxKey = iota
	ctxTournamentRoleKey
	ctxTournamentKey
)

func putUser(r *http.Request, u *domain.User) *http.Request {
	return r.WithContext(context.WithValue(r.Context(), ctxUserKey, u))
}

// UserFromContext returns the authenticated user from the request context, if any.
func UserFromContext(ctx context.Context) *domain.User {
	if u, ok := ctx.Value(ctxUserKey).(*domain.User); ok {
		return u
	}
	return nil
}

// putTournamentRole stores the actor's role in this tournament on the context.
func putTournamentRole(r *http.Request, role string) *http.Request {
	return r.WithContext(context.WithValue(r.Context(), ctxTournamentRoleKey, role))
}

// TournamentRoleFromContext returns the actor's role on the current tournament.
func TournamentRoleFromContext(ctx context.Context) string {
	if r, ok := ctx.Value(ctxTournamentRoleKey).(string); ok {
		return r
	}
	return ""
}

// PutTournament sets the loaded tournament on the context for downstream handlers.
func PutTournament(r *http.Request, t *domain.Tournament) *http.Request {
	return r.WithContext(context.WithValue(r.Context(), ctxTournamentKey, t))
}

// TournamentFromContext returns the loaded tournament from the request context.
func TournamentFromContext(ctx context.Context) *domain.Tournament {
	if t, ok := ctx.Value(ctxTournamentKey).(*domain.Tournament); ok {
		return t
	}
	return nil
}

// Role constants used by RequireTournamentRole.
const (
	RoleManager         = "manager"
	RolePlayer          = "player"
	RoleSpectatorToken  = "spectator_token"
	RoleAnonymous       = "anonymous"
)

// visibilityAllowed checks whether an anonymous viewer can see a tournament.
func visibilityAllowed(t domain.Tournament, user *domain.User, hasSpectatorToken bool) bool {
	switch t.Visibility {
	case domain.VisibilityPublic, domain.VisibilityUnlisted:
		return true
	case domain.VisibilityPrivate:
		if user != nil {
			return true
		}
		return hasSpectatorToken
	}
	return false
}