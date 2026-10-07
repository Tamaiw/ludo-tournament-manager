package sqlite

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/Tamaiw/ludo-tournament-manager/backend/internal/core/domain"
	"github.com/Tamaiw/ludo-tournament-manager/backend/internal/core/ports"
)

// compile-time port assertions in one place.
var (
	_ ports.UserRepository                    = (*UserRepo)(nil)
	_ ports.TournamentRepository               = (*TournamentRepo)(nil)
	_ ports.TournamentManagerRepository        = (*TournamentManagerRepo)(nil)
	_ ports.TournamentPlayerRepository         = (*TournamentPlayerRepo)(nil)
	_ ports.TournamentSpectatorTokenRepository = (*TournamentSpectatorTokenRepo)(nil)
	_ ports.MatchRepository                    = (*MatchRepo)(nil)
	_ ports.AuditLogRepository                 = (*AuditLogRepo)(nil)
	_ ports.AuthTokenRepository                = (*AuthTokenRepo)(nil)
	_ ports.Transactional                      = (*TxRunner)(nil)
)

// UserRepo persists User rows.
type UserRepo struct{ db *sql.DB }

func (r *UserRepo) Save(ctx context.Context, u domain.User) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO users(id, email, name, password_hash, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, u.ID, u.Email, u.Name, u.PasswordHash, domain.FormatTime(u.CreatedAt), domain.FormatTime(u.UpdatedAt))
	return err
}

func (r *UserRepo) FindByID(ctx context.Context, id domain.UserID) (domain.User, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, email, name, password_hash, created_at, updated_at
		FROM users WHERE id = ?`, id)
	return scanUser(row)
}

func (r *UserRepo) FindByEmail(ctx context.Context, email string) (domain.User, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, email, name, password_hash, created_at, updated_at
		FROM users WHERE email = ?`, email)
	return scanUser(row)
}

func (r *UserRepo) UpdatePasswordHash(ctx context.Context, id domain.UserID, hash string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE users SET password_hash = ?, updated_at = ? WHERE id = ?`,
		hash, domain.FormatTime(time.Now().UTC()), id)
	return err
}

func scanUser(row *sql.Row) (domain.User, error) {
	var (
		id, email, name, hash, created, updated string
	)
	if err := row.Scan(&id, &email, &name, &hash, &created, &updated); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.User{}, domain.ErrUserNotFound
		}
		return domain.User{}, err
	}
	ct, _ := domain.ParseTime(created)
	ut, _ := domain.ParseTime(updated)
	return domain.User{
		ID: domain.UserID(id), Email: email, Name: name, PasswordHash: hash,
		CreatedAt: ct, UpdatedAt: ut,
	}, nil
}

// TournamentRepo persists Tournament rows.
type TournamentRepo struct{ db *sql.DB }

func (r *TournamentRepo) Save(ctx context.Context, t domain.Tournament) error {
	advJSON, _ := json.Marshal(t.PlayersAdvancingPerRound)
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO tournaments(
			id, name, description, format, status,
			min_players_per_match, max_players_per_match,
			players_advancing_per_round,
			scheduled_start_at,
			visibility, registration_mode, max_players,
			created_by, created_at, updated_at,
			started_at, completed_at, cancelled_at, bracket_prng_seed
		) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)
		ON CONFLICT(id) DO UPDATE SET
			name=excluded.name, format=excluded.format, status=excluded.status,
			min_players_per_match=excluded.min_players_per_match,
			max_players_per_match=excluded.max_players_per_match,
			players_advancing_per_round=excluded.players_advancing_per_round,
			scheduled_start_at=excluded.scheduled_start_at,
			visibility=excluded.visibility,
			registration_mode=excluded.registration_mode,
			max_players=excluded.max_players,
			updated_at=excluded.updated_at,
			started_at=excluded.started_at,
			completed_at=excluded.completed_at,
			cancelled_at=excluded.cancelled_at,
			bracket_prng_seed=excluded.bracket_prng_seed
	`,
		t.ID, t.Name, t.Description, t.Format, t.Status,
		t.MinPlayersPerMatch, t.MaxPlayersPerMatch,
		string(advJSON),
		domain.FormatTimePtr(t.ScheduledStartAt),
		t.Visibility, t.RegistrationMode, t.MaxPlayers,
		t.CreatedBy,
		domain.FormatTime(t.CreatedAt), domain.FormatTime(t.UpdatedAt),
		domain.FormatTimePtr(t.StartedAt),
		domain.FormatTimePtr(t.CompletedAt),
		domain.FormatTimePtr(t.CancelledAt),
		domain.FormatInt64Ptr(t.BracketPRNGSeed),
	)
	return err
}

