# Permissions Matrix

This document is the source of truth for *who can do what* in the Ludo Tournament Manager. The matrix is role × action; where an action depends on a tournament setting (visibility, registration mode) or lifecycle state, the dependency is called out.

Companion docs: [`../CONTEXT.md`](../CONTEXT.md) (glossary), [`../docs/domain-model.md`](./domain-model.md) (entities + relationships), [`../docs/adr/0003-tournament-visibility-tiers.md`](./adr/0003-tournament-visibility-tiers.md) (why three tiers, not two).

---

## Roles

| Role | Definition | Scope |
|------|------------|-------|
| **Manager** | A user who can edit, cancel, and run a specific tournament. All managers of a tournament are equal — no creator-elevated role (per W6). | Per-tournament |
| **Player** | A user registered for a specific tournament. | Per-tournament |
| **Spectator** | Anyone viewing a tournament. May or may not have an account; visibility settings decide what they can reach. | Tournament-wide (by visibility) |
| **Authenticated user** | A user with an account but no role in this tournament. May create tournaments and (in some modes) register as a player. | Global |
| **Anonymous** | Anyone hitting the site without an account. | Global |

A single user can be Manager of T₁ and Player of T₂ in the same week, and Manager **and** Player of T₃ simultaneously (a Manager may also register as a Player in the same tournament they manage — see [Manager-as-Player](#manager-as-player)).

---

## Tournament settings that gate permissions

These two tournament-level settings control who can reach and join the tournament. They appear as conditions in the matrix below.

### Visibility

Three tiers (see [ADR 0003](./adr/0003-tournament-visibility-tiers.md)):

- **`public`** — listed on the public tournaments index; viewable by anyone (logged in or not).
- **`unlisted`** — *not* listed, but anyone with the URL can view (logged in or not). For casual house games and small clubs that don't want to broadcast but are OK with link-sharing.
- **`private`** — only managers, players, and holders of a **spectator token** (see glossary) can view. For club-internal or invite-only events.

Managers can change a tournament's visibility at any time (see [Visibility matrix](#visibility-matrix)).

### Registration mode

Manager-set at creation, editable while `draft` or `registration_open`:

- **`invite_only`** — only managers can add players. The manager either registers an existing user, or sends a platform invite (which auto-adds the new user to the roster when accepted).
- **`self_register`** — any authenticated user can register themselves, subject to `max_players` and `registration_open` state. Managers can still remove players.

---

## Matrix

Legend: ✓ allowed · ✗ = denied · ⚙ = allowed under a condition spelled out below the row.

#### Global actions

| Action | Anonymous | Authenticated user | Manager | Player |
|--------|:---------:|:------------------:|:-------:|:------:|
| Sign in / sign up | ✗ (no self-signup; invite-only onboarding) | ✓ | ✓ | ✓ |
| View public tournament index | ✓ | ✓ | ✓ | ✓ |
| Create a tournament | ✗ | ✓ | ✓ | ✓ |
| Invite a new user (platform invite) | ✗ | ✗ | ✓ (any tournament the user manages) | ✗ |
| Manage own user profile (name, password) | n/a | ✓ | ✓ | ✓ |

#### Tournament discovery & viewing

| Action | Anonymous | Authenticated user | Manager | Player |
|--------|:---------:|:------------------:|:-------:|:------:|
| View `public` tournament | ✓ | ✓ | ✓ | ✓ |
| View `unlisted` tournament | ⚙ (only with the URL; not listed) | ⚙ (same) | ✓ | ✓ |
| View `private` tournament | ⚙ (only with a valid spectator token) | ⚙ (same) | ✓ | ✓ |
| Download bracket as CSV / image | ✓ | ✓ | ✓ | ✓ |

#### Manager roster

| Action | Manager |
|--------|:-------:|
| Add another manager | ✓ (any manager) |
| Remove another manager | ✓ (any manager; not blocked by last-manager — see note) |
| Self-remove from the manager list | ✗ unless `cancelled` (see [Manager self-removal](#manager-self-removal)) |

#### Tournament configuration (manager actions)

| Action | Manager |
|--------|:-------:|
| Edit tournament settings (name, description, format, max/min players, advance map, scheduled start) | ✓ (any state) |
| Change visibility (`public / unlisted / private`) | ✓ (any state; see [Visibility matrix](#visibility-matrix)) |
| Change registration mode (`invite_only / self_register`) | ⚙ (only while `draft` or `registration_open`) |
| Transition `draft → registration_open` | ✓ |
| Transition `registration_open → in_progress` (generates bracket) | ✓ |
| Transition `in_progress → completed` (force-close) | ✓ |
| Transition → `cancelled` | ⚙ (only valid before `in_progress`; see [Cancellation](#cancellation)) |
| Add a player (in `invite_only` mode, existing user) | ✓ |
| Add a player (in `invite_only` mode, via platform invite) | ✓ |
| Remove another player (before bracket generation) | ✓ |
| Edit a player's `seed` (before bracket generation) | ✓ (audit-logged) |

#### Player self-service

| Action | Authenticated user | Player | Manager |
|--------|:------------------:|:------:|:-------:|
| Register self as a player (in `self_register` mode, while `registration_open`, below `max_players`) | ✓ | ✓ | ✓ (Manager-as-Player; see [Manager-as-Player](#manager-as-player)) |
| Withdraw self from the roster (before bracket generation) | ✗ | ✓ | ✓ (only if also a player — Manager-as-Player) |

#### Bracket & match recording

| Action | Manager | Player |
|--------|:-------:|:------:|
| Mark a match `in_progress` | ✓ | ⚙ (only if a participant of that match) |
| Record a match outcome (`in_progress → completed`) | ✓ | ⚙ (only if a participant of that match) |
| Correct a completed match (edit `advancing_position`s) | ⚙ (only if no downstream match is `in_progress` or `completed`) | ✗ |
| Record a forfeit / no-show post-bracket-generation | n/a (handled by the match outcome flow — see [Withdrawal & forfeit](#withdrawal--forfeit)) | n/a |

#### Audit log

| Action | Manager | Player | Spectator |
|--------|:-------:|:------:|:---------:|
| Read the tournament audit log | ✓ | ✗ | ✗ |

The audit log is a manager-only tool. Players see what they did via their own match and dashboard surfaces, not the log.

---

## Conditions and notes

### Manager-as-Player

A Manager may also register as a Player in the **same** tournament they manage. The Manager row appears in both `tournament_manager` and `tournament_player` for that tournament — uniqueness is enforced on each table independently, not across them.

When a Manager records or corrects a result for a match they themselves participate in, the audit log records the action with the actor and a `"manager_self_record": true` flag so the dual role is visible to other managers reviewing the log.

### Manager removal

Managers are equal. Any Manager can remove any other Manager. The matrix protects only **self-removal**: a Manager may not drop themselves off the manager list except by cancelling the tournament (see [Cancellation](#cancellation)). This is to prevent an "I have a rule, but I want out" path that strands the tournament with no managers.

Removing a co-manager does **not** require their consent, but the audit log records the actor and the snapshot. Players are unaffected.

### Cancellation

Cancellation transitions the tournament to `cancelled` from `draft` or `registration_open`. Once matches have been played (`in_progress`), cancellation is **not** allowed; the manager must instead force-close to `completed`. This protects the historical record of played games.

Cancellation is also the only path for a Manager to leave a tournament without remaining as a manager (see [Manager removal](#manager-removal)). If a Manager wants to bow out of an active tournament, they must either (a) cancel the tournament (only valid pre-`in_progress`), or (b) add a successor Manager first and ask them to remove the original Manager. Note: per the previous paragraph, (b) requires a co-manager's *consent*, since the manager cannot unilaterally remove themselves; this is intentional friction.

### Visibility matrix

The manager can change visibility at any tournament state. Effect on currently-reachable spectators:

| Change | Effect |
|--------|--------|
| `public → unlisted` | Existing public-index entries are removed; users with the URL still see it. No data loss. |
| `public → private` | Public-index entry removed; users without a spectator token get a 403 if they navigate to the URL. Links break. |
| `unlisted → public` | Tournament now appears in the public index. Existing URL viewers unaffected. |
| `unlisted → private` | URL viewers without a spectator token now get 403. |
| `private → public` | All holders of a spectator token retain read access (URL access alone is enough). Spectator tokens become redundant; existing tokens can be left or revoked. |
| `private → unlisted` | Holders of a spectator token retain read access; URL access alone is also enough. |

The audit log records every transition with the previous and new visibility. Spectator tokens can be issued or revoked independently of visibility changes.

### Withdrawal & forfeit

- **Before the bracket is generated** (`draft` or `registration_open`): a Player can withdraw themselves; a Manager can remove any Player. If withdrawals drop the count below the minimum that the configured bracket geometry can handle, bracket generation will fail — the manager must adjust the advance map or cancel. The matrix does not block withdrawals; it just makes them surface a bracket-generation failure later.

- **After the bracket is generated** (`in_progress`): there is no separate "forfeit" action. The Player remains in `match_participant` rows. When the match is recorded, the absent Player is marked with `advancing_position = NULL` (the losers are advancing-positioned normally). Whoever records the match (winner, surviving player, or Manager) carries out the recording through the normal outcome flow. This is why the matrix's match-recording row covers forfeit implicitly.

### Manager-only cancel-only self-removal

A tournament's managers collectively own it from creation to completion. There is no mechanism for a Manager to unilaterally leave a tournament that is still in any non-terminal state. The paths off are:

1. The tournament transitions to `cancelled` (only valid pre-`in_progress`).
2. The Manager remains a Manager until the tournament reaches `completed`.
3. The Manager adds a Co-Manager, then asks that Co-Manager to remove the original Manager.

This is intentional friction. The alternative — "Manager can self-remove, last-manager removal is blocked" — would mean a Manager who doesn't want to manage must click "cancel" to leave (effectively destroying it) or be blocked from leaving at all. Allowing Manager self-removal with a last-manager guard would leave a tournament temporarily manager-less while the original Manager awaits a replacement. Keeping it cancel-only is the simplest, most predictable rule.

---

## Decisions log

Every "✓" in the matrix is backed by an explicit grilling decision. The full session is recorded in [`.wayfinder/tickets/W7-user-roles-permissions-matrix.md`](../.wayfinder/tickets/W7-user-roles-permissions-matrix.md).

| # | Decision | Choice |
|---|----------|--------|
| Q1 | Can a user be both Manager and Player of the same tournament? | **Yes, allowed** |
| Q2 | Visibility tiers | **Three: `public / unlisted / private`** (see [ADR 0003](./adr/0003-tournament-visibility-tiers.md)) |
| Q3 | Registration model | **Both `invite_only` and `self_register`, manager-toggled per tournament** |
| Q4 | Who can create a tournament? | **Any authenticated user** |
| Q5 | Match-outcome recording protocol | **Single-recorder, no confirmation window** (audit captures recorder) |
| Q6 | Spectator scope at each visibility tier | **Read-only at every tier; private uses per-tournament spectator tokens** |
| Q7 | Player withdrawal / forfeit | **Post-bracket forfeits recorded through the match outcome (no separate action); pre-bracket withdrawal is its own action** |
| Q8 | Invite-new-users scope | **Manager-only platform invite; tournament-context auto-add via `registration_mode`** |
| Q9 | Manager self-removal | **Only via cancel** |
| Q10 | Visibility transitions | **Anytime, with state-aware default behaviour for existing spectators** |
| Q11 | Audit log access scope | **Managers only; players never see the audit log** |