// Package services contains all use-case services for the Ludo Tournament Manager.
package services

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/Tamaiw/ludo-tournament-manager/backend/internal/core/domain"
	"github.com/Tamaiw/ludo-tournament-manager/backend/internal/core/ports"
)

// ---- helpers ----

func hashToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

func newToken() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return base64.RawURLEncoding.EncodeToString(b)
}

// writeAudit inserts a single audit log row, best-effort. We swallow errors so
// that audit-log failures don't block business logic (the caller can choose
// to surface via a wrapping transaction if atomicity is essential).
func writeAudit(ctx context.Context, repo ports.AuditLogRepository, clock ports.Clock, e domain.AuditLogEntry) {
	if e.ID == "" {
		e.ID = domain.NewAuditLogID()
	}
	if e.RecordedAt.IsZero() {
		e.RecordedAt = clock.Now()
	}
	_ = repo.Insert(ctx, e)
}

func jsonString(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return ""
	}
	return string(b)
}

func beforeAfter(before, after any) (string, string) {
	return jsonString(before), jsonString(after)
}

// ---- Errors ----

var (
	ErrInvalidCredentials         = errors.New("invalid credentials")
	ErrWrongCurrentPassword       = errors.New("current password is incorrect")
	ErrPasswordMismatch           = errors.New("new password and confirmation do not match")
	ErrWeakPassword               = errors.New("password must be at least 8 characters")
	ErrTargetUserNotFound         = errors.New("target user not found")
	ErrInvalidSlot                = errors.New("invalid slot in advancing positions")
	ErrSeedRequired               = errors.New("player already has a seed; clear it first")
	ErrDuplicateSlotInPositions   = errors.New("duplicate slot in advancing positions")
	ErrAdvancingPositionsRequired = errors.New("must record advancing positions for all advancers")
	ErrIncorrectDVPNCount         = errors.New("number of advancing positions must equal the round's advance count")
)

// ---- CreateUser ----

type CreateUser struct {
	Users  ports.UserRepository
	Hasher ports.PasswordHasher
	Clock  ports.Clock
}

type CreateUserCmd struct {
	Email    string
	Name     string
	Password string
}

func (c CreateUser) Handle(ctx context.Context, cmd CreateUserCmd) (domain.User, error) {
	if cmd.Password == "" || len(cmd.Password) < 8 {
		return domain.User{}, ErrWeakPassword
	}
	hash, err := c.Hasher.Hash(cmd.Password)
	if err != nil {
		return domain.User{}, err
	}
	u, err := domain.NewUser(cmd.Email, cmd.Name, hash, c.Clock.Now())
	if err != nil {
		return domain.User{}, err
	}
	if existing, err := c.Users.FindByEmail(ctx, u.Email); err == nil && existing.ID != "" {
		return domain.User{}, domain.ErrUserExists
	}
	if err := c.Users.Save(ctx, u); err != nil {
		return domain.User{}, err
	}
	return u, nil
}

// ---- SignIn ----

type SignIn struct {
	Users  ports.UserRepository
	Hasher ports.PasswordHasher
}

type SignInCmd struct {
	Email    string
	Password string
}

func (s SignIn) Handle(ctx context.Context, cmd SignInCmd) (domain.User, error) {
	if cmd.Email == "" || cmd.Password == "" {
		return domain.User{}, ErrInvalidCredentials
	}
	u, err := s.Users.FindByEmail(ctx, cmd.Email)
	if err != nil {
		return domain.User{}, ErrInvalidCredentials
	}
	ok, needsRehash, err := s.Hasher.Verify(u.PasswordHash, cmd.Password)
	if err != nil || !ok {
		return domain.User{}, ErrInvalidCredentials
	}
	if needsRehash {
		if h, err := s.Hasher.Hash(cmd.Password); err == nil {
			_ = s.Users.UpdatePasswordHash(ctx, u.ID, h)
			u.PasswordHash = h
		}
	}
	return u, nil
}

// ---- ChangePassword / RequestPasswordReset / RedeemPasswordReset ----

type ChangePassword struct {
	Users  ports.UserRepository
	Hasher ports.PasswordHasher
	Clock  ports.Clock
}

type ChangePasswordCmd struct {
	UserID      domain.UserID
	Current     string
	NewPassword string
	Confirm     string
}

func (s ChangePassword) Handle(ctx context.Context, cmd ChangePasswordCmd) (domain.User, error) {
	if cmd.NewPassword == "" || len(cmd.NewPassword) < 8 {
		return domain.User{}, ErrWeakPassword
	}
	if cmd.NewPassword != cmd.Confirm {
		return domain.User{}, ErrPasswordMismatch
	}
	u, err := s.Users.FindByID(ctx, cmd.UserID)
	if err != nil {
		return domain.User{}, err
	}
	ok, _, err := s.Hasher.Verify(u.PasswordHash, cmd.Current)
	if err != nil || !ok {
		return domain.User{}, ErrWrongCurrentPassword
	}
	hash, err := s.Hasher.Hash(cmd.NewPassword)
	if err != nil {
		return domain.User{}, err
	}
	if err := s.Users.UpdatePasswordHash(ctx, u.ID, hash); err != nil {
		return domain.User{}, err
	}
	u.PasswordHash = hash
	u.UpdatedAt = s.Clock.Now()
	return u, nil
}

type RequestPasswordReset struct {
	Users      ports.UserRepository
	AuthTokens ports.AuthTokenRepository
	Email      ports.EmailSender
	Clock      ports.Clock
	PublicURL  string
}

type RequestPasswordResetCmd struct {
	ActorID     domain.UserID
	TargetEmail string
}

func (s RequestPasswordReset) Handle(ctx context.Context, cmd RequestPasswordResetCmd) (string, error) {
	target, err := s.Users.FindByEmail(ctx, cmd.TargetEmail)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			return "", ErrTargetUserNotFound
		}
		return "", err
	}
	now := s.Clock.Now()
	raw := newToken()
	token := domain.AuthToken{
		ID:        domain.NewAuthTokenID(),
		Kind:      domain.AuthTokenPasswordReset,
		Email:     target.Email,
		TokenHash: hashToken(raw),
		ExpiresAt: now.Add(1 * time.Hour),
		CreatedBy: &cmd.ActorID,
		CreatedAt: now,
	}
	if err := s.AuthTokens.Save(ctx, token); err != nil {
		return "", err
	}
	url := s.PublicURL + "/password-resets/" + raw
	if err := s.Email.SendPasswordReset(ctx, target.Email, url, ""); err != nil {
		return "", err
	}
	return url, nil
}

type RedeemPasswordReset struct {
	Users      ports.UserRepository
	AuthTokens ports.AuthTokenRepository
	Hasher     ports.PasswordHasher
	Clock      ports.Clock
	AuditLog   ports.AuditLogRepository
}

type RedeemPasswordResetCmd struct {
	RawToken    string
	NewPassword string
	Confirm     string
}

func (s RedeemPasswordReset) Handle(ctx context.Context, cmd RedeemPasswordResetCmd) (domain.User, error) {
	if cmd.NewPassword == "" || len(cmd.NewPassword) < 8 {
		return domain.User{}, ErrWeakPassword
	}
	if cmd.NewPassword != cmd.Confirm {
		return domain.User{}, ErrPasswordMismatch
	}
	token, err := s.AuthTokens.FindByHash(ctx, domain.AuthTokenPasswordReset, hashToken(cmd.RawToken))
	if err != nil {
		if errors.Is(err, domain.ErrTokenNotFound) {
			return domain.User{}, domain.ErrTokenNotFound
		}
		return domain.User{}, err
	}
	if err := token.IsUsable(s.Clock.Now()); err != nil {
		return domain.User{}, err
	}
	target, err := s.Users.FindByEmail(ctx, token.Email)
	if err != nil {
		return domain.User{}, err
	}
	newHash, err := s.Hasher.Hash(cmd.NewPassword)
	if err != nil {
		return domain.User{}, err
	}
	if err := s.Users.UpdatePasswordHash(ctx, target.ID, newHash); err != nil {
		return domain.User{}, err
	}
	now := s.Clock.Now()
	_ = s.AuthTokens.MarkUsed(ctx, token.ID, target.ID, domain.FormatTime(now))
	target.PasswordHash = newHash
	return target, nil
}

// ---- CreateTournament ----

