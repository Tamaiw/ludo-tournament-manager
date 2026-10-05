# Ludo Tournament Manager

A web application for organising and running Ludo tournaments. Managers create tournaments, brackets auto-generate from registered players, and match results advance the bracket in real time.

## Language

**Tournament**:
A single-elimination bracket event with a fixed set of registered players, a start time, and a final winner.
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
A single game between two players within a tournament. Has a status, two participants (once determined), and a recorded outcome that advances the bracket.
_Avoid_: Game, round — a Round is the set of Matches at the same depth in the bracket

**Bracket**:
The full single-elimination tree for a tournament: every match, its dependency on previous matches, and the path to the final.
_Avoid_: Tree, ladder

**Invite**:
The mechanism for creating a new user. A Manager enters an email; the system emails a token link; clicking it lets the recipient set a password.
_Avoid_: Signup, registration — those terms refer to joining a tournament, not creating an account