// Package ws is the inbound WebSocket adapter. It exposes a Broadcaster port
// implementation backed by a per-tournament hub. The full implementation is
// built in T16; this stub exists so T01's composition root can wire a hub
// reference without breaking the build.
package ws

import (
	"context"

	"github.com/Tamaiw/ludo-tournament-manager/backend/internal/core/ports"
)

// Hub is the per-tournament room registry.
type Hub struct{}

// NewHub returns an empty hub. T16 expands this into a full per-tournament
// room registry with buffered send channels and a heartbeat loop.
func NewHub() *Hub { return &Hub{} }

// Broadcast is a no-op until T16 wires real clients.
func (h *Hub) Broadcast(ctx context.Context, event ports.Event) error { return nil }

// Register is a no-op until T16.
func (h *Hub) Register(tournamentID string, s ports.Subscription) {}

// Unregister is a no-op until T16.
func (h *Hub) Unregister(tournamentID string, s ports.Subscription) {}