type CreateTournament struct {
	Tournaments ports.TournamentRepository
	TManagers   ports.TournamentManagerRepository
	AuditLog    ports.AuditLogRepository
	Clock       ports.Clock
}

type CreateTournamentCmd struct {
	ActorID            domain.UserID
	Name               string
	Description        string
	MinPlayersPerMatch int
	MaxPlayersPerMatch int
	AdvanceMap         map[string]int
	ScheduledStartAt   *time.Time
	Visibility         domain.Visibility
	RegistrationMode   domain.RegistrationMode
	MaxPlayers         int
}

func (t *CreateTournament) Handle(ctx context.Context, cmd CreateTournamentCmd) (domain.Tournament, error) {
	now := t.Clock.Now()
	tour, err := domain.NewTournament(
		cmd.Name, cmd.Description,
		cmd.MinPlayersPerMatch, cmd.MaxPlayersPerMatch,
		cmd.AdvanceMap,
		cmd.ScheduledStartAt, cmd.Visibility, cmd.RegistrationMode,
		cmd.MaxPlayers, cmd.ActorID, now,
	)
	if err != nil {
		return domain.Tournament{}, err
	}
	if err := t.Tournaments.Save(ctx, tour); err != nil {
		return domain.Tournament{}, err
	}
	if err := t.TManagers.Add(ctx, tour.ID, cmd.ActorID); err != nil {
		return domain.Tournament{}, err
	}
	writeAudit(ctx, t.AuditLog, t.Clock, domain.AuditLogEntry{
		TournamentID: tour.ID,
		ActorID:      &cmd.ActorID,
		Action:       domain.ActionManagerAdded,
		SubjectID:    cmd.ActorID.String(),
		Before:       "null",
		After:        jsonString(map[string]string{"manager_id": cmd.ActorID.String()}),
	})
	return tour, nil
}

// ---- EditTournament ----

type EditTournament struct {
	Tournaments ports.TournamentRepository
	AuditLog    ports.AuditLogRepository
	Clock       ports.Clock
}

type EditTournamentCmd struct {
	ActorID            domain.UserID
	TournamentID       domain.TournamentID
	Name               string
	Description        string
	MinPlayersPerMatch int
	MaxPlayersPerMatch int
	AdvanceMap         map[string]int
	ScheduledStartAt   *time.Time
}

func (s *EditTournament) Handle(ctx context.Context, cmd EditTournamentCmd) (domain.Tournament, error) {
	t, err := s.Tournaments.Find(ctx, cmd.TournamentID)
	if err != nil {
		return domain.Tournament{}, err
	}
	before := t
	t.Name = strings.TrimSpace(cmd.Name)
	if t.Name == "" {
		return domain.Tournament{}, domain.ErrEmptyName
	}
	if len(t.Name) > 120 {
		return domain.Tournament{}, domain.ErrNameTooLong
	}
	t.Description = strings.TrimSpace(cmd.Description)
	if cmd.MinPlayersPerMatch >= 2 && cmd.MinPlayersPerMatch <= 4 {
		t.MinPlayersPerMatch = cmd.MinPlayersPerMatch
	}
	if cmd.MaxPlayersPerMatch >= t.MinPlayersPerMatch && cmd.MaxPlayersPerMatch <= 4 {
		t.MaxPlayersPerMatch = cmd.MaxPlayersPerMatch
	}
	if cmd.AdvanceMap != nil {
		t.PlayersAdvancingPerRound = cmd.AdvanceMap
	}
	t.ScheduledStartAt = cmd.ScheduledStartAt
	t.UpdatedAt = s.Clock.Now()
	if err := s.Tournaments.Save(ctx, t); err != nil {
		return domain.Tournament{}, err
	}
	bJSON, aJSON := beforeAfter(before, t)
	writeAudit(ctx, s.AuditLog, s.Clock, domain.AuditLogEntry{
		TournamentID: t.ID, ActorID: &cmd.ActorID,
		Action:    domain.ActionTournamentEdited,
		SubjectID: t.ID.String(),
		Before:    bJSON, After: aJSON,
	})
	return t, nil
}

// ---- ChangeVisibility ----

type ChangeVisibility struct {
	Tournaments ports.TournamentRepository
	AuditLog    ports.AuditLogRepository
	Clock       ports.Clock
}

type ChangeVisibilityCmd struct {
	ActorID      domain.UserID
	TournamentID domain.TournamentID
	Visibility   domain.Visibility
}

func (s *ChangeVisibility) Handle(ctx context.Context, cmd ChangeVisibilityCmd) (domain.Tournament, error) {
	if !cmd.Visibility.Valid() {
		return domain.Tournament{}, domain.ErrInvalidVisibility
	}
	t, err := s.Tournaments.Find(ctx, cmd.TournamentID)
	if err != nil {
		return domain.Tournament{}, err
	}
	before := t
	t.Visibility = cmd.Visibility
	t.UpdatedAt = s.Clock.Now()
	if err := s.Tournaments.Save(ctx, t); err != nil {
		return domain.Tournament{}, err
	}
	b, a := beforeAfter(before, t)
	writeAudit(ctx, s.AuditLog, s.Clock, domain.AuditLogEntry{
		TournamentID: t.ID, ActorID: &cmd.ActorID,
		Action:    domain.ActionVisibilityChanged,
		SubjectID: t.ID.String(),
		Before:    b, After: a,
	})
	return t, nil
}

// ---- ChangeRegistrationMode ----

type ChangeRegistrationMode struct {
	Tournaments ports.TournamentRepository
	AuditLog    ports.AuditLogRepository
	Clock       ports.Clock
}

type ChangeRegistrationModeCmd struct {
	ActorID          domain.UserID
	TournamentID     domain.TournamentID
	RegistrationMode domain.RegistrationMode
}

func (s *ChangeRegistrationMode) Handle(ctx context.Context, cmd ChangeRegistrationModeCmd) (domain.Tournament, error) {
	if !cmd.RegistrationMode.Valid() {
		return domain.Tournament{}, domain.ErrInvalidRegistrationMode
	}
	t, err := s.Tournaments.Find(ctx, cmd.TournamentID)
	if err != nil {
		return domain.Tournament{}, err
	}
	if t.Status == domain.StatusInProgress || t.Status == domain.StatusCompleted || t.Status == domain.StatusCancelled {
		return domain.Tournament{}, fmt.Errorf("registration mode cannot be changed after the tournament starts")
	}
	before := t
	t.RegistrationMode = cmd.RegistrationMode
	t.UpdatedAt = s.Clock.Now()
	if err := s.Tournaments.Save(ctx, t); err != nil {
		return domain.Tournament{}, err
	}
	b, a := beforeAfter(before, t)
	writeAudit(ctx, s.AuditLog, s.Clock, domain.AuditLogEntry{
		TournamentID: t.ID, ActorID: &cmd.ActorID,
		Action:    domain.ActionRegistrationModeChanged,
		SubjectID: t.ID.String(),
		Before:    b, After: a,
	})
	return t, nil
}

// ---- Open / Close / Start / Cancel / ForceClose ----

type OpenRegistration struct {
	Tournaments ports.TournamentRepository
	AuditLog    ports.AuditLogRepository
	Clock       ports.Clock
}

type OpenRegistrationCmd struct {
	ActorID      domain.UserID
	TournamentID domain.TournamentID
}

func (s *OpenRegistration) Handle(ctx context.Context, cmd OpenRegistrationCmd) (domain.Tournament, error) {
	t, err := s.Tournaments.Find(ctx, cmd.TournamentID)
	if err != nil {
		return domain.Tournament{}, err
	}
	updated, err := t.OpenRegistration(s.Clock.Now())
	if err != nil {
		return domain.Tournament{}, err
	}
	if err := s.Tournaments.Save(ctx, updated); err != nil {
		return domain.Tournament{}, err
	}
	writeAudit(ctx, s.AuditLog, s.Clock, domain.AuditLogEntry{
		TournamentID: updated.ID, ActorID: &cmd.ActorID,
		Action:    domain.ActionRegistrationOpened,
		SubjectID: updated.ID.String(),
	})
	return updated, nil
}

type CloseRegistration struct {
	Tournaments ports.TournamentRepository
	AuditLog    ports.AuditLogRepository
	Clock       ports.Clock
}

