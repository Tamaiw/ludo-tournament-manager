# Tournament visibility: three-tier model

A tournament has one of three visibility settings — `public`, `unlisted`, or `private` — chosen by a Manager and editable at any time. The tier controls who can reach and view the tournament; the matrix of permitted actions is in [`../permissions.md`](../permissions.md).

## Context

The CONTEXT glossary described Spectators as "anyone viewing a tournament, no account needed, read-only" — but said nothing about how a tournament decides who can reach it. A binary `is_public bool` (default true) was carried over from the original sketch and is too coarse: a club that wants to share a bracket link with regulars but not with the world either has to broadcast publicly or fully hide, with no middle ground. We need three: link-shareable but not listed, listed and open, and gated.

## Decision

1. **Three tiers, encoded as an enum.** A tournament has `visibility ∈ {public, unlisted, private}`. The column replaces the original `is_public bool`.
2. **`public`**: tournament appears in the public index and is reachable by anyone with the URL, with or without an account.
3. **`unlisted`**: tournament is *not* in the public index, but is reachable by anyone with the URL. No account needed. The URL is the only access control.
4. **`private`**: only managers, registered players, and holders of a valid Spectator Token can view. The Tournament URL alone returns 403.
5. **Managers can change visibility at any tournament state.** Transitions are recorded in the audit log. Effect on currently-reachable spectators is in [`../permissions.md`](../permissions.md#visibility-matrix).
7. **`private` tournaments use per-tournament Spectator Tokens** for non-account holders. Tokens are issued and revoked by Managers, are independent of the user account system, and have no associated user. See [`../permissions.md`](../permissions.md) and [`../domain-model.md`](../domain-model.md#tournament-spectator-token).
8. **Spectators are read-only at every tier.** No commenting, no reactions, no following. The three tiers differ only in *reach*, not in capability.

## Considered options

- **Two-tier (`is_public bool`)**: rejected. Forces clubs that want link-only access to either broadcast or fully hide; the link-only use case is real and common.
- **Two-tier (`public / unlisted`, no private)**: rejected. No way to run a club-internal tournament without exposing the URL to anyone who finds it; link leakage in a chat or screenshot would compromise the entire roster's privacy.
- **Four-tier (`public / unlisted / private / password`)**: rejected. A password on top of `private` adds friction without buying much: the Spectator Token already serves as a per-recipient password. A shared password would be shared, defeating the per-recipient revocation Spectator Tokens enable.
- **ACL-based sharing (manager invites specific users)**: rejected as the *primary* mechanism. Spectator Tokens are a lightweight version of this for non-account holders; building a full ACL on top is overkill for v1 and can be added in v2 without breaking the data model.
- **Visibility inherits from registration mode**: rejected. `private + self_register` is a valid combination (a club-internal tournament where current members can sign up to play but the world can't see it). Visibility and registration mode are orthogonal.

## Consequences

- **Schema change**: `tournaments.is_public` is replaced by `tournaments.visibility` (enum). Migration: any `is_public = true` row maps to `visibility = 'public'`; `false` rows default to `'private'` (the conservative choice — silent reclassification to `public` would be worse). The migration is part of the schema bootstrap in W3.
- **New entity**: `tournament_spectator_tokens`. See [`../domain-model.md`](../domain-model.md#tournament-spectator-token) for fields.
- **New audit-log actions**: `visibility_changed`, `spectator_token_issued`, `spectator_token_revoked`.
- **Reversibility**: changing the tier set later (e.g. to four tiers with a password variant) is a small additive migration. The tokens table is forward-compatible — a future "password" tier could live alongside them.