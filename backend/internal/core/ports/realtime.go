package ports

import (
	"context"
	"encoding/json"
)

// EventType is the closed enum of events the broadcaster sends.
type EventType string

const (
	EventMatchStatusChanged         EventType = "match_status_changed"
	EventResultRecorded             EventType = "result_recorded"
	EventBracketRebuilt             EventType = "bracket_rebuilt"
	EventTournamentLifecycleChanged EventType = "tournament_lifecycle_changed"
)

const SchemaVersion = 1

// Event is the wire envelope for every broadcast.
type Event struct {
	SchemaVersion int             `json:"schema_version"`
	Type          EventType       `json:"type"`
	TournamentID  string          `json:"tournament_id"`
	Payload       json.RawMessage `json:"payload,omitempty"`
}

// Broadcaster publishes events to per-tournament WebSocket rooms.
type Broadcaster interface {
	Broadcast(ctx context.Context, event Event) error
	Register(tournamentID string, sub Subscription)
}

// Subscription is a per-connection handle the broadcaster uses to push events.
type Subscription interface {
	Send(event Event)
	Close()
}