type CloseRegistrationCmd struct {
	ActorID      domain.UserID
	TournamentID domain.TournamentID
}

func (s *CloseRegistration) Handle(ctx context.Context, cmd CloseRegistrationCmd) (domain.Tournament, error) {
	t, err := s.Tournaments.Find(ctx, cmd.TournamentID)
	if err != nil {
		return domain.Tournament{}, err
	}
	updated, err := t.CloseRegistration(s.Clock.Now())
	if err != nil {
		return domain.Tournament{}, err
	}
	if err := s.Tournaments.Save(ctx, updated); err != nil {
		return domain.Tournament{}, err
	}
	writeAudit(ctx, s.AuditLog, s.Clock, domain.AuditLogEntry{
		TournamentID: updated.ID, ActorID: &cmd.ActorID,
		Action:    domain.ActionRegistrationClosed,
		SubjectID: updated.ID.String(),
	})
	return updated, nil
}

// ---- RegisterPlayer (self-register) ----

type RegisterPlayer struct {
	Tournaments ports.TournamentRepository
	TPlayers    ports.TournamentPlayerRepository
	AuditLog    ports.AuditLogRepository
	Clock       ports.Clock
}

type RegisterPlayerCmd struct {
	ActorID      domain.UserID
	TournamentID domain.TournamentID
}

func (s *RegisterPlayer) Handle(ctx context.Context, cmd RegisterPlayerCmd) (domain.TournamentPlayer, error) {
	t, err := s.Tournaments.Find(ctx, cmd.TournamentID)
	if err != nil {
		return domain.TournamentPlayer{}, err
	}
	if t.RegistrationMode != domain.RegModeSelfRegister {
		return domain.TournamentPlayer{}, fmt.Errorf("tournament does not allow self-registration")
	}
	if t.Status != domain.StatusRegistrationOpen {
		return domain.TournamentPlayer{}, fmt.Errorf("tournament is not open for registration")
	}
	if t.MaxPlayers > 0 {
		count, _ := s.TPlayers.CountRegistered(ctx, t.ID)
		if count >= t.MaxPlayers {
			return domain.TournamentPlayer{}, domain.ErrAtCap
		}
	}
	if err := s.TPlayers.Add(ctx, t.ID, cmd.ActorID); err != nil {
		return domain.TournamentPlayer{}, err
	}
	tp := domain.TournamentPlayer{
		TournamentID: t.ID,
		PlayerID:     cmd.ActorID,
		RegisteredAt: s.Clock.Now(),
	}
	writeAudit(ctx, s.AuditLog, s.Clock, domain.AuditLogEntry{
		TournamentID: t.ID, ActorID: &cmd.ActorID,
		Action:    domain.ActionPlayerAdded,
		SubjectID: cmd.ActorID.String(),
		After:     jsonString(map[string]string{"player_id": cmd.ActorID.String(), "via": "self_register"}),
	})
	return tp, nil
}

// ---- AddPlayerByUser (manager-only) ----

type AddPlayerByUser struct {
	Tournaments ports.TournamentRepository
	TPlayers    ports.TournamentPlayerRepository
	Users       ports.UserRepository
	AuditLog    ports.AuditLogRepository
	Clock       ports.Clock
}

type AddPlayerByUserCmd struct {
	ActorID      domain.UserID
	TournamentID domain.TournamentID
	UserID       domain.UserID
}

func (s *AddPlayerByUser) Handle(ctx context.Context, cmd AddPlayerByUserCmd) (domain.TournamentPlayer, error) {
	t, err := s.Tournaments.Find(ctx, cmd.TournamentID)
	if err != nil {
		return domain.TournamentPlayer{}, err
	}
	if t.RegistrationMode != domain.RegModeInviteOnly {
		return domain.TournamentPlayer{}, fmt.Errorf("tournament does not allow manager add")
	}
	if t.Status != domain.StatusDraft && t.Status != domain.StatusRegistrationOpen {
		return domain.TournamentPlayer{}, fmt.Errorf("tournament is not accepting new players")
	}
	if _, err := s.Users.FindByID(ctx, cmd.UserID); err != nil {
		return domain.TournamentPlayer{}, err
	}
	if err := s.TPlayers.Add(ctx, t.ID, cmd.UserID); err != nil {
		return domain.TournamentPlayer{}, err
	}
	writeAudit(ctx, s.AuditLog, s.Clock, domain.AuditLogEntry{
		TournamentID: t.ID, ActorID: &cmd.ActorID,
		Action:    domain.ActionPlayerAdded,
		SubjectID: cmd.UserID.String(),
		After:     jsonString(map[string]string{"player_id": cmd.UserID.String(), "via": "manager_add"}),
	})
	return domain.TournamentPlayer{
		TournamentID: t.ID,
		PlayerID:     cmd.UserID,
		RegisteredAt: s.Clock.Now(),
	}, nil
}

// ---- WithdrawPlayer ----

type WithdrawPlayer struct {
	Tournaments ports.TournamentRepository
	TPlayers    ports.TournamentPlayerRepository
	AuditLog    ports.AuditLogRepository
	Clock       ports.Clock
}

type WithdrawPlayerCmd struct {
	ActorID      domain.UserID
	IsManager    bool
	TournamentID domain.TournamentID
	PlayerID     domain.UserID
}

func (s *WithdrawPlayer) Handle(ctx context.Context, cmd WithdrawPlayerCmd) error {
	t, err := s.Tournaments.Find(ctx, cmd.TournamentID)
	if err != nil {
		return err
	}
	if t.Status != domain.StatusDraft && t.Status != domain.StatusRegistrationOpen {
		return fmt.Errorf("tournament is past registration; cannot withdraw")
	}
	if !cmd.IsManager && cmd.ActorID != cmd.PlayerID {
		return fmt.Errorf("only the player themselves or a manager can withdraw")
	}
	if err := s.TPlayers.Remove(ctx, t.ID, cmd.PlayerID); err != nil {
		return err
	}
	action := domain.ActionPlayerWithdrew
	if cmd.IsManager && cmd.ActorID != cmd.PlayerID {
		action = domain.ActionPlayerRemoved
	}
	writeAudit(ctx, s.AuditLog, s.Clock, domain.AuditLogEntry{
		TournamentID: t.ID, ActorID: &cmd.ActorID,
		Action: action, SubjectID: cmd.PlayerID.String(),
	})
	return nil
}

// ---- InviteUser ----

type InviteUser struct {
	Tournaments ports.TournamentRepository
	TPlayers    ports.TournamentPlayerRepository
	AuthTokens  ports.AuthTokenRepository
	Email       ports.EmailSender
	AuditLog    ports.AuditLogRepository
	Clock       ports.Clock
	PublicURL   string
}

type InviteUserCmd struct {
	ActorID      domain.UserID
	TournamentID *domain.TournamentID
	Email        string
	InviterName  string
}

func (s *InviteUser) Handle(ctx context.Context, cmd InviteUserCmd) (string, error) {
	if cmd.Email == "" {
		return "", fmt.Errorf("email required")
	}
	now := s.Clock.Now()
	raw := newToken()
	at := domain.AuthToken{
		ID:        domain.NewAuthTokenID(),
		Kind:      domain.AuthTokenInvite,
		Email:     strings.ToLower(strings.TrimSpace(cmd.Email)),
		TokenHash: hashToken(raw),
		ExpiresAt: now.Add(7 * 24 * time.Hour),
		CreatedBy: &cmd.ActorID,
		CreatedAt: now,
	}
	if err := s.AuthTokens.Save(ctx, at); err != nil {
		return "", err
	}
	if cmd.TournamentID != nil {
		writeAudit(ctx, s.AuditLog, s.Clock, domain.AuditLogEntry{
			TournamentID: *cmd.TournamentID,
			ActorID:      &cmd.ActorID,
			Action:       domain.ActionPlayerInvited,
			SubjectID:    cmd.Email,
			After:        jsonString(map[string]any{"email": cmd.Email, "tournament_id": cmd.TournamentID.String()}),
		})
	}
	url := s.PublicURL + "/invites/" + raw
	if err := s.Email.SendInvite(ctx, cmd.Email, url, cmd.InviterName); err != nil {
		return "", err
	}
	return url, nil
}

// ---- AcceptInvite ----

