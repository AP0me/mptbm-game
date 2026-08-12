package shared

import (
	"net"
)

// GameState is the top-level state container.
// It wraps a DirectedTree (progression) and EventLogs (messages).
type GameState struct {
	Tree      *DirectedTree `json:"tree"`
	EventLogs []string      `json:"event_logs"`

	// — Engine-level fields (kept for compatibility) —
	Status       string   `json:"status,omitempty"`
	PlayerOrder  []string `json:"player_order,omitempty"`
	ActingPlayer string   `json:"acting_player,omitempty"`
}

// Player represents a connected human or a robot (WASM-side decision).
type Player struct {
	Name   string
	Conn   net.Conn
	Decide func(state *GameState) string
}

// Card is the engine's representation of a playable option.
type Card struct {
	Name       string
	Conditions func(state *GameState) bool
	Action     func(state *GameState)
}

// HostPlayer is sent to the client so it knows which seats are robots.
type HostPlayer struct {
	Name string `json:"name"`
}

// CardProfile is the slim card description sent to clients for display.
type CardProfile struct {
	Name string `json:"name"`
}

// ---------------------------------------------------------------------------
// Event-log helpers (the only state mutations the story mod needs)
// ---------------------------------------------------------------------------

func LogEvent(s *GameState, msg string) {
	s.EventLogs = append(s.EventLogs, msg)
}

func ClearEventLogs(s *GameState) {
	s.EventLogs = []string{}
}

// ---------------------------------------------------------------------------
// Deprecated flat-map helpers — kept for backward compatibility but the
// story mod does NOT use them.  If you need typed access, use the tree.
// ---------------------------------------------------------------------------

func GetStringList(s *GameState, key string) []string {
	// Only works for top-level keys that are []string in a hypothetical
	// flat map.  The tree-based state doesn't store flat maps, so this
	// is a no-op stub.  Kept so the engine compiles if it references it.
	_ = key
	return []string{}
}

func GetActingPlayer(s *GameState) string {
	return s.ActingPlayer
}