func (r *TournamentRepo) Find(ctx context.Context, id domain.TournamentID) (domain.Tournament, error) {
	row := r.db.QueryRowContext(ctx, tournamentSelect+` WHERE id = ?`, id)
	return scanTournament(row)
}

func (r *TournamentRepo) ListPublic(ctx context.Context) ([]domain.Tournament, error) {
	rows, err := r.db.QueryContext(ctx, tournamentSelect+`
		WHERE visibility = 'public' AND status != 'cancelled'
		ORDER BY COALESCE(started_at,'') DESC, COALESCE(scheduled_start_at,'') DESC, created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Tournament
	for rows.Next() {
		t, err := scanTournament(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (r *TournamentRepo) ListForUser(ctx context.Context, userID domain.UserID) ([]domain.Tournament, error) {
	rows, err := r.db.QueryContext(ctx, tournamentSelect+`
		WHERE id IN (
			SELECT tournament_id FROM tournament_manager WHERE manager_id = ?
			UNION
			SELECT tournament_id FROM tournament_player WHERE player_id = ?
		)
		ORDER BY COALESCE(started_at,'') DESC, COALESCE(scheduled_start_at,'') DESC, created_at DESC`,
		userID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Tournament
	for rows.Next() {
		t, err := scanTournament(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

const tournamentSelect = `
SELECT id, name, description, format, status,
       min_players_per_match, max_players_per_match,
       players_advancing_per_round,
       scheduled_start_at,
       visibility, registration_mode, max_players,
       created_by, created_at, updated_at,
       started_at, completed_at, cancelled_at, bracket_prng_seed
FROM tournaments
`

func scanTournament(s scanner) (domain.Tournament, error) {
	var (
		id, name, description, format, status,
		advJSON,
		scheduled, visibility, regMode,
		createdBy, created, updated,
		started, completed, cancelled, seed string
		minPer, maxPer, maxPlayers            int
	)
	if err := s.Scan(&id, &name, &description, &format, &status,
		&minPer, &maxPer, &advJSON,
		&scheduled, &visibility, &regMode,
		&maxPlayers, &createdBy, &created, &updated,
		&started, &completed, &cancelled, &seed); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.Tournament{}, domain.ErrTournamentNotFound
		}
		return domain.Tournament{}, err
	}
	ct, _ := domain.ParseTime(created)
	ut, _ := domain.ParseTime(updated)
	adv := map[string]int{}
	if advJSON != "" {
		_ = json.Unmarshal([]byte(advJSON), &adv)
	}
	return domain.Tournament{
		ID:                       domain.TournamentID(id),
		Name:                     name,
		Description:              description,
		Format:                   domain.TournamentFormat(format),
		Status:                   domain.TournamentStatus(status),
		MinPlayersPerMatch:       minPer,
		MaxPlayersPerMatch:       maxPer,
		PlayersAdvancingPerRound: adv,
		ScheduledStartAt:         domain.ParseTimePtr(&scheduled),
		Visibility:               domain.Visibility(visibility),
		RegistrationMode:         domain.RegistrationMode(regMode),
		MaxPlayers:               maxPlayers,
		CreatedBy:                domain.UserID(createdBy),
		CreatedAt:                ct,
		UpdatedAt:                ut,
		StartedAt:                domain.ParseTimePtr(&started),
		CompletedAt:              domain.ParseTimePtr(&completed),
		CancelledAt:              domain.ParseTimePtr(&cancelled),
		BracketPRNGSeed:          domain.ParseInt64Ptr(seed),
	}, nil
}

type scanner interface{ Scan(dest ...any) error }

// TournamentManagerRepo persists tournament↔manager join rows.
type TournamentManagerRepo struct{ db *sql.DB }

func (r *TournamentManagerRepo) Add(ctx context.Context, tid domain.TournamentID, mid domain.UserID) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO tournament_manager(tournament_id, manager_id, added_at)
		VALUES (?, ?, ?)
		ON CONFLICT DO NOTHING`,
		tid, mid, domain.FormatTime(time.Now().UTC()))
	return err
}

