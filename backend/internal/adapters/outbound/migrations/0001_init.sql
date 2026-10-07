-- Initial schema for the Ludo Tournament Manager.
-- Schema source for sqlc.
-- The migration runner owns the schema_migrations table; do not CREATE it here.

CREATE TABLE users (
    id            TEXT PRIMARY KEY,
    email         TEXT NOT NULL UNIQUE,
    name          TEXT NOT NULL,
    password_hash TEXT NOT NULL,
    created_at    TEXT NOT NULL,
    updated_at    TEXT NOT NULL
);
CREATE INDEX users_email_idx ON users(email);

CREATE TABLE sessions (
    token_hash TEXT PRIMARY KEY,
    data       BLOB NOT NULL,
    expiry     REAL NOT NULL
);
CREATE INDEX sessions_expiry_idx ON sessions(expiry);

CREATE TABLE auth_tokens (
    id          TEXT PRIMARY KEY,
    kind        TEXT NOT NULL CHECK (kind IN ('invite','password_reset')),
    user_id     TEXT REFERENCES users(id) ON DELETE SET NULL,
    email       TEXT NOT NULL,
    token_hash  TEXT NOT NULL,
    expires_at  TEXT NOT NULL,
    used_at     TEXT,
    created_by  TEXT REFERENCES users(id) ON DELETE SET NULL,
    created_at  TEXT NOT NULL
);
CREATE INDEX auth_tokens_token_hash_idx ON auth_tokens(token_hash);
CREATE INDEX auth_tokens_email_idx ON auth_tokens(email);
CREATE INDEX auth_tokens_kind_idx ON auth_tokens(kind);

CREATE TABLE tournaments (
    id                          TEXT PRIMARY KEY,
    name                        TEXT NOT NULL,
    description                 TEXT NOT NULL DEFAULT '',
    format                      TEXT NOT NULL DEFAULT 'multi_player_elimination' CHECK (format IN ('multi_player_elimination')),
    status                      TEXT NOT NULL DEFAULT 'draft' CHECK (status IN ('draft','registration_open','in_progress','completed','cancelled')),
    min_players_per_match       INTEGER NOT NULL DEFAULT 2 CHECK (min_players_per_match >= 2 AND min_players_per_match <= 4),
    max_players_per_match       INTEGER NOT NULL DEFAULT 4 CHECK (max_players_per_match >= 2 AND max_players_per_match <= 4),
    players_advancing_per_round TEXT NOT NULL DEFAULT '{"1":1,"2":1}' CHECK (json_valid(players_advancing_per_round)),
    scheduled_start_at          TEXT,
    visibility                  TEXT NOT NULL DEFAULT 'private' CHECK (visibility IN ('public','unlisted','private')),
    registration_mode           TEXT NOT NULL DEFAULT 'invite_only' CHECK (registration_mode IN ('invite_only','self_register')),
    max_players                 INTEGER NOT NULL DEFAULT 0,
    created_by                  TEXT NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    created_at                  TEXT NOT NULL,
    updated_at                  TEXT NOT NULL,
    started_at                  TEXT,
    completed_at                TEXT,
    cancelled_at                TEXT,
    bracket_prng_seed           INTEGER,
    CONSTRAINT min_le_max CHECK (min_players_per_match <= max_players_per_match)
);
CREATE INDEX tournaments_status_idx ON tournaments(status);
CREATE INDEX tournaments_visibility_idx ON tournaments(visibility);
CREATE INDEX tournaments_started_at_idx ON tournaments(started_at);

CREATE TABLE tournament_manager (
    tournament_id TEXT NOT NULL REFERENCES tournaments(id) ON DELETE CASCADE,
    manager_id    TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    added_at      TEXT NOT NULL,
    PRIMARY KEY (tournament_id, manager_id)
);
CREATE INDEX tournament_manager_manager_idx ON tournament_manager(manager_id);

CREATE TABLE tournament_player (
    tournament_id TEXT NOT NULL REFERENCES tournaments(id) ON DELETE CASCADE,
    player_id     TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    registered_at TEXT NOT NULL,
    seed          INTEGER,
    PRIMARY KEY (tournament_id, player_id)
);
CREATE INDEX tournament_player_player_idx ON tournament_player(player_id);
CREATE UNIQUE INDEX tournament_player_seed_uniq
    ON tournament_player(tournament_id, seed)
    WHERE seed IS NOT NULL;

CREATE TABLE tournament_spectator_tokens (
    id            TEXT PRIMARY KEY,
    tournament_id TEXT NOT NULL REFERENCES tournaments(id) ON DELETE CASCADE,
    token_hash    TEXT NOT NULL,
    label         TEXT NOT NULL DEFAULT '',
    issued_at     TEXT NOT NULL,
    issued_by     TEXT NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    revoked_at    TEXT,
    revoked_by    TEXT REFERENCES users(id) ON DELETE SET NULL
);
CREATE INDEX tournament_spectator_tokens_tournament_idx ON tournament_spectator_tokens(tournament_id);
CREATE INDEX tournament_spectator_tokens_hash_idx ON tournament_spectator_tokens(token_hash);

CREATE TABLE matches (
    id                 TEXT PRIMARY KEY,
    tournament_id      TEXT NOT NULL REFERENCES tournaments(id) ON DELETE CASCADE,
    round              INTEGER NOT NULL CHECK (round >= 1),
    position_in_round  INTEGER NOT NULL CHECK (position_in_round >= 1),
    status             TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending','in_progress','completed')),
    scheduled_at       TEXT,
    next_match_id      TEXT REFERENCES matches(id) ON DELETE SET NULL,
    slot_in_next_match TEXT CHECK (slot_in_next_match IN ('home','away','third','fourth')),
    created_at         TEXT NOT NULL,
    updated_at         TEXT NOT NULL,
    UNIQUE (tournament_id, round, position_in_round)
);
CREATE INDEX matches_tournament_idx ON matches(tournament_id);
CREATE INDEX matches_next_match_idx ON matches(next_match_id);

CREATE TABLE match_participants (
    match_id           TEXT NOT NULL REFERENCES matches(id) ON DELETE CASCADE,
    slot               TEXT NOT NULL CHECK (slot IN ('home','away','third','fourth')),
    player_id          TEXT REFERENCES users(id) ON DELETE SET NULL,
    advancing_position INTEGER CHECK (advancing_position >= 1 AND advancing_position <= 4),
    is_bye             INTEGER NOT NULL DEFAULT 0 CHECK (is_bye IN (0,1)),
    PRIMARY KEY (match_id, slot)
);
CREATE INDEX match_participants_player_idx ON match_participants(player_id);

CREATE TABLE tournament_audit_log (
    id            TEXT PRIMARY KEY,
    tournament_id TEXT NOT NULL REFERENCES tournaments(id) ON DELETE CASCADE,
    actor_id      TEXT REFERENCES users(id) ON DELETE SET NULL,
    action        TEXT NOT NULL,
    subject_id    TEXT,
    before        TEXT,
    after         TEXT,
    recorded_at   TEXT NOT NULL
);
CREATE INDEX tournament_audit_log_tournament_idx ON tournament_audit_log(tournament_id, recorded_at DESC);