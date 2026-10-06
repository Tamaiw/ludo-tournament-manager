# Ludo Tournament Manager

A web application for organising and running Ludo tournaments. Managers create tournaments, brackets auto-generate from registered players, and match results advance the bracket in real time.

## Language

**Tournament**:
A multi-player elimination event with a fixed set of registered players, a start time, and a final winner. Each match in the bracket has 2-4 participants (a single Ludo game played on a physical board); the manager decides how many finishers advance from each match, and the final round has exactly 4 players whose finishing positions determine 1st/2nd/3rd/4th place.
_Avoid_: Event, contest, league

**Manager**:
A user who organises one or more tournaments. The same role as "admin" in casual usage; can also register as a player in other tournaments.
_Avoid_: Admin, organizer — the canonical term here is "Manager"

**Player**:
A user who has registered for a specific tournament and competes in its bracket.
_Avoid_: Participant, contestant, competitor

**Spectator**:
Anyone viewing a tournament. Spectators do not need an account; they have read-only access to bracket state, results, and standings.
_Avoid_: Viewer, observer

**Match**:
A single Ludo game within a tournament. Has 2 to 4 participants, a status, and a recorded outcome that advances the bracket. Each Ludo game is played on a physical board; the app records the outcome, it does not run the game.
_Avoid_: Game, round — a Round is the set of Matches at the same depth in the bracket

**Round**:
The set of matches at the same depth in the bracket. A tournament progresses round by round, with the top finishers from each match in round N advancing to round N+1.
_Avoid_: Level, stage

**Bracket**:
The full elimination structure for a tournament: every match, its dependency on previous matches, and the path to the final. In this app, a bracket is a multi-player elimination where each match has 2-4 players and the manager decides how many finishers advance from each match.
_Avoid_: Tree, ladder

**Match Participant**:
A player's entry in a match, recording their finishing position (1 = winner, 2 = second, 3 = third, 4 = fourth). Positions for players who do not advance are optional and may go unrecorded.
_Avoid_: Match entry, slot

**Advancing Position**:
The finishing position a participant earned in a match. Participants with advancing positions 1 through N (where N is the per-round advance count) proceed to the next round; the rest are eliminated.
_Avoid_: Rank, place

**Bye**:
A player who is automatically advanced to the next round without playing a match. Byes are not matches; they exist only when player count does not divide cleanly into the configured game size.
_Avoid_: Skip, pass

**Seed**:
A number assigned to a registered player that determines their position in the bracket when the tournament starts. Lower numbers indicate stronger players (seed 1 is the top seed). When the manager does not assign seeds, the system generates them randomly at bracket-generation time. Manager edits to seeds are recorded in the audit log.
_Avoid_: Ranking, rating

**Tournament Audit Log**:
A record of every state-changing action taken on a tournament (seed changes, manager roster changes, registration transitions, match results, cancellations). Visible to managers; not exposed to spectators.
_Avoid_: History (ambiguous — also used for player match history)