func (r *TournamentManagerRepo) Remove(ctx context.Context, tid domain.TournamentID, mid domain.UserID) error {
	_, err := r.db.ExecContext(ctx,
		`DELETE FROM tournament_manager WHERE tournament_id = ? AND manager_id = ?`, tid, mid)
	return err
}

func (r *TournamentManagerRepo) ListByTournament(ctx context.Context, tid domain.TournamentID) ([]domain.UserID, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT manager_id FROM tournament_manager WHERE tournament_id = ? ORDER BY added_at`, tid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.UserID
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, domain.UserID(id))
	}
	return out, rows.Err()
}

func (r *TournamentManagerRepo) IsManager(ctx context.Context, tid domain.TournamentID, uid domain.UserID) (bool, error) {
	var count int
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(1) FROM tournament_manager WHERE tournament_id = ? AND manager_id = ?`,
		tid, uid).Scan(&count)
	return count > 0, err
}

// TournamentPlayerRepo persists tournament↔player join rows.
type TournamentPlayerRepo struct{ db *sql.DB }

func (r *TournamentPlayerRepo) Add(ctx context.Context, tid domain.TournamentID, pid domain.UserID) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO tournament_player(tournament_id, player_id, registered_at, seed)
		VALUES (?, ?, ?, NULL)
		ON CONFLICT DO NOTHING`,
		tid, pid, domain.FormatTime(time.Now().UTC()))
	return err
}

func (r *TournamentPlayerRepo) Remove(ctx context.Context, tid domain.TournamentID, pid domain.UserID) error {
	_, err := r.db.ExecContext(ctx,
		`DELETE FROM tournament_player WHERE tournament_id = ? AND player_id = ?`, tid, pid)
	return err
}

func (r *TournamentPlayerRepo) ListByTournament(ctx context.Context, tid domain.TournamentID) ([]domain.TournamentPlayer, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT tournament_id, player_id, registered_at, seed
		FROM tournament_player
		WHERE tournament_id = ?
		ORDER BY registered_at`, tid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.TournamentPlayer
	for rows.Next() {
		var (
			tidStr, pidStr, regStr string
			seed                 sql.NullInt64
		)
		if err := rows.Scan(&tidStr, &pidStr, &regStr, &seed); err != nil {
			return nil, err
		}
		reg, _ := domain.ParseTime(regStr)
		var seedPtr *int
		if seed.Valid {
			v := int(seed.Int64)
			seedPtr = &v
		}
		out = append(out, domain.TournamentPlayer{
			TournamentID: domain.TournamentID(tidStr),
			PlayerID:     domain.UserID(pidStr),
			RegisteredAt: reg,
			Seed:         seedPtr,
		})
	}
	return out, rows.Err()
}

func (r *TournamentPlayerRepo) ListByPlayer(ctx context.Context, pid domain.UserID) ([]domain.TournamentID, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT tournament_id FROM tournament_player WHERE player_id = ?`, pid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.TournamentID
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, domain.TournamentID(id))
	}
	return out, rows.Err()
}

func (r *TournamentPlayerRepo) IsRegistered(ctx context.Context, tid domain.TournamentID, pid domain.UserID) (bool, error) {
	var count int
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(1) FROM tournament_player WHERE tournament_id = ? AND player_id = ?`,
		tid, pid).Scan(&count)
	return count > 0, err
}