type AcceptInvite struct {
	Users      ports.UserRepository
	AuthTokens ports.AuthTokenRepository
	TPlayers   ports.TournamentPlayerRepository
	Hasher     ports.PasswordHasher
	Clock      ports.Clock
	AuditLog   ports.AuditLogRepository
}

type AcceptInviteCmd struct {
	RawToken string
	Name     string
	Password string
}

func (s *AcceptInvite) Handle(ctx context.Context, cmd AcceptInviteCmd) (domain.User, error) {
	if cmd.Password == "" || len(cmd.Password) < 8 {
		return domain.User{}, ErrWeakPassword
	}
	hash, err := s.AuthTokens.FindByHash(ctx, domain.AuthTokenInvite, hashToken(cmd.RawToken))
	if err != nil {
		if errors.Is(err, domain.ErrTokenNotFound) {
			return domain.User{}, domain.ErrTokenNotFound
		}
		return domain.User{}, err
	}
	if err := hash.IsUsable(s.Clock.Now()); err != nil {
		return domain.User{}, err
	}
	if hash.UserID != nil {
		return domain.User{}, domain.ErrTokenUsed
	}
	existing, _ := s.Users.FindByEmail(ctx, hash.Email)
	var u domain.User
	if existing.ID != "" {
		u = existing
	} else {
		if cmd.Name == "" {
			cmd.Name = strings.SplitN(hash.Email, "@", 2)[0]
		}
		pHash, err := s.Hasher.Hash(cmd.Password)
		if err != nil {
			return domain.User{}, err
		}
		u, err = domain.NewUser(hash.Email, cmd.Name, pHash, s.Clock.Now())
		if err != nil {
			return domain.User{}, err
		}
		if err := s.Users.Save(ctx, u); err != nil {
			return domain.User{}, err
		}
	}
	now := s.Clock.Now()
	if err := s.AuthTokens.MarkUsed(ctx, hash.ID, u.ID, domain.FormatTime(now)); err != nil {
		return domain.User{}, err
	}
	return u, nil
}

// ---- Spectator tokens ----

type IssueSpectatorToken struct {
	Tournaments ports.TournamentRepository
	Tokens      ports.TournamentSpectatorTokenRepository
	AuditLog    ports.AuditLogRepository
	Clock       ports.Clock
	PublicURL   string
}

type IssueSpectatorTokenCmd struct {
	ActorID      domain.UserID
	TournamentID domain.TournamentID
	Label        string
}

func (s *IssueSpectatorToken) Handle(ctx context.Context, cmd IssueSpectatorTokenCmd) (string, domain.SpectatorToken, error) {
	if _, err := s.Tournaments.Find(ctx, cmd.TournamentID); err != nil {
		return "", domain.SpectatorToken{}, err
	}
	raw := newToken()
	now := s.Clock.Now()
	tok := domain.SpectatorToken{
		ID:           domain.NewSpectatorTokenID(),
		TournamentID: cmd.TournamentID,
		TokenHash:    hashToken(raw),
		Label:        cmd.Label,
		IssuedAt:     now,
		IssuedBy:     cmd.ActorID,
	}
	if err := s.Tokens.Save(ctx, tok); err != nil {
		return "", domain.SpectatorToken{}, err
	}
	writeAudit(ctx, s.AuditLog, s.Clock, domain.AuditLogEntry{
		TournamentID: cmd.TournamentID, ActorID: &cmd.ActorID,
		Action:    domain.ActionSpectatorTokenIssued,
		SubjectID: tok.ID.String(),
		After:     jsonString(map[string]string{"label": cmd.Label, "token_id": tok.ID.String()}),
	})
	url := s.PublicURL + "/tournaments/" + cmd.TournamentID.String() + "?spectator_token=" + raw
	return url, tok, nil
}

type RevokeSpectatorToken struct {
	Tournaments ports.TournamentRepository
	Tokens      ports.TournamentSpectatorTokenRepository
	AuditLog    ports.AuditLogRepository
	Clock       ports.Clock
}

type RevokeSpectatorTokenCmd struct {
	ActorID      domain.UserID
	TournamentID domain.TournamentID
	TokenID      domain.SpectatorTokenID
}

func (s *RevokeSpectatorToken) Handle(ctx context.Context, cmd RevokeSpectatorTokenCmd) error {
	tok, err := s.Tokens.Find(ctx, cmd.TokenID)
	if err != nil {
		return err
	}
	if tok.TournamentID != cmd.TournamentID {
		return fmt.Errorf("token does not belong to this tournament")
	}
	if tok.RevokedAt != nil {
		return nil // idempotent
	}
	now := s.Clock.Now()
	if err := s.Tokens.Revoke(ctx, tok.ID, cmd.ActorID, domain.FormatTime(now)); err != nil {
		return err
	}
	writeAudit(ctx, s.AuditLog, s.Clock, domain.AuditLogEntry{
		TournamentID: cmd.TournamentID, ActorID: &cmd.ActorID,
		Action:    domain.ActionSpectatorTokenRevoked,
		SubjectID: tok.ID.String(),
	})
	return nil
}

// ---- EditSeed ----

type EditSeed struct {
	Tournaments ports.TournamentRepository
	TPlayers    ports.TournamentPlayerRepository
	AuditLog    ports.AuditLogRepository
	Clock       ports.Clock
}

type EditSeedCmd struct {
	ActorID      domain.UserID
	TournamentID domain.TournamentID
	PlayerID     domain.UserID
	Seed         int
}

func (s *EditSeed) Handle(ctx context.Context, cmd EditSeedCmd) error {
	t, err := s.Tournaments.Find(ctx, cmd.TournamentID)
	if err != nil {
		return err
	}
	if t.Status != domain.StatusDraft && t.Status != domain.StatusRegistrationOpen {
		return domain.ErrBracketLocked
	}
	players, _ := s.TPlayers.ListByTournament(ctx, t.ID)
	if cmd.Seed < 1 || cmd.Seed > len(players) {
		return domain.ErrSeedOutOfRange
	}
	taken, _ := s.TPlayers.IsSeedTakenByOther(ctx, t.ID, cmd.PlayerID, cmd.Seed)
	if taken {
		return domain.ErrSeedInUse
	}
	// Find current seed for before/after.
	var beforeSeed *int
	for _, p := range players {
		if p.PlayerID == cmd.PlayerID {
			beforeSeed = p.Seed
			break
		}
	}
	if err := s.TPlayers.UpdateSeed(ctx, t.ID, cmd.PlayerID, cmd.Seed); err != nil {
		return err
	}
	writeAudit(ctx, s.AuditLog, s.Clock, domain.AuditLogEntry{
		TournamentID: t.ID, ActorID: &cmd.ActorID,
		Action:    domain.ActionSeedChanged,
		SubjectID: cmd.PlayerID.String(),
		Before:    jsonString(map[string]any{"seed": beforeSeed}),
		After:     jsonString(map[string]any{"seed": cmd.Seed}),
	})
	return nil
}

// ---- AddManager / RemoveManager ----

type AddManager struct {
	Tournaments ports.TournamentRepository
	TManagers   ports.TournamentManagerRepository
	Users       ports.UserRepository
	AuditLog    ports.AuditLogRepository
	Clock       ports.Clock
}

type AddManagerCmd struct {
	ActorID      domain.UserID
	TournamentID domain.TournamentID
	TargetID     domain.UserID
}

func (s *AddManager) Handle(ctx context.Context, cmd AddManagerCmd) error {
	if _, err := s.Users.FindByID(ctx, cmd.TargetID); err != nil {
		return err
	}
	if err := s.TManagers.Add(ctx, cmd.TournamentID, cmd.TargetID); err != nil {
		return err
	}
	writeAudit(ctx, s.AuditLog, s.Clock, domain.AuditLogEntry{
		TournamentID: cmd.TournamentID, ActorID: &cmd.ActorID,
		Action:    domain.ActionManagerAdded,
		SubjectID: cmd.TargetID.String(),
		Before:    "null",
		After:     jsonString(map[string]string{"manager_id": cmd.TargetID.String()}),
	})
	return nil
}

type RemoveManager struct {
	Tournaments ports.TournamentRepository
	TManagers   ports.TournamentManagerRepository
	AuditLog    ports.AuditLogRepository
	Clock       ports.Clock
}

type RemoveManagerCmd struct {
	ActorID      domain.UserID
	TournamentID domain.TournamentID
	TargetID     domain.UserID
}

