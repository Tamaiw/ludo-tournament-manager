// Package ws is the inbound WebSocket adapter. It exposes a Broadcaster port
// implementation backed by a per-tournament hub.
package ws

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"

	"github.com/Tamaiw/ludo-tournament-manager/backend/internal/core/ports"
	"github.com/coder/websocket"
)

// Hub is the per-tournament room registry.
type Hub struct {
	mu   sync.RWMutex
	room map[string]map[*subscriber]struct{}
}

type subscriber struct {
	tournamentID string
	conn         *websocket.Conn
	hub          *Hub
	mu           sync.Mutex
	closed       bool
}

// Send delivers the event to the subscriber's connection.
func (s *subscriber) Send(event ports.Event) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return
	}
	data, err := json.Marshal(event)
	if err != nil {
		return
	}
	ctx := context.Background()
	_ = s.conn.Write(ctx, websocket.MessageText, data)
}

// Close marks the subscriber as closed.
func (s *subscriber) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return
	}
	s.closed = true
	_ = s.conn.Close(websocket.StatusNormalClosure, "")
}

// NewHub returns an empty hub.
func NewHub() *Hub { return &Hub{room: map[string]map[*subscriber]struct{}{}} }

// Broadcast delivers the event to every subscriber in the tournament's room.
func (h *Hub) Broadcast(ctx context.Context, event ports.Event) error {
	h.mu.RLock()
	subs := make([]*subscriber, 0, len(h.room[event.TournamentID]))
	for c := range h.room[event.TournamentID] {
		subs = append(subs, c)
	}
	h.mu.RUnlock()
	for _, c := range subs {
		c.Send(event)
	}
	return nil
}

// Register adds a connection to the tournament's room.
func (h *Hub) Register(tournamentID string, sub ports.Subscription) {
	s, ok := sub.(*subscriber)
	if !ok {
		return
	}
	h.mu.Lock()
	if _, ok := h.room[tournamentID]; !ok {
		h.room[tournamentID] = map[*subscriber]struct{}{}
	}
	h.room[tournamentID][s] = struct{}{}
	h.mu.Unlock()
}

// Unregister removes a connection from the tournament's room.
func (h *Hub) Unregister(tournamentID string, sub ports.Subscription) {
	s, ok := sub.(*subscriber)
	if !ok {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if room, ok := h.room[tournamentID]; ok {
		delete(room, s)
		if len(room) == 0 {
			delete(h.room, tournamentID)
		}
	}
	s.Close()
}

// Upgrade accepts an HTTP request and returns a Subscription for the upgrade.
func (h *Hub) Upgrade(w http.ResponseWriter, r *http.Request, tournamentID string) (ports.Subscription, error) {
	conn, err := websocket.Accept(w, r, nil)
	if err != nil {
		return nil, err
	}
	sub := &subscriber{
		tournamentID: tournamentID,
		conn:         conn,
		hub:          h,
	}
	go h.readLoop(sub)
	return sub, nil
}

func (h *Hub) readLoop(s *subscriber) {
	defer h.Unregister(s.tournamentID, s)
	for {
		_, _, err := s.conn.Read(context.Background())
		if err != nil {
			return
		}
		// We don't act on inbound messages; the protocol is server-push only.
	}
}
