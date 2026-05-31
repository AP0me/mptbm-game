package shared

import (
	"fmt"
	"net"
)

type GameState struct {
	Data map[string]interface{}
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

func GetInt(state *GameState, key string) int {
	if val, ok := state.Data[key].(int); ok {
		return val
	}
	return 0
}

func GetString(state *GameState, key string) string {
	if val, ok := state.Data[key].(string); ok {
		return val
	}
	return ""
}

func GetStringList(state *GameState, key string) []string {
	if val, ok := state.Data[key].([]interface{}); ok {
		list := make([]string, len(val))
		for i, v := range val {
			list[i] = fmt.Sprintf("%v", v)
		}
		return list
	}
	return []string{}
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
