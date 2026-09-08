package shared

import (
	"fmt"
	"net"
	"slices"
	"time"
)

type GameState struct {
	Data map[string]any
}

type Player struct {
	Name   string
	Conn   net.Conn
	Decide func(state *GameState) string
}

type Card struct {
	Name       string
	Conditions func(state *GameState) bool
	Action     func(state *GameState)
}

type HostPlayer struct {
	Name string `json:"name"`
}

type CardProfile struct {
	Name string `json:"name"`
}

func GetInt(state *GameState, key string) int {
	if state.Data == nil {
		return 0
	}

	switch val := state.Data[key].(type) {
	case int:
		return int(val)
	case float64:
		return int(val)
	case int64:
		return int(val)
	case float32:
		return int(val)
	default:
		return 0
	}
}

func GetFloat(state *GameState, key string) float64 {
	if state.Data == nil {
		return 0
	}

	switch val := state.Data[key].(type) {
	case int:
		return float64(val)
	case float64:
		return float64(val)
	case int64:
		return float64(val)
	case float32:
		return float64(val)
	default:
		return 0
	}
}

func GetString(state *GameState, key string) string {
	if val, ok := state.Data[key].(string); ok {
		return val
	}
	return ""
}

func GetStringList(state *GameState, key string) []string {
	if val, ok := state.Data[key].([]any); ok {
		list := make([]string, len(val))
		for i, v := range val {
			list[i] = fmt.Sprintf("%v", v)
		}
		return list
	} else {
		if val, ok := state.Data[key].([]string); ok {
			return val
		}
		return []string{}
	}
}

func GetBool(state *GameState, key string) bool {
	if val, ok := state.Data[key].(bool); ok {
		return val
	}
	return false
}

func GetActingPlayer(s *GameState) string {
	return GetString(s, "acting_player")
}

func StateInvisibleSet(s *GameState, key string, val any) {
	if key == "" { return }

	if val == nil {
		delete(s.Data, key)
		return
	}

	s.Data[key] = val
}

func StateSet(s *GameState, key string, val any) {
	if (val != nil) {
		display_order := GetStringList(s, "display_order")
		if !slices.Contains(display_order, key) {
			s.Data["display_order"] = append(display_order, key)
		}
	}
	StateInvisibleSet(s, key, val)
}

func LogEvent(s *GameState, msg string) {
	logs := GetStringList(s, "event_logs")
	StateInvisibleSet(s, "event_logs", append(logs, msg))
}

func ClearEventLogs(s *GameState) {
	StateInvisibleSet(s, "event_logs", []string{})
}

func RemovePlayers(s *GameState, playerKeys []string) {
	order := GetStringList(s, "player_order")

	var newOrder []string
	for _, player := range order {
		if !slices.Contains(playerKeys, player) {
			newOrder = append(newOrder, player)
		}
	}

	StateSet(s, "player_order", newOrder)
}

func EndTurn(s *GameState) {
	order := GetStringList(s, "player_order")
	current := GetActingPlayer(s)
	nextIdx := 0
	for i, name := range order {
		if name == current {
			nextIdx = (i + 1) % len(order)
			break
		}
	}
	StateInvisibleSet(s, "acting_player", order[nextIdx])
}

func RobotKeys(robots map[string]*Player) []string {
	keys := []string{}
	for k := range robots {
		keys = append(keys, k)
	}
	return keys
}

func RobotInput(val string) string {
	time.Sleep(1 * time.Second)
	return val
}

func PropedSet(s *GameState, key string, val int) {
	if val < 0 {
		val = 0
	}
	StateSet(s, key, val)
}