func (r *TournamentPlayerRepo) CountRegistered(ctx context.Context, tid domain.TournamentID) (int, error) {
	var n int
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(1) FROM tournament_player WHERE tournament_id = ?`, tid).Scan(&n)
	return n, err
}

func (r *TournamentPlayerRepo) UpdateSeed(ctx context.Context, tid domain.TournamentID, pid domain.UserID, seed int) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE tournament_player SET seed = ? WHERE tournament_id = ? AND player_id = ?`,
		seed, tid, pid)
	return err
}

func (r *TournamentPlayerRepo) SeedUnique(ctx context.Context, tid domain.TournamentID, seed int) (bool, error) {
	var count int
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(1) FROM tournament_player WHERE tournament_id = ? AND seed = ?`,
		tid, seed).Scan(&count)
	return count == 0, err
}

func (r *TournamentPlayerRepo) IsSeedTakenByOther(ctx context.Context, tid domain.TournamentID, pid domain.UserID, seed int) (bool, error) {
	var count int
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(1) FROM tournament_player WHERE tournament_id = ? AND seed = ? AND player_id != ?`,
		tid, seed, pid).Scan(&count)
	return count > 0, err
}

// TournamentSpectatorTokenRepo persists SpectatorToken rows.
type TournamentSpectatorTokenRepo struct{ db *sql.DB }

func (r *TournamentSpectatorTokenRepo) Save(ctx context.Context, t domain.SpectatorToken) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO tournament_spectator_tokens(id, tournament_id, token_hash, label, issued_at, issued_by, revoked_at, revoked_by)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`,
		t.ID, t.TournamentID, t.TokenHash, t.Label,
		domain.FormatTime(t.IssuedAt), t.IssuedBy,
		domain.FormatTimePtr(t.RevokedAt),
		nullableUserID(t.RevokedBy))
	return err
}

func (r *TournamentSpectatorTokenRepo) Find(ctx context.Context, id domain.SpectatorTokenID) (domain.SpectatorToken, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, tournament_id, token_hash, label, issued_at, issued_by, revoked_at, revoked_by
		FROM tournament_spectator_tokens WHERE id = ?`, id)
	return scanSpectatorToken(row)
}