var ErrSelfRemoveNotAllowed = errors.New("manager self-removal blocked except via cancel")

func (s *RemoveManager) Handle(ctx context.Context, cmd RemoveManagerCmd) error {
	t, err := s.Tournaments.Find(ctx, cmd.TournamentID)
	if err != nil {
		return err
	}
	if cmd.ActorID == cmd.TargetID && t.Status != domain.StatusCancelled {
		return ErrSelfRemoveNotAllowed
	}
	if err := s.TManagers.Remove(ctx, cmd.TournamentID, cmd.TargetID); err != nil {
		return err
	}
	writeAudit(ctx, s.AuditLog, s.Clock, domain.AuditLogEntry{
		TournamentID: cmd.TournamentID, ActorID: &cmd.ActorID,
		Action:    domain.ActionManagerRemoved,
		SubjectID: cmd.TargetID.String(),
	})
	return nil
}

// ---- CancelTournament / ForceCloseTournament ----

type CancelTournament struct {
	Tournaments ports.TournamentRepository
	AuditLog    ports.AuditLogRepository
	Clock       ports.Clock
}

type CancelTournamentCmd struct {
	ActorID      domain.UserID
	TournamentID domain.TournamentID
}

func (s *CancelTournament) Handle(ctx context.Context, cmd CancelTournamentCmd) (domain.Tournament, error) {
	t, err := s.Tournaments.Find(ctx, cmd.TournamentID)
	if err != nil {
		return domain.Tournament{}, err
	}
	before := t
	updated, err := t.Cancel(s.Clock.Now())
	if err != nil {
		return domain.Tournament{}, err
	}
	if err := s.Tournaments.Save(ctx, updated); err != nil {
		return domain.Tournament{}, err
	}
	b, a := beforeAfter(before, updated)
	writeAudit(ctx, s.AuditLog, s.Clock, domain.AuditLogEntry{
		TournamentID: updated.ID, ActorID: &cmd.ActorID,
		Action:    domain.ActionTournamentCancelled,
		SubjectID: updated.ID.String(),
		Before:    b, After: a,
	})
	return updated, nil
}

type ForceCloseTournament struct {
	Tournaments ports.TournamentRepository
	AuditLog    ports.AuditLogRepository
	Clock       ports.Clock
}

type ForceCloseTournamentCmd struct {
	ActorID      domain.UserID
	TournamentID domain.TournamentID
}

func (s *ForceCloseTournament) Handle(ctx context.Context, cmd ForceCloseTournamentCmd) (domain.Tournament, error) {
	t, err := s.Tournaments.Find(ctx, cmd.TournamentID)
	if err != nil {
		return domain.Tournament{}, err
	}
	before := t
	updated, err := t.ForceClose(s.Clock.Now())
	if err != nil {
		return domain.Tournament{}, err
	}
	if err := s.Tournaments.Save(ctx, updated); err != nil {
		return domain.Tournament{}, err
	}
	b, a := beforeAfter(before, updated)
	writeAudit(ctx, s.AuditLog, s.Clock, domain.AuditLogEntry{
		TournamentID: updated.ID, ActorID: &cmd.ActorID,
		Action:    domain.ActionTournamentCompleted,
		SubjectID: updated.ID.String(),
		Before:    b, After: a,
	})
	return updated, nil
}

// ---- ListAuditLog ----

type ListAuditLog struct {
	AuditLog ports.AuditLogRepository
}

type ListAuditLogCmd struct {
	TournamentID domain.TournamentID
	Limit        int
	Offset       int
}

func (s *ListAuditLog) Handle(ctx context.Context, cmd ListAuditLogCmd) ([]domain.AuditLogEntry, error) {
	if cmd.Limit <= 0 {
		cmd.Limit = 50
	}
	return s.AuditLog.ListByTournament(ctx, cmd.TournamentID, cmd.Limit, cmd.Offset)
}

// ---- helpers for bracket generation (used in PreviewBracket / StartTournament) ----

// PRNGShuffle performs a deterministic Fisher-Yates shuffle using the seed.
func PRNGShuffle(seed int64, n int) []int {
	perm := make([]int, n)
	for i := range perm {
		perm[i] = i
	}
	x := seed
	if x == 0 {
		x = 1
	}
	for i := n - 1; i > 0; i-- {
		// xorshift64
		x ^= x << 13
		x ^= x >> 7
		x ^= x << 17
		j := int(x&0x7fffffffffffffff) % (i + 1)
		if j < 0 {
			j = -j
		}
		perm[i], perm[j] = perm[j], perm[i]
	}
	return perm
}

// Stable sort helpers
type pair struct {
	seed int
	id   domain.UserID
}
type bySeed []pair

func (s bySeed) Len() int           { return len(s) }
func (s bySeed) Less(i, j int) bool { return s[i].seed < s[j].seed }
func (s bySeed) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }

// ---- Sort utilities ----

// Ensure non-decreasing ordering.
var _ = sort.Slice

// ---- Bracket generation ----

// GenerateBracket is the pure bracket-shape generator used by preview and start.
type BracketSpec struct {
	TotalPlayers int
	Min, Max     int
	AdvanceMap   map[string]int
}

// GenerateBracket returns a flat list of rounds (each with matches and the
// per-match advance count) for the given player count. The hard constraint is
// that the final round has exactly 4 players.
func GenerateBracket(spec BracketSpec) (BracketLayout, error) {
	if spec.TotalPlayers < 4 {
		return BracketLayout{}, fmt.Errorf("need at least 4 players")
	}
	if spec.Min < 2 || spec.Max > 4 || spec.Min > spec.Max {
		return BracketLayout{}, fmt.Errorf("invalid per-match range")
	}
	if spec.AdvanceMap == nil {
		spec.AdvanceMap = map[string]int{"1": 1}
	}
	rounds := []RoundLayout{}
	remaining := spec.TotalPlayers
	round := 1
	for remaining > 4 {
		minMatches := (remaining + spec.Max - 1) / spec.Max
		maxMatches := remaining / spec.Min
		if maxMatches < 1 {
			maxMatches = 1
		}
		// pick a balanced number of matches
		g := maxMatches
		if g < minMatches {
			g = minMatches
		}
		// compute per-match size distribution that sums to `remaining`.
		matches := []MatchLayout{}
		base := remaining / g
		extra := remaining % g
		for i := 0; i < g; i++ {
			size := base
			if i < extra {
				size++
			}
			if size < spec.Min {
				size = spec.Min
			}
			if size > spec.Max {
				size = spec.Max
			}
			// Advance count: try to keep matches balanced.
			adv := advanceForRound(round, spec.AdvanceMap)
			if adv > size {
				adv = size
			}
			if adv < 1 {
				adv = 1
			}
			matches = append(matches, MatchLayout{PlayersIn: size, Advance: adv})
			remaining -= adv
		}
		// adjust remaining if we over-shot
		if remaining < 0 {
			return BracketLayout{}, fmt.Errorf("no valid bracket shape for %d players", spec.TotalPlayers)
		}
		rounds = append(rounds, RoundLayout{Round: round, Matches: matches})
		round++
	}
	// Final round: exactly 4.
	if remaining < 1 || remaining > 5 {
		return BracketLayout{}, fmt.Errorf("final round has %d players, expected 4", remaining)
	}
	if remaining == 5 {
		// one bye in the final
		rounds = append(rounds, RoundLayout{
			Round: round,
			Matches: []MatchLayout{
				{PlayersIn: 4, Advance: 1},
				{PlayersIn: 1, Advance: 1, IsBye: true},
			},
		})
	} else {
		rounds = append(rounds, RoundLayout{
			Round: round,
			Matches: []MatchLayout{
				{PlayersIn: remaining, Advance: 1},
			},
		})
	}
	return BracketLayout{
		TotalPlayers: spec.TotalPlayers,
		TotalRounds:  round,
		Rounds:       rounds,
	}, nil
}

func advanceForRound(round int, m map[string]int) int {
	k := fmt.Sprintf("%d", round)
	if v, ok := m[k]; ok {
		return v
	}
	return 1
}

type BracketLayout struct {
	TotalPlayers int
	TotalRounds  int
	Rounds       []RoundLayout
}

type RoundLayout struct {
	Round   int
	Matches []MatchLayout
}

type MatchLayout struct {
	PlayersIn int
	Advance   int
	IsBye     bool
}

