package ports

import (
	"context"

	"github.com/Tamaiw/ludo-tournament-manager/backend/internal/core/domain"
)

// UserRepository persists User rows.
type UserRepository interface {
	Save(ctx context.Context, u domain.User) error
	FindByID(ctx context.Context, id domain.UserID) (domain.User, error)
	FindByEmail(ctx context.Context, email string) (domain.User, error)
	UpdatePasswordHash(ctx context.Context, id domain.UserID, hash string) error
}

// TournamentRepository persists Tournament rows.
type TournamentRepository interface {
	Save(ctx context.Context, t domain.Tournament) error
	Find(ctx context.Context, id domain.TournamentID) (domain.Tournament, error)
	ListPublic(ctx context.Context) ([]domain.Tournament, error)
	ListForUser(ctx context.Context, userID domain.UserID) ([]domain.Tournament, error)
}

// TournamentManagerRepository persists tournament↔manager join rows.
type TournamentManagerRepository interface {
	Add(ctx context.Context, tournamentID domain.TournamentID, managerID domain.UserID) error
	Remove(ctx context.Context, tournamentID domain.TournamentID, managerID domain.UserID) error
	ListByTournament(ctx context.Context, tournamentID domain.TournamentID) ([]domain.UserID, error)
	IsManager(ctx context.Context, tournamentID domain.TournamentID, userID domain.UserID) (bool, error)
}

// TournamentPlayerRepository persists tournament↔player join rows.
type TournamentPlayerRepository interface {
	Add(ctx context.Context, tournamentID domain.TournamentID, playerID domain.UserID) error
	Remove(ctx context.Context, tournamentID domain.TournamentID, playerID domain.UserID) error
	ListByTournament(ctx context.Context, tournamentID domain.TournamentID) ([]domain.TournamentPlayer, error)
	ListByPlayer(ctx context.Context, playerID domain.UserID) ([]domain.TournamentID, error)
	IsRegistered(ctx context.Context, tournamentID domain.TournamentID, playerID domain.UserID) (bool, error)
	CountRegistered(ctx context.Context, tournamentID domain.TournamentID) (int, error)
	UpdateSeed(ctx context.Context, tournamentID domain.TournamentID, playerID domain.UserID, seed int) error
	SeedUnique(ctx context.Context, tournamentID domain.TournamentID, seed int) (bool, error)
	IsSeedTakenByOther(ctx context.Context, tournamentID domain.TournamentID, playerID domain.UserID, seed int) (bool, error)
}

// TournamentSpectatorTokenRepository persists SpectatorToken rows.
type TournamentSpectatorTokenRepository interface {
	Save(ctx context.Context, t domain.SpectatorToken) error
	Find(ctx context.Context, id domain.SpectatorTokenID) (domain.SpectatorToken, error)
	FindByHash(ctx context.Context, tournamentID domain.TournamentID, tokenHash string) (domain.SpectatorToken, error)
	ListByTournament(ctx context.Context, tournamentID domain.TournamentID) ([]domain.SpectatorToken, error)
	Revoke(ctx context.Context, id domain.SpectatorTokenID, revokedBy domain.UserID, at string) error
}

// MatchRepository persists Match and MatchParticipant rows.
type MatchRepository interface {
	SaveBatch(ctx context.Context, matches []domain.Match) error
	Find(ctx context.Context, id domain.MatchID) (domain.Match, error)
	FindWithParticipants(ctx context.Context, id domain.MatchID) (domain.Match, []domain.MatchParticipant, error)
	ListByTournament(ctx context.Context, tournamentID domain.TournamentID) ([]domain.Match, error)
	ListByTournamentWithParticipants(ctx context.Context, tournamentID domain.TournamentID) ([]domain.Match, []domain.MatchParticipant, error)
	UpdateStatus(ctx context.Context, id domain.MatchID, status domain.MatchStatus, updatedAt string) error
	UpdateParticipantPosition(ctx context.Context, matchID domain.MatchID, slot domain.Slot, position int) error
	UpsertParticipant(ctx context.Context, matchID domain.MatchID, slot domain.Slot, playerID domain.UserID) error
	GetDownstreamChain(ctx context.Context, matchID domain.MatchID) ([]domain.Match, error)
	GetUpstreamMatches(ctx context.Context, matchID domain.MatchID) ([]domain.Match, error)
}

// MatchParticipantRepository persists match_participants rows.
type MatchParticipantRepository interface {
	InsertBatch(ctx context.Context, parts []domain.MatchParticipant) error
	RemoveAllForMatch(ctx context.Context, mid domain.MatchID) error
}

// AuditLogRepository persists tournament audit log rows.
type AuditLogRepository interface {
	Insert(ctx context.Context, entry domain.AuditLogEntry) error
	ListByTournament(ctx context.Context, tournamentID domain.TournamentID, limit, offset int) ([]domain.AuditLogEntry, error)
}

// AuthTokenRepository persists auth_tokens rows.
type AuthTokenRepository interface {
	Save(ctx context.Context, t domain.AuthToken) error
	FindByHash(ctx context.Context, kind domain.AuthTokenKind, tokenHash string) (domain.AuthToken, error)
	MarkUsed(ctx context.Context, id domain.AuthTokenID, userID domain.UserID, usedAt string) error
}

// SessionStore is a port over the cookie session backing store.
type SessionStore interface {
	// scs.Store methods (we embed this via the scs adapter in production code).
}

// Transactional exposes a single port wrapping SQLite for service-layer writes
// that need atomicity. Concrete adapter writes the tx begin/commit.
type Transactional interface {
	RunInTransaction(ctx context.Context, fn func(ctx context.Context) error) error
}