func (r *TournamentSpectatorTokenRepo) FindByHash(ctx context.Context, tid domain.TournamentID, tokenHash string) (domain.SpectatorToken, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, tournament_id, token_hash, label, issued_at, issued_by, revoked_at, revoked_by
		FROM tournament_spectator_tokens WHERE tournament_id = ? AND token_hash = ?`, tid, tokenHash)
	return scanSpectatorToken(row)
}

func (r *TournamentSpectatorTokenRepo) ListByTournament(ctx context.Context, tid domain.TournamentID) ([]domain.SpectatorToken, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, tournament_id, token_hash, label, issued_at, issued_by, revoked_at, revoked_by
		FROM tournament_spectator_tokens WHERE tournament_id = ?
		ORDER BY issued_at DESC`, tid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.SpectatorToken
	for rows.Next() {
		t, err := scanSpectatorToken(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (r *TournamentSpectatorTokenRepo) Revoke(ctx context.Context, id domain.SpectatorTokenID, by domain.UserID, at string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE tournament_spectator_tokens SET revoked_at = ?, revoked_by = ? WHERE id = ? AND revoked_at IS NULL`,
		at, by, id)
	return err
}

func scanSpectatorToken(s scanner) (domain.SpectatorToken, error) {
	var (
		id, tid, hash, label, issuedAt, issuedBy, revokedAt, revokedBy string
	)
	if err := s.Scan(&id, &tid, &hash, &label, &issuedAt, &issuedBy, &revokedAt, &revokedBy); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.SpectatorToken{}, domain.ErrTokenNotFound
		}
		return domain.SpectatorToken{}, err
	}
	iat, _ := domain.ParseTime(issuedAt)
	t := domain.SpectatorToken{
		ID:           domain.SpectatorTokenID(id),
		TournamentID: domain.TournamentID(tid),
		TokenHash:    hash,
		Label:        label,
		IssuedAt:     iat,
		IssuedBy:     domain.UserID(issuedBy),
		RevokedAt:    domain.ParseTimePtr(&revokedAt),
		RevokedBy:    nil,
	}
	if revokedBy != "" {
		t.RevokedBy = &[]domain.UserID{domain.UserID(revokedBy)}[0]
	}
	return t, nil
}

func nullableUserID(u *domain.UserID) any {
	if u == nil {
		return nil
	}
	return string(*u)
}

// MatchRepo persists Match and MatchParticipant rows.
type MatchRepo struct{ db *sql.DB }

func (r *MatchRepo) SaveBatch(ctx context.Context, matches []domain.Match) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, m := range matches {
		_, err := tx.ExecContext(ctx, `
			INSERT INTO matches(id, tournament_id, round, position_in_round, status, scheduled_at,
			                    next_match_id, slot_in_next_match, created_at, updated_at)
			VALUES (?,?,?,?,?,?,?,?,?,?)
		`,
			m.ID, m.TournamentID, m.Round, m.PositionInRound, m.Status,
			domain.FormatTimePtr(m.ScheduledAt),
			nullableMatchID(m.NextMatchID),
			nullableSlot(m.SlotInNextMatch),
			domain.FormatTime(m.CreatedAt), domain.FormatTime(m.UpdatedAt))
		if err != nil {
			return err
		}
	}
	return tx.Commit()
}

func nullableMatchID(m *domain.MatchID) any {
	if m == nil {
		return nil
	}
	return string(*m)
}

func nullableSlot(s *domain.Slot) any {
	if s == nil {
		return nil
	}
	return string(*s)
}

func (r *MatchRepo) Find(ctx context.Context, id domain.MatchID) (domain.Match, error) {
	row := r.db.QueryRowContext(ctx, matchSelect+` WHERE id = ?`, id)
	return scanMatch(row)
}

func (r *MatchRepo) FindWithParticipants(ctx context.Context, id domain.MatchID) (domain.Match, []domain.MatchParticipant, error) {
	m, err := r.Find(ctx, id)
	if err != nil {
		return domain.Match{}, nil, err
	}
	parts, err := r.participantsByMatch(ctx, []domain.MatchID{id})
	if err != nil {
		return domain.Match{}, nil, err
	}
	return m, parts[id], nil
}

func (r *MatchRepo) ListByTournament(ctx context.Context, tid domain.TournamentID) ([]domain.Match, error) {
	rows, err := r.db.QueryContext(ctx, matchSelect+` WHERE tournament_id = ? ORDER BY round, position_in_round`, tid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Match
	for rows.Next() {
		m, err := scanMatch(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (r *MatchRepo) ListByTournamentWithParticipants(ctx context.Context, tid domain.TournamentID) ([]domain.Match, []domain.MatchParticipant, error) {
	matches, err := r.ListByTournament(ctx, tid)
	if err != nil {
		return nil, nil, err
	}
	ids := make([]domain.MatchID, len(matches))
	for i, m := range matches {
		ids[i] = m.ID
	}
	parts, err := r.participantsByMatch(ctx, ids)
	if err != nil {
		return nil, nil, err
	}
	var flat []domain.MatchParticipant
	for _, m := range matches {
		flat = append(flat, parts[m.ID]...)
	}
	return matches, flat, nil
}

func (r *MatchRepo) participantsByMatch(ctx context.Context, ids []domain.MatchID) (map[domain.MatchID][]domain.MatchParticipant, error) {
	if len(ids) == 0 {
		return map[domain.MatchID][]domain.MatchParticipant{}, nil
	}
	args := make([]any, len(ids))
	placeholders := ""
	for i, id := range ids {
		if i > 0 {
			placeholders += ","
		}
		placeholders += "?"
		args[i] = id
	}
	rows, err := r.db.QueryContext(ctx, `
		SELECT match_id, slot, player_id, advancing_position, is_bye
		FROM match_participants
		WHERE match_id IN (`+placeholders+`)
		ORDER BY match_id, slot`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[domain.MatchID][]domain.MatchParticipant{}
	for rows.Next() {
		var (
			mid, slot string
			pid       sql.NullString
			pos       sql.NullInt64
			isBye     int
		)
		if err := rows.Scan(&mid, &slot, &pid, &pos, &isBye); err != nil {
			return nil, err
		}
		var (
			pidPtr *domain.UserID
			posPtr *int
		)
		if pid.Valid && pid.String != "" {
			uid := domain.UserID(pid.String)
			pidPtr = &uid
		}
		if pos.Valid {
			v := int(pos.Int64)
			posPtr = &v
		}
		key := domain.MatchID(mid)
		out[key] = append(out[key], domain.MatchParticipant{
			MatchID:           key,
			Slot:              domain.Slot(slot),
			PlayerID:          pidPtr,
			AdvancingPosition: posPtr,
			IsBye:             isBye != 0,
		})
	}
	return out, rows.Err()
}

func (r *MatchRepo) UpdateStatus(ctx context.Context, id domain.MatchID, status domain.MatchStatus, updatedAt string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE matches SET status = ?, updated_at = ? WHERE id = ?`,
		status, updatedAt, id)
	return err
}

func (r *MatchRepo) UpdateParticipantPosition(ctx context.Context, mid domain.MatchID, slot domain.Slot, position int) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE match_participants SET advancing_position = ? WHERE match_id = ? AND slot = ?`,
		position, mid, slot)
	return err
}

func (r *MatchRepo) UpsertParticipant(ctx context.Context, mid domain.MatchID, slot domain.Slot, playerID domain.UserID) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO match_participants(match_id, slot, player_id, is_bye)
		VALUES (?, ?, ?, 0)
		ON CONFLICT DO NOTHING`,
		mid, slot, playerID)
	return err
}