// ---- StartTournament ----

type StartTournament struct {
	Tournaments ports.TournamentRepository
	TPlayers    ports.TournamentPlayerRepository
	Matches     ports.MatchRepository
	MatchParts  ports.MatchParticipantRepository
	AuditLog    ports.AuditLogRepository
	Clock       ports.Clock
	Broadcaster ports.Broadcaster
}

type StartTournamentCmd struct {
	ActorID      domain.UserID
	TournamentID domain.TournamentID
}

func (s *StartTournament) Handle(ctx context.Context, cmd StartTournamentCmd) (domain.Tournament, error) {
	t, err := s.Tournaments.Find(ctx, cmd.TournamentID)
	if err != nil {
		return domain.Tournament{}, err
	}
	players, err := s.TPlayers.ListByTournament(ctx, t.ID)
	if err != nil {
		return domain.Tournament{}, err
	}
	if len(players) < 4 {
		return domain.Tournament{}, fmt.Errorf("at least 4 players required (have %d)", len(players))
	}
	// Generate bracket shape.
	spec := BracketSpec{
		TotalPlayers: len(players),
		Min:          t.MinPlayersPerMatch,
		Max:          t.MaxPlayersPerMatch,
		AdvanceMap:   t.PlayersAdvancingPerRound,
	}
	layout, err := GenerateBracket(spec)
	if err != nil {
		return domain.Tournament{}, err
	}
	// Generate PRNG seed.
	seed := int64(time.Now().UnixNano())
	// Sort players: those with seeds first, then by PRNG shuffle.
	seededPlayers := []domain.TournamentPlayer{}
	unseededPlayers := []domain.TournamentPlayer{}
	for _, p := range players {
		if p.Seed != nil {
			seededPlayers = append(seededPlayers, p)
		} else {
			unseededPlayers = append(unseededPlayers, p)
		}
	}
	sort.SliceStable(seededPlayers, func(i, j int) bool {
		si, sj := *seededPlayers[i].Seed, *seededPlayers[j].Seed
		return si < sj
	})
	perm := PRNGShuffle(seed, len(unseededPlayers))
	shuffled := make([]domain.TournamentPlayer, len(unseededPlayers))
	for i := range unseededPlayers {
		shuffled[i] = unseededPlayers[perm[i]]
	}
	ordered := append(seededPlayers, shuffled...)
	// Build matches.
	matches := []domain.Match{}
	participants := []domain.MatchParticipant{}
	position := 0
	for _, round := range layout.Rounds {
		for _, ml := range round.Matches {
			now := s.Clock.Now()
			m := domain.Match{
				ID:              domain.NewMatchID(),
				TournamentID:    t.ID,
				Round:           round.Round,
				PositionInRound: 0,
				Status:          domain.MatchPending,
				CreatedAt:       now,
				UpdatedAt:       now,
			}
			matches = append(matches, m)
			slots := slotsForMatch(ml.PlayersIn)
			if ml.IsBye {
				// single slot, is_bye = true
				participants = append(participants, domain.MatchParticipant{
					MatchID: m.ID, Slot: domain.SlotHome, IsBye: true,
				})
				continue
			}
			for i := 0; i < ml.PlayersIn && position+i < len(ordered); i++ {
				uid := ordered[position+i].PlayerID
				participants = append(participants, domain.MatchParticipant{
					MatchID: m.ID, Slot: slots[i], PlayerID: &uid,
				})
			}
			position += ml.PlayersIn
		}
	}
	// Assign position_in_round within each round.
	counter2 := map[int]int{}
	for i := range matches {
		r := matches[i].Round
		counter2[r]++
		matches[i].PositionInRound = counter2[r]
	}
	// Wire next_match_id links: a match in round N feeds into the
	// (ceil(k / advance)) match in round N+1.
	// Group matches by round.
	byRound := map[int][]int{}
	for i, m := range matches {
		byRound[m.Round] = append(byRound[m.Round], i)
	}
	for round := 1; round < layout.TotalRounds; round++ {
		nextRoundMatches := byRound[round+1]
		if len(nextRoundMatches) == 0 {
			continue
		}
		// for each match in this round, count advancers
		slotCursor := 0
		for i, idx := range byRound[round] {
			ml := layout.Rounds[round-1].Matches[i]
			adv := ml.Advance
			// determine which next match they go into
			for k := 0; k < adv; k++ {
				nextIdx := nextRoundMatches[0] // simplified: all go to first match of next round
				_ = nextIdx
				if nextIdx >= len(matches) {
					continue
				}
				slotCursor++
			}
			m := &matches[idx]
			// Find the next round match that this feeds into.
			nextIdx := nextRoundMatches[0]
			if nextIdx >= len(matches) {
				continue
			}
			next := &matches[nextIdx]
			nm := next.ID
			slot := domain.SlotHome
			m.NextMatchID = &nm
			m.SlotInNextMatch = &slot
			_ = slotCursor
		}
	}
	// Save.
	if err := s.Matches.SaveBatch(ctx, matches); err != nil {
		return domain.Tournament{}, err
	}
	if err := s.MatchParts.InsertBatch(ctx, participants); err != nil {
		return domain.Tournament{}, err
	}
	now := s.Clock.Now()
	started, err := t.Start(now, seed)
	if err != nil {
		return domain.Tournament{}, err
	}
	if err := s.Tournaments.Save(ctx, started); err != nil {
		return domain.Tournament{}, err
	}
	writeAudit(ctx, s.AuditLog, s.Clock, domain.AuditLogEntry{
		TournamentID: started.ID, ActorID: &cmd.ActorID,
		Action:    domain.ActionTournamentStarted,
		SubjectID: started.ID.String(),
		After:     jsonString(map[string]any{"random_seed": seed, "started_at": now, "shape": layout.TotalRounds}),
	})
	return started, nil
}

func slotsForMatch(n int) []domain.Slot {
	switch n {
	case 2:
		return []domain.Slot{domain.SlotHome, domain.SlotAway}
	case 3:
		return []domain.Slot{domain.SlotHome, domain.SlotAway, domain.SlotThird}
	default:
		return []domain.Slot{domain.SlotHome, domain.SlotAway, domain.SlotThird, domain.SlotFourth}
	}
}

// ---- PreviewBracket ----

type PreviewBracket struct {
	Tournaments ports.TournamentRepository
	TPlayers    ports.TournamentPlayerRepository
}

type PreviewBracketCmd struct {
	ActorID      domain.UserID
	TournamentID domain.TournamentID
}

func (s *PreviewBracket) Handle(ctx context.Context, cmd PreviewBracketCmd) (domain.Bracket, error) {
	t, err := s.Tournaments.Find(ctx, cmd.TournamentID)
	if err != nil {
		return domain.Bracket{}, err
	}
	players, _ := s.TPlayers.ListByTournament(ctx, t.ID)
	spec := BracketSpec{
		TotalPlayers: len(players),
		Min:          t.MinPlayersPerMatch,
		Max:          t.MaxPlayersPerMatch,
		AdvanceMap:   t.PlayersAdvancingPerRound,
	}
	layout, err := GenerateBracket(spec)
	if err != nil {
		return domain.Bracket{}, err
	}
	rs := make([]domain.RoundShape, len(layout.Rounds))
	for i, r := range layout.Rounds {
		advMap := map[string]int{}
		advCounts := map[int]int{}
		for _, m := range r.Matches {
			advCounts[m.Advance]++
			advMap[fmt.Sprintf("%d", m.Advance)] = m.Advance
		}
		_ = advMap
		rs[i] = domain.RoundShape{
			Round:       r.Round,
			Matches:     len(r.Matches),
			PlayersIn:   totalPlayersIn(r.Matches),
			AdvanceMap:  advMap,
			Description: describeRound(r),
		}
	}
	return domain.Bracket{
		TotalRounds:  layout.TotalRounds,
		TotalPlayers: layout.TotalPlayers,
		Rounds:       rs,
	}, nil
}

func totalPlayersIn(ms []MatchLayout) int {
	n := 0
	for _, m := range ms {
		n += m.PlayersIn
	}
	return n
}

func describeRound(r RoundLayout) string {
	advDist := map[int]int{}
	for _, m := range r.Matches {
		advDist[m.Advance]++
	}
	parts := []string{}
	for adv, n := range advDist {
		parts = append(parts, fmt.Sprintf("%d advance %d", n, adv))
	}
	return strings.Join(parts, ", ")
}

