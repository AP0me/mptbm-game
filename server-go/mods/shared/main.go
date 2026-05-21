package shared

import "net"

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

