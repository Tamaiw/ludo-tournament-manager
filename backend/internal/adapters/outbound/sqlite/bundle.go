// Package sqlite contains the SQLite outbound adapter for the Ludo Tournament Manager.
package sqlite

import (
	"database/sql"
)

// RepoBundle bundles every sqlite adapter behind one parameter object so
// main.go and tests don't have to spell out each repo.
type RepoBundle struct {
	DB                        *sql.DB
	Users                     *UserRepo
	Tournaments               *TournamentRepo
	TournamentManagers        *TournamentManagerRepo
	TournamentPlayers         *TournamentPlayerRepo
	TournamentSpectatorTokens *TournamentSpectatorTokenRepo
	Matches                   *MatchRepo
	MatchParticipants         *MatchParticipantRepo
	AuditLog                  *AuditLogRepo
	AuthTokens                *AuthTokenRepo
	Sessions                  *SessionStore
	Tx                        *TxRunner
}

// NewRepos constructs the full bundle from a *sql.DB.
func NewRepos(db *sql.DB) *RepoBundle {
	return &RepoBundle{
		DB:                        db,
		Users:                     &UserRepo{db: db},
		Tournaments:               &TournamentRepo{db: db},
		TournamentManagers:        &TournamentManagerRepo{db: db},
		TournamentPlayers:         &TournamentPlayerRepo{db: db},
		TournamentSpectatorTokens: &TournamentSpectatorTokenRepo{db: db},
		Matches:                   &MatchRepo{db: db},
		MatchParticipants:         &MatchParticipantRepo{db: db},
		AuditLog:                  &AuditLogRepo{db: db},
		AuthTokens:                &AuthTokenRepo{db: db},
		Sessions:                  &SessionStore{db: db},
		Tx:                        &TxRunner{db: db},
	}
}