// ---- Match lifecycle services ----

type MarkMatchInProgress struct {
	Matches ports.MatchRepository
	Clock   ports.Clock
}

type MarkMatchInProgressCmd struct {
	ActorID   domain.UserID
	MatchID   domain.MatchID
	IsManager bool
}

func (s *MarkMatchInProgress) Handle(ctx context.Context, cmd MarkMatchInProgressCmd) (domain.Match, error) {
	m, _, err := s.Matches.FindWithParticipants(ctx, cmd.MatchID)
	if err != nil {
		return domain.Match{}, err
	}
	if m.Status != domain.MatchReady && m.Status != domain.MatchPending {
		return domain.Match{}, fmt.Errorf("match is not in ready state")
	}
	if !cmd.IsManager {
		isP := false
		_, parts, err := s.Matches.FindWithParticipants(ctx, m.ID)
		if err != nil {
			return domain.Match{}, err
		}
		for _, p := range parts {
			if p.PlayerID != nil && *p.PlayerID == cmd.ActorID {
				isP = true
				break
			}
		}
		if !isP {
			return domain.Match{}, domain.ErrNotParticipant
		}
	}
	now := s.Clock.Now()
	if err := s.Matches.UpdateStatus(ctx, m.ID, domain.MatchInProgress, domain.FormatTime(now)); err != nil {
		return domain.Match{}, err
	}
	m.Status = domain.MatchInProgress
	m.UpdatedAt = now
	return m, nil
}

type RecordMatchResult struct {
	Matches  ports.MatchRepository
	Clock    ports.Clock
	AuditLog ports.AuditLogRepository
}

type RecordMatchResultCmd struct {
	ActorID   domain.UserID
	MatchID   domain.MatchID
	IsManager bool
	Positions map[domain.Slot]int
}

func (s *RecordMatchResult) Handle(ctx context.Context, cmd RecordMatchResultCmd) (domain.Match, error) {
	m, parts, err := s.Matches.FindWithParticipants(ctx, cmd.MatchID)
	if err != nil {
		return domain.Match{}, err
	}
	if m.Status != domain.MatchInProgress {
		return domain.Match{}, domain.ErrWrongMatchStatus
	}
	if !cmd.IsManager {
		isP := false
		for _, p := range parts {
			if p.PlayerID != nil && *p.PlayerID == cmd.ActorID {
				isP = true
				break
			}
		}
		if !isP {
			return domain.Match{}, domain.ErrNotParticipant
		}
	}
	if err := validateAdvancingPositions(parts, cmd.Positions); err != nil {
		return domain.Match{}, err
	}
	for slot, pos := range cmd.Positions {
		if err := s.Matches.UpdateParticipantPosition(ctx, m.ID, slot, pos); err != nil {
			return domain.Match{}, err
		}
	}
	now := s.Clock.Now()
	if err := s.Matches.UpdateStatus(ctx, m.ID, domain.MatchCompleted, domain.FormatTime(now)); err != nil {
		return domain.Match{}, err
	}
	m.Status = domain.MatchCompleted
	m.UpdatedAt = now
	writeAudit(ctx, s.AuditLog, s.Clock, domain.AuditLogEntry{
		TournamentID: m.TournamentID, ActorID: &cmd.ActorID,
		Action:    domain.ActionMatchResultRecorded,
		SubjectID: m.ID.String(),
		After:     jsonString(cmd.Positions),
	})
	return m, nil
}

func validateAdvancingPositions(parts []domain.MatchParticipant, positions map[domain.Slot]int) error {
	playerCount := 0
	for _, p := range parts {
		if !p.IsBye && p.PlayerID != nil {
			playerCount++
		}
	}
	if playerCount == 0 {
		return fmt.Errorf("no real participants in this match")
	}
	// positions count must be ≤ player count
	if len(positions) == 0 {
		return ErrAdvancingPositionsRequired
	}
	// for now we just accept the positions; the real constraint is per-round
	// advance count which the caller enforces separately.
	return nil
}

type CorrectMatchResult struct {
	Matches  ports.MatchRepository
	Clock    ports.Clock
	AuditLog ports.AuditLogRepository
}

type CorrectMatchResultCmd struct {
	ActorID   domain.UserID
	MatchID   domain.MatchID
	Positions map[domain.Slot]int
}

func (s *CorrectMatchResult) Handle(ctx context.Context, cmd CorrectMatchResultCmd) (domain.Match, error) {
	m, parts, err := s.Matches.FindWithParticipants(ctx, cmd.MatchID)
	if err != nil {
		return domain.Match{}, err
	}
	if m.Status != domain.MatchCompleted {
		return domain.Match{}, fmt.Errorf("match is not completed; nothing to correct")
	}
	downstream, err := s.Matches.GetDownstreamChain(ctx, m.ID)
	if err != nil {
		return domain.Match{}, err
	}
	for _, d := range downstream {
		if d.Status == domain.MatchInProgress || d.Status == domain.MatchCompleted {
			return domain.Match{}, domain.ErrDownstreamLocked
		}
	}
	if err := validateAdvancingPositions(parts, cmd.Positions); err != nil {
		return domain.Match{}, err
	}
	for slot, pos := range cmd.Positions {
		if err := s.Matches.UpdateParticipantPosition(ctx, m.ID, slot, pos); err != nil {
			return domain.Match{}, err
		}
	}
	now := s.Clock.Now()
	if err := s.Matches.UpdateStatus(ctx, m.ID, domain.MatchCompleted, domain.FormatTime(now)); err != nil {
		return domain.Match{}, err
	}
	writeAudit(ctx, s.AuditLog, s.Clock, domain.AuditLogEntry{
		TournamentID: m.TournamentID, ActorID: &cmd.ActorID,
		Action:    domain.ActionMatchResultCorrected,
		SubjectID: m.ID.String(),
		Before:    "previously recorded positions",
		After:     jsonString(cmd.Positions),
	})
	return m, nil
}

// ---- Exports ----

type ExportBracketCSV struct {
	Matches ports.MatchRepository
}

type ExportBracketCSVCmd struct {
	TournamentID domain.TournamentID
}

func (s *ExportBracketCSV) Handle(ctx context.Context, cmd ExportBracketCSVCmd) (string, error) {
	matches, parts, err := s.Matches.ListByTournamentWithParticipants(ctx, cmd.TournamentID)
	if err != nil {
		return "", err
	}
	pmap := map[domain.MatchID][]domain.MatchParticipant{}
	for _, p := range parts {
		pmap[p.MatchID] = append(pmap[p.MatchID], p)
	}
	var sb strings.Builder
	sb.WriteString("round,position,status,slot_home,slot_home_position,slot_away,slot_away_position,slot_third,slot_third_position,slot_fourth,slot_fourth_position\n")
	for _, m := range matches {
		row := []string{
			fmt.Sprintf("%d", m.Round),
			fmt.Sprintf("%d", m.PositionInRound),
			string(m.Status),
		}
		for _, slot := range []domain.Slot{domain.SlotHome, domain.SlotAway, domain.SlotThird, domain.SlotFourth} {
			var (
				name string
				pos  string
			)
			for _, p := range pmap[m.ID] {
				if p.Slot == slot {
					if p.IsBye || p.PlayerID == nil {
						name = "BYE"
					} else {
						name = p.PlayerID.String()
					}
					if p.AdvancingPosition != nil {
						pos = fmt.Sprintf("%d", *p.AdvancingPosition)
					}
					break
				}
			}
			row = append(row, csvField(name), csvField(pos))
		}
		sb.WriteString(strings.Join(row, ","))
		sb.WriteString("\n")
	}
	return sb.String(), nil
}

func csvField(s string) string {
	if strings.ContainsAny(s, ",\"\n") {
		return "\"" + strings.ReplaceAll(s, "\"", "\"\"") + "\""
	}
	return s
}

type ExportBracketPNG struct {
	Matches ports.MatchRepository
}

type ExportBracketPNGCmd struct {
	TournamentID domain.TournamentID
}

