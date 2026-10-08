// Package domain contains the pure types and invariants of the Ludo Tournament Manager.
// Domain has no I/O and no project imports outside this package.
package domain

import (
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

// ---- IDs ----

type UserID string

func NewUserID() UserID           { return UserID(uuid.NewString()) }
func (u UserID) String() string   { return string(u) }
func ParseUserID(s string) UserID { return UserID(s) }

type TournamentID string

func NewTournamentID() TournamentID           { return TournamentID(uuid.NewString()) }
func (t TournamentID) String() string         { return string(t) }
func ParseTournamentID(s string) TournamentID { return TournamentID(s) }

type MatchID string

func NewMatchID() MatchID           { return MatchID(uuid.NewString()) }
func (m MatchID) String() string    { return string(m) }
func ParseMatchID(s string) MatchID { return MatchID(s) }

type SpectatorTokenID string

func NewSpectatorTokenID() SpectatorTokenID { return SpectatorTokenID(uuid.NewString()) }
func (s SpectatorTokenID) String() string   { return string(s) }

type AuthTokenID string

func NewAuthTokenID() AuthTokenID    { return AuthTokenID(uuid.NewString()) }
func (a AuthTokenID) String() string { return string(a) }

type Slot string

const (
	SlotHome   Slot = "home"
	SlotAway   Slot = "away"
	SlotThird  Slot = "third"
	SlotFourth Slot = "fourth"
)

func (s Slot) Valid() bool {
	switch s {
	case SlotHome, SlotAway, SlotThird, SlotFourth:
		return true
	}
	return false
}

func ParseSlot(s string) (Slot, error) {
	sl := Slot(s)
	if !sl.Valid() {
		return "", errors.New("invalid slot: " + s)
	}
	return sl, nil
}

// ---- Enums ----

type TournamentStatus string

const (
	StatusDraft            TournamentStatus = "draft"
	StatusRegistrationOpen TournamentStatus = "registration_open"
	StatusInProgress       TournamentStatus = "in_progress"
	StatusCompleted        TournamentStatus = "completed"
	StatusCancelled        TournamentStatus = "cancelled"
)

func (s TournamentStatus) Valid() bool {
	switch s {
	case StatusDraft, StatusRegistrationOpen, StatusInProgress, StatusCompleted, StatusCancelled:
		return true
	}
	return false
}

type TournamentFormat string

const (
	FormatMultiPlayerElimination TournamentFormat = "multi_player_elimination"
)

func (f TournamentFormat) Valid() bool { return f == FormatMultiPlayerElimination }

type Visibility string

const (
	VisibilityPublic   Visibility = "public"
	VisibilityUnlisted Visibility = "unlisted"
	VisibilityPrivate  Visibility = "private"
)

func (v Visibility) Valid() bool {
	switch v {
	case VisibilityPublic, VisibilityUnlisted, VisibilityPrivate:
		return true
	}
	return false
}

type RegistrationMode string

const (
	RegModeInviteOnly   RegistrationMode = "invite_only"
	RegModeSelfRegister RegistrationMode = "self_register"
)

func (m RegistrationMode) Valid() bool {
	return m == RegModeInviteOnly || m == RegModeSelfRegister
}

type MatchStatus string

const (
	MatchPending    MatchStatus = "pending"
	MatchReady      MatchStatus = "ready"
	MatchInProgress MatchStatus = "in_progress"
	MatchCompleted  MatchStatus = "completed"
)

func (s MatchStatus) Valid() bool {
	switch s {
	case MatchPending, MatchReady, MatchInProgress, MatchCompleted:
		return true
	}
	return false
}

type AuthTokenKind string

const (
	AuthTokenInvite        AuthTokenKind = "invite"
	AuthTokenPasswordReset AuthTokenKind = "password_reset"
)

func (k AuthTokenKind) Valid() bool {
	return k == AuthTokenInvite || k == AuthTokenPasswordReset
}

type AuditAction string

const (
	ActionSeedChanged             AuditAction = "seed_changed"
	ActionManagerAdded            AuditAction = "manager_added"
	ActionManagerRemoved          AuditAction = "manager_removed"
	ActionRegistrationOpened      AuditAction = "registration_opened"
	ActionRegistrationClosed      AuditAction = "registration_closed"
	ActionRegistrationModeChanged AuditAction = "registration_mode_changed"
	ActionVisibilityChanged       AuditAction = "visibility_changed"
	ActionSpectatorTokenIssued    AuditAction = "spectator_token_issued"
	ActionSpectatorTokenRevoked   AuditAction = "spectator_token_revoked"
	ActionTournamentStarted       AuditAction = "tournament_started"
	ActionMatchResultRecorded     AuditAction = "match_result_recorded"
	ActionMatchResultCorrected    AuditAction = "match_result_corrected"
	ActionTournamentCancelled     AuditAction = "tournament_cancelled"
	ActionTournamentCompleted     AuditAction = "tournament_completed"
	ActionPlayerAdded             AuditAction = "player_added"
	ActionPlayerRemoved           AuditAction = "player_removed"
	ActionPlayerWithdrew          AuditAction = "player_withdrew"
	ActionPlayerInvited           AuditAction = "player_invited"
	ActionInviteAccepted          AuditAction = "invite_accepted"
	ActionPasswordResetIssued     AuditAction = "password_reset_issued"
	ActionPasswordResetRedeemed   AuditAction = "password_reset_redeemed"
	ActionPasswordChanged         AuditAction = "password_changed"
	ActionTournamentEdited        AuditAction = "tournament_edited"
	ActionTournamentCreated       AuditAction = "tournament_created"
)

func (a AuditAction) Valid() bool {
	switch a {
	case ActionSeedChanged, ActionManagerAdded, ActionManagerRemoved,
		ActionRegistrationOpened, ActionRegistrationClosed, ActionRegistrationModeChanged,
		ActionVisibilityChanged, ActionSpectatorTokenIssued, ActionSpectatorTokenRevoked,
		ActionTournamentStarted, ActionMatchResultRecorded, ActionMatchResultCorrected,
		ActionTournamentCancelled, ActionTournamentCompleted, ActionPlayerAdded,
		ActionPlayerRemoved, ActionPlayerWithdrew, ActionPlayerInvited,
		ActionInviteAccepted, ActionPasswordResetIssued, ActionPasswordResetRedeemed,
		ActionPasswordChanged, ActionTournamentEdited, ActionTournamentCreated:
		return true
	}
	return false
}

// ---- Errors ----

var (
	ErrEmptyName                        = errors.New("name must not be empty")
	ErrNameTooLong                      = errors.New("name must be 120 characters or fewer")
	ErrInvalidStatus                    = errors.New("invalid tournament status")
	ErrInvalidFormat                    = errors.New("invalid tournament format")
	ErrInvalidVisibility                = errors.New("invalid visibility")
	ErrInvalidRegistrationMode          = errors.New("invalid registration mode")
	ErrInvalidSlot                      = errors.New("invalid slot")
	ErrInvalidMatchStatus               = errors.New("invalid match status")
	ErrNotDraft                         = errors.New("tournament is not in draft")
	ErrNotRegistrationOpen              = errors.New("tournament is not open for registration")
	ErrNotInProgress                    = errors.New("tournament is not in progress")
	ErrBracketLocked                    = errors.New("tournament bracket is locked")
	ErrAlreadyStarted                   = errors.New("tournament has already been started")
	ErrNotCancellable                   = errors.New("tournament can only be cancelled from draft or registration_open")
	ErrInvalidAdvanceMap                = errors.New("invalid players_advancing_per_round map")
	ErrSeedOutOfRange                   = errors.New("seed out of range")
	ErrSeedInUse                        = errors.New("seed already in use by another player")
	ErrDownstreamLocked                 = errors.New("downstream match has been played; result is locked")
	ErrUserExists                       = errors.New("user already exists")
	ErrUserNotFound                     = errors.New("user not found")
	ErrTournamentNotFound               = errors.New("tournament not found")
	ErrMatchNotFound                    = errors.New("match not found")
	ErrTokenNotFound                    = errors.New("token not found")
	ErrTokenExpired                     = errors.New("token has expired")
	ErrTokenUsed                        = errors.New("token has already been used")
	ErrNotParticipant                   = errors.New("actor is not a participant in this match")
	ErrWrongMatchStatus                 = errors.New("match is not in the expected status")
	ErrInvalidAdvancingPositions        = errors.New("invalid advancing positions")
	ErrAtCap                            = errors.New("tournament is at maximum player capacity")
	ErrNotRegistered                    = errors.New("user is not registered for this tournament")
	ErrInvalidRegistrationModeForAction = errors.New("invalid registration mode for this action")
)

// ---- Entity types ----

type User struct {
	ID           UserID
	Email        string
	Name         string
	PasswordHash string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func NewUser(email, name, passwordHash string, now time.Time) (User, error) {
	email = strings.TrimSpace(strings.ToLower(email))
	name = strings.TrimSpace(name)
	if email == "" {
		return User{}, errors.New("email must not be empty")
	}
	if !strings.Contains(email, "@") {
		return User{}, errors.New("invalid email")
	}
	if name == "" {
		return User{}, errors.New("name must not be empty")
	}
	if passwordHash == "" {
		return User{}, errors.New("password hash must not be empty")
	}
	return User{
		ID:           NewUserID(),
		Email:        email,
		Name:         name,
		PasswordHash: passwordHash,
		CreatedAt:    now,
		UpdatedAt:    now,
	}, nil
}

type Tournament struct {
	ID                       TournamentID
	Name                     string
	Description              string
	Format                   TournamentFormat
	Status                   TournamentStatus
	MinPlayersPerMatch       int
	MaxPlayersPerMatch       int
	PlayersAdvancingPerRound map[string]int
	ScheduledStartAt         *time.Time
	Visibility               Visibility
	RegistrationMode         RegistrationMode
	MaxPlayers               int
	CreatedBy                UserID
	CreatedAt                time.Time
	UpdatedAt                time.Time
	StartedAt                *time.Time
	CompletedAt              *time.Time
	CancelledAt              *time.Time
	BracketPRNGSeed          *int64
}

func (t Tournament) IsTerminal() bool {
	return t.Status == StatusCompleted || t.Status == StatusCancelled
}

func NewTournament(
	name, description string,
	minPerMatch, maxPerMatch int,
	advanceMap map[string]int,
	scheduled *time.Time,
	visibility Visibility,
	regMode RegistrationMode,
	maxPlayers int,
	createdBy UserID,
	now time.Time,
) (Tournament, error) {
	name = strings.TrimSpace(name)
	description = strings.TrimSpace(description)
	if name == "" {
		return Tournament{}, ErrEmptyName
	}
	if len(name) > 120 {
		return Tournament{}, ErrNameTooLong
	}
	if minPerMatch < 2 || minPerMatch > 4 {
		return Tournament{}, errors.New("min_players_per_match must be between 2 and 4")
	}
	if maxPerMatch < minPerMatch || maxPerMatch > 4 {
		return Tournament{}, errors.New("max_players_per_match must be in [min,4]")
	}
	if advanceMap == nil {
		advanceMap = map[string]int{"1": 1, "2": 1}
	}
	for k, v := range advanceMap {
		r, err := strconv.Atoi(k)
		if err != nil || r < 1 {
			return Tournament{}, ErrInvalidAdvanceMap
		}
		if v < 1 || v > maxPerMatch {
			return Tournament{}, ErrInvalidAdvanceMap
		}
	}
	if !visibility.Valid() {
		return Tournament{}, ErrInvalidVisibility
	}
	if !regMode.Valid() {
		return Tournament{}, ErrInvalidRegistrationMode
	}
	if maxPlayers < 0 {
		return Tournament{}, errors.New("max_players must be >= 0")
	}
	return Tournament{
		ID:                       NewTournamentID(),
		Name:                     name,
		Description:              description,
		Format:                   FormatMultiPlayerElimination,
		Status:                   StatusDraft,
		MinPlayersPerMatch:       minPerMatch,
		MaxPlayersPerMatch:       maxPerMatch,
		PlayersAdvancingPerRound: advanceMap,
		ScheduledStartAt:         scheduled,
		Visibility:               visibility,
		RegistrationMode:         regMode,
		MaxPlayers:               maxPlayers,
		CreatedBy:                createdBy,
		CreatedAt:                now,
		UpdatedAt:                now,
	}, nil
}

func (t Tournament) OpenRegistration(now time.Time) (Tournament, error) {
	if t.Status != StatusDraft {
		return t, ErrNotDraft
	}
	t.Status = StatusRegistrationOpen
	t.UpdatedAt = now
	return t, nil
}

func (t Tournament) CloseRegistration(now time.Time) (Tournament, error) {
	if t.Status != StatusRegistrationOpen {
		return t, ErrNotRegistrationOpen
	}
	t.Status = StatusDraft
	t.UpdatedAt = now
	return t, nil
}

func (t Tournament) Start(now time.Time, prngSeed int64) (Tournament, error) {
	if t.Status != StatusRegistrationOpen {
		return t, ErrNotRegistrationOpen
	}
	t.Status = StatusInProgress
	t.StartedAt = &now
	t.UpdatedAt = now
	t.BracketPRNGSeed = &prngSeed
	return t, nil
}

func (t Tournament) ForceClose(now time.Time) (Tournament, error) {
	if t.Status != StatusInProgress {
		return t, ErrNotInProgress
	}
	t.Status = StatusCompleted
	t.CompletedAt = &now
	t.UpdatedAt = now
	return t, nil
}

func (t Tournament) Cancel(now time.Time) (Tournament, error) {
	if t.Status != StatusDraft && t.Status != StatusRegistrationOpen {
		return t, ErrNotCancellable
	}
	t.Status = StatusCancelled
	t.CancelledAt = &now
	t.UpdatedAt = now
	return t, nil
}

func (t Tournament) Complete(now time.Time) (Tournament, error) {
	if t.Status != StatusInProgress {
		return t, ErrNotInProgress
	}
	t.Status = StatusCompleted
	t.CompletedAt = &now
	t.UpdatedAt = now
	return t, nil
}

// ---- TournamentPlayer ----

type TournamentPlayer struct {
	TournamentID TournamentID
	PlayerID     UserID
	RegisteredAt time.Time
	Seed         *int
}

// ---- Match / MatchParticipant ----

type Match struct {
	ID              MatchID
	TournamentID    TournamentID
	Round           int
	PositionInRound int
	Status          MatchStatus
	ScheduledAt     *time.Time
	NextMatchID     *MatchID
	SlotInNextMatch *Slot
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type MatchParticipant struct {
	MatchID           MatchID
	Slot              Slot
	PlayerID          *UserID
	AdvancingPosition *int
	IsBye             bool
}

// ---- AuthToken ----

type AuthToken struct {
	ID        AuthTokenID
	Kind      AuthTokenKind
	UserID    *UserID
	Email     string
	TokenHash string
	ExpiresAt time.Time
	UsedAt    *time.Time
	CreatedBy *UserID
	CreatedAt time.Time
}

func (a AuthToken) IsUsable(now time.Time) error {
	if a.UsedAt != nil {
		return ErrTokenUsed
	}
	if !now.Before(a.ExpiresAt) {
		return ErrTokenExpired
	}
	return nil
}

// ---- SpectatorToken ----

type SpectatorToken struct {
	ID           SpectatorTokenID
	TournamentID TournamentID
	TokenHash    string
	Label        string
	IssuedAt     time.Time
	IssuedBy     UserID
	RevokedAt    *time.Time
	RevokedBy    *UserID
}

func (s SpectatorToken) IsActive(now time.Time) bool {
	return s.RevokedAt == nil
}

// ---- AuditLogEntry ----

type AuditLogEntry struct {
	ID           AuditLogID
	TournamentID TournamentID
	ActorID      *UserID
	Action       AuditAction
	SubjectID    string
	Before       string // JSON
	After        string // JSON
	RecordedAt   time.Time
}

type AuditLogID string

func NewAuditLogID() AuditLogID { return AuditLogID(uuid.NewString()) }

// ---- Bracket ----

// Bracket is the per-round layout returned by the pure generator.
type Bracket struct {
	TotalRounds  int
	FinalFour    bool
	TotalPlayers int
	Rounds       []RoundShape
}

type RoundShape struct {
	Round       int
	Matches     int
	PlayersIn   int
	AdvanceMap  map[string]int // per-slot advance count for this round
	Description string
}

// ---- Time helpers ----

// FormatTime returns an ISO-8601 UTC time string suitable for SQLite.
func FormatTime(t time.Time) string {
	return t.UTC().Format("2006-01-02T15:04:05.000000Z")
}

func ParseTime(s string) (time.Time, error) {
	if s == "" {
		return time.Time{}, nil
	}
	return time.Parse("2006-01-02T15:04:05.000000Z", s)
}

func FormatTimePtr(t *time.Time) *string {
	if t == nil {
		return nil
	}
	s := FormatTime(*t)
	return &s
}

func ParseTimePtr(s *string) *time.Time {
	if s == nil || *s == "" {
		return nil
	}
	t, err := ParseTime(*s)
	if err != nil {
		return nil
	}
	return &t
}

func FormatIntPtr(p *int) *string {
	if p == nil {
		return nil
	}
	s := strconv.Itoa(*p)
	return &s
}

func ParseIntPtr(s string) *int {
	if s == "" {
		return nil
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return nil
	}
	return &v
}

func FormatInt64Ptr(p *int64) *string {
	if p == nil {
		return nil
	}
	s := strconv.FormatInt(*p, 10)
	return &s
}

func ParseInt64Ptr(s string) *int64 {
	if s == "" {
		return nil
	}
	v, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return nil
	}
	return &v
}