func (r *MatchRepo) GetDownstreamChain(ctx context.Context, id domain.MatchID) ([]domain.Match, error) {
	rows, err := r.db.QueryContext(ctx, `
		WITH RECURSIVE chain(id) AS (
			SELECT next_match_id FROM matches WHERE id = ? AND next_match_id IS NOT NULL
			UNION ALL
			SELECT m.next_match_id FROM matches m JOIN chain c ON m.id = c.id WHERE m.next_match_id IS NOT NULL
		)
		SELECT id, tournament_id, round, position_in_round, status, scheduled_at,
		       next_match_id, slot_in_next_match, created_at, updated_at
		FROM matches WHERE id IN (SELECT id FROM chain)
	`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Match
	for rows.Next() {
		m, err := scanMatch(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (r *MatchRepo) GetUpstreamMatches(ctx context.Context, id domain.MatchID) ([]domain.Match, error) {
	rows, err := r.db.QueryContext(ctx, `
		WITH RECURSIVE chain(id) AS (
			SELECT id FROM matches WHERE next_match_id = ?
			UNION ALL
			SELECT m.id FROM matches m JOIN chain c ON m.next_match_id = c.id
		)
		SELECT id, tournament_id, round, position_in_round, status, scheduled_at,
		       next_match_id, slot_in_next_match, created_at, updated_at
		FROM matches WHERE id IN (SELECT id FROM chain)
	`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Match
	for rows.Next() {
		m, err := scanMatch(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

const matchSelect = `
SELECT id, tournament_id, round, position_in_round, status, scheduled_at,
       next_match_id, slot_in_next_match, created_at, updated_at
FROM matches
`

func scanMatch(s scanner) (domain.Match, error) {
	var (
		id, tid, status, createdAt, updatedAt string
		round, pos                            int
		scheduled                             sql.NullString
		nextID                                sql.NullString
		nextSlot                              sql.NullString
	)
	if err := s.Scan(&id, &tid, &round, &pos, &status, &scheduled,
		&nextID, &nextSlot, &createdAt, &updatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.Match{}, domain.ErrMatchNotFound
		}
		return domain.Match{}, err
	}
	ct, _ := domain.ParseTime(createdAt)
	ut, _ := domain.ParseTime(updatedAt)
	m := domain.Match{
		ID:              domain.MatchID(id),
		TournamentID:    domain.TournamentID(tid),
		Round:           round,
		PositionInRound: pos,
		Status:          domain.MatchStatus(status),
		ScheduledAt:     nil,
		CreatedAt:       ct,
		UpdatedAt:       ut,
	}
	if scheduled.Valid && scheduled.String != "" {
		t, _ := domain.ParseTime(scheduled.String)
		m.ScheduledAt = &t
	}
	if nextID.Valid && nextID.String != "" {
		v := domain.MatchID(nextID.String)
		m.NextMatchID = &v
	}
	if nextSlot.Valid && nextSlot.String != "" {
		v := domain.Slot(nextSlot.String)
		m.SlotInNextMatch = &v
	}
	return m, nil
}

// MatchParticipantRepo persists match_participants rows.
type MatchParticipantRepo struct{ db *sql.DB }

func (r *MatchParticipantRepo) InsertBatch(ctx context.Context, parts []domain.MatchParticipant) error {
	if len(parts) == 0 {
		return nil
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, p := range parts {
		var pid any = nil
		if p.PlayerID != nil {
			pid = *p.PlayerID
		}
		_, err := tx.ExecContext(ctx, `
			INSERT INTO match_participants(match_id, slot, player_id, advancing_position, is_bye)
			VALUES (?, ?, ?, ?, ?)
			ON CONFLICT DO NOTHING`,
			p.MatchID, p.Slot, pid,
			domain.FormatIntPtr(p.AdvancingPosition),
			boolToInt(p.IsBye))
		if err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (r *MatchParticipantRepo) RemoveAllForMatch(ctx context.Context, mid domain.MatchID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM match_participants WHERE match_id = ?`, mid)
	return err
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// AuditLogRepo persists tournament_audit_log rows.
type AuditLogRepo struct{ db *sql.DB }

func (r *AuditLogRepo) Insert(ctx context.Context, e domain.AuditLogEntry) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO tournament_audit_log(id, tournament_id, actor_id, action, subject_id, before, after, recorded_at)
		VALUES (?,?,?,?,?,?,?,?)
	`, e.ID, e.TournamentID, nullableUserID(e.ActorID), e.Action, e.SubjectID, e.Before, e.After, domain.FormatTime(e.RecordedAt))
	return err
}

func (r *AuditLogRepo) ListByTournament(ctx context.Context, tid domain.TournamentID, limit, offset int) ([]domain.AuditLogEntry, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, tournament_id, actor_id, action, subject_id, before, after, recorded_at
		FROM tournament_audit_log
		WHERE tournament_id = ?
		ORDER BY recorded_at DESC, id DESC
		LIMIT ? OFFSET ?`, tid, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.AuditLogEntry
	for rows.Next() {
		var (
			id, tidStr, action, subject, recorded string
			actor                              sql.NullString
			before, after                     sql.NullString
		)
		if err := rows.Scan(&id, &tidStr, &actor, &action, &subject, &before, &after, &recorded); err != nil {
			return nil, err
		}
		rec, _ := domain.ParseTime(recorded)
		e := domain.AuditLogEntry{
			ID:           domain.AuditLogID(id),
			TournamentID: domain.TournamentID(tidStr),
			Action:       domain.AuditAction(action),
			SubjectID:    subject,
			RecordedAt:   rec,
		}
		if actor.Valid && actor.String != "" {
			uid := domain.UserID(actor.String)
			e.ActorID = &uid
		}
		if before.Valid {
			e.Before = before.String
		}
		if after.Valid {
			e.After = after.String
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// AuthTokenRepo persists auth_tokens rows.
type AuthTokenRepo struct{ db *sql.DB }

func (r *AuthTokenRepo) Save(ctx context.Context, t domain.AuthToken) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO auth_tokens(id, kind, user_id, email, token_hash, expires_at, used_at, created_by, created_at)
		VALUES (?,?,?,?,?,?,?,?,?)
	`, t.ID, t.Kind, nullableUserID(t.UserID), t.Email, t.TokenHash,
		domain.FormatTime(t.ExpiresAt),
		domain.FormatTimePtr(t.UsedAt),
		nullableUserID(t.CreatedBy),
		domain.FormatTime(t.CreatedAt))
	return err
}

func (r *AuthTokenRepo) FindByHash(ctx context.Context, kind domain.AuthTokenKind, tokenHash string) (domain.AuthToken, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, kind, user_id, email, token_hash, expires_at, used_at, created_by, created_at
		FROM auth_tokens WHERE kind = ? AND token_hash = ?`, kind, tokenHash)
	return scanAuthToken(row)
}

func (r *AuthTokenRepo) MarkUsed(ctx context.Context, id domain.AuthTokenID, uid domain.UserID, usedAt string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE auth_tokens SET user_id = ?, used_at = ? WHERE id = ?`,
		uid, usedAt, id)
	return err
}

func scanAuthToken(row *sql.Row) (domain.AuthToken, error) {
	var (
		id, kind, email, hash, expires, created string
		userID, usedAt, createdBy               sql.NullString
	)
	if err := row.Scan(&id, &kind, &userID, &email, &hash, &expires, &usedAt, &createdBy, &created); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.AuthToken{}, domain.ErrTokenNotFound
		}
		return domain.AuthToken{}, err
	}
	ex, _ := domain.ParseTime(expires)
	ca, _ := domain.ParseTime(created)
	t := domain.AuthToken{
		ID:        domain.AuthTokenID(id),
		Kind:      domain.AuthTokenKind(kind),
		Email:     email,
		TokenHash: hash,
		ExpiresAt: ex,
		CreatedAt: ca,
	}
	if userID.Valid && userID.String != "" {
		v := domain.UserID(userID.String)
		t.UserID = &v
	}
	t.UsedAt = domain.ParseTimePtr(&usedAt.String)
	if createdBy.Valid && createdBy.String != "" {
		v := domain.UserID(createdBy.String)
		t.CreatedBy = &v
	}
	return t, nil
}

// TxRunner wraps SQLite in transactions.
type TxRunner struct{ db *sql.DB }

func (t *TxRunner) RunInTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	tx, err := t.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := fn(ctx); err != nil {
		return err
	}
	return tx.Commit()
}

// HashToken returns the SHA-256 hex digest of a token. Used for both spectator
// tokens and auth tokens (invite / password reset).
func HashToken(plaintext string) string {
	sum := sha256.Sum256([]byte(plaintext))
	return hex.EncodeToString(sum[:])
}

// HashSessionCookie returns the SHA-256 hex digest of a session cookie value,
// suitable for scs's HashTokenInStore option.
func HashSessionCookie(value string) string {
	return HashToken(value)
}

// NewToken returns a fresh 256-bit opaque token (base64url, no padding).
func NewToken() string {
	b := make([]byte, 32)
	if _, err := crandRead(b); err != nil {
		panic(fmt.Errorf("rand: %w", err))
	}
	return base64.RawURLEncoding.EncodeToString(b)
}

var crandRead = cryptoRandRead

// Export for reuse from other adapters if needed.
func init() {}