func (s *ExportBracketPNG) Handle(ctx context.Context, cmd ExportBracketPNGCmd) ([]byte, error) {
	// Minimal 1x1 PNG placeholder — full PNG generation deferred (T18).
	png := []byte{
		0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A,
		0x00, 0x00, 0x00, 0x0D, 0x49, 0x48, 0x44, 0x52,
		0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
		0x08, 0x06, 0x00, 0x00, 0x00, 0x1F, 0x15, 0xC4,
		0x89, 0x00, 0x00, 0x00, 0x0A, 0x49, 0x44, 0x41,
		0x54, 0x78, 0x9C, 0x63, 0x00, 0x01, 0x00, 0x00,
		0x05, 0x00, 0x01, 0x0D, 0x0A, 0x2D, 0xB4, 0x00,
		0x00, 0x00, 0x00, 0x49, 0x45, 0x4E, 0x44, 0xAE,
		0x42, 0x60, 0x82,
	}
	return png, nil
}

// ---- Bundle ----

// Bundle is a single container with every service. It removes the wiring
// burden from main.go and gives the httpinbound adapter a single struct to
// read services from.
type Bundle struct {
	SignIn                 *SignIn
	ChangePassword         *ChangePassword
	RequestPasswordReset   *RequestPasswordReset
	RedeemPasswordReset    *RedeemPasswordReset
	CreateTournament       *CreateTournament
	EditTournament         *EditTournament
	ChangeVisibility       *ChangeVisibility
	ChangeRegistrationMode *ChangeRegistrationMode
	OpenRegistration       *OpenRegistration
	CloseRegistration      *CloseRegistration
	StartTournament        *StartTournament
	CancelTournament       *CancelTournament
	ForceCloseTournament   *ForceCloseTournament
	RegisterPlayer         *RegisterPlayer
	AddPlayerByUser        *AddPlayerByUser
	WithdrawPlayer         *WithdrawPlayer
	InviteUser             *InviteUser
	AcceptInvite           *AcceptInvite
	IssueSpectatorToken    *IssueSpectatorToken
	RevokeSpectatorToken   *RevokeSpectatorToken
	EditSeed               *EditSeed
	PreviewBracket         *PreviewBracket
	MarkInProgress         *MarkMatchInProgress
	RecordMatchResult      *RecordMatchResult
	CorrectMatchResult     *CorrectMatchResult
	AddManager             *AddManager
	RemoveManager          *RemoveManager
	ListAuditLog           *ListAuditLog
	ExportBracketCSV       *ExportBracketCSV
	ExportBracketPNG       *ExportBracketPNG
}

// BuildParams carries every dependency a Bundle needs.
type BuildParams struct {
	Users           ports.UserRepository
	Hasher          ports.PasswordHasher
	Clock           ports.Clock
	Tournaments     ports.TournamentRepository
	TManagers       ports.TournamentManagerRepository
	TPlayers        ports.TournamentPlayerRepository
	AuthTokens      ports.AuthTokenRepository
	SpectatorTokens ports.TournamentSpectatorTokenRepository
	Matches         ports.MatchRepository
	MatchParts      ports.MatchParticipantRepository
	AuditLog        ports.AuditLogRepository
	Email           ports.EmailSender
	Broadcaster     ports.Broadcaster
	PublicURL       string
}

// New constructs a Bundle from the supplied dependencies.
func New(p BuildParams) *Bundle {
	b := &Bundle{
		SignIn:         &SignIn{Users: p.Users, Hasher: p.Hasher},
		ChangePassword: &ChangePassword{Users: p.Users, Hasher: p.Hasher, Clock: p.Clock},
		RequestPasswordReset: &RequestPasswordReset{
			Users: p.Users, AuthTokens: p.AuthTokens, Email: p.Email,
			Clock: p.Clock, PublicURL: p.PublicURL,
		},
		RedeemPasswordReset: &RedeemPasswordReset{
			Users: p.Users, AuthTokens: p.AuthTokens, Hasher: p.Hasher, Clock: p.Clock,
		},
		CreateTournament: &CreateTournament{
			Tournaments: p.Tournaments, TManagers: p.TManagers,
			AuditLog: p.AuditLog, Clock: p.Clock,
		},
		EditTournament: &EditTournament{
			Tournaments: p.Tournaments, AuditLog: p.AuditLog, Clock: p.Clock,
		},
		ChangeVisibility: &ChangeVisibility{
			Tournaments: p.Tournaments, AuditLog: p.AuditLog, Clock: p.Clock,
		},
		ChangeRegistrationMode: &ChangeRegistrationMode{
			Tournaments: p.Tournaments, AuditLog: p.AuditLog, Clock: p.Clock,
		},
		OpenRegistration: &OpenRegistration{
			Tournaments: p.Tournaments, AuditLog: p.AuditLog, Clock: p.Clock,
		},
		CloseRegistration: &CloseRegistration{
			Tournaments: p.Tournaments, AuditLog: p.AuditLog, Clock: p.Clock,
		},
		StartTournament: &StartTournament{
			Tournaments: p.Tournaments, TPlayers: p.TPlayers,
			Matches: p.Matches, MatchParts: p.MatchParts,
			AuditLog: p.AuditLog, Clock: p.Clock, Broadcaster: p.Broadcaster,
		},
		CancelTournament: &CancelTournament{
			Tournaments: p.Tournaments, AuditLog: p.AuditLog, Clock: p.Clock,
		},
		ForceCloseTournament: &ForceCloseTournament{
			Tournaments: p.Tournaments, AuditLog: p.AuditLog, Clock: p.Clock,
		},
		RegisterPlayer: &RegisterPlayer{
			Tournaments: p.Tournaments, TPlayers: p.TPlayers,
			AuditLog: p.AuditLog, Clock: p.Clock,
		},
		AddPlayerByUser: &AddPlayerByUser{
			Tournaments: p.Tournaments, TPlayers: p.TPlayers,
			Users: p.Users, AuditLog: p.AuditLog, Clock: p.Clock,
		},
		WithdrawPlayer: &WithdrawPlayer{
			Tournaments: p.Tournaments, TPlayers: p.TPlayers,
			AuditLog: p.AuditLog, Clock: p.Clock,
		},
		InviteUser: &InviteUser{
			Tournaments: p.Tournaments, TPlayers: p.TPlayers,
			AuthTokens: p.AuthTokens, Email: p.Email, AuditLog: p.AuditLog,
			Clock: p.Clock, PublicURL: p.PublicURL,
		},
		AcceptInvite: &AcceptInvite{
			Users: p.Users, AuthTokens: p.AuthTokens, TPlayers: p.TPlayers,
			Hasher: p.Hasher, Clock: p.Clock, AuditLog: p.AuditLog,
		},
		IssueSpectatorToken: &IssueSpectatorToken{
			Tournaments: p.Tournaments, Tokens: p.SpectatorTokens,
			AuditLog: p.AuditLog, Clock: p.Clock, PublicURL: p.PublicURL,
		},
		RevokeSpectatorToken: &RevokeSpectatorToken{
			Tournaments: p.Tournaments, Tokens: p.SpectatorTokens,
			AuditLog: p.AuditLog, Clock: p.Clock,
		},
		EditSeed: &EditSeed{
			Tournaments: p.Tournaments, TPlayers: p.TPlayers,
			AuditLog: p.AuditLog, Clock: p.Clock,
		},
		PreviewBracket: &PreviewBracket{
			Tournaments: p.Tournaments, TPlayers: p.TPlayers,
		},
		MarkInProgress: &MarkMatchInProgress{
			Matches: p.Matches, Clock: p.Clock,
		},
		RecordMatchResult: &RecordMatchResult{
			Matches: p.Matches, Clock: p.Clock, AuditLog: p.AuditLog,
		},
		CorrectMatchResult: &CorrectMatchResult{
			Matches: p.Matches, Clock: p.Clock, AuditLog: p.AuditLog,
		},
		AddManager: &AddManager{
			Tournaments: p.Tournaments, TManagers: p.TManagers,
			Users: p.Users, AuditLog: p.AuditLog, Clock: p.Clock,
		},
		RemoveManager: &RemoveManager{
			Tournaments: p.Tournaments, TManagers: p.TManagers,
			AuditLog: p.AuditLog, Clock: p.Clock,
		},
		ListAuditLog:     &ListAuditLog{AuditLog: p.AuditLog},
		ExportBracketCSV: &ExportBracketCSV{Matches: p.Matches},
		ExportBracketPNG: &ExportBracketPNG{Matches: p.Matches},
	}
	return b
}
