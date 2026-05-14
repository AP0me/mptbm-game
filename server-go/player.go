package main

import (
	"bufio"
	"fmt"
	"net"
	"strings"
	"time"
)

type Player struct {
	Name   string
	Conn   net.Conn
	Decide func(state *GameState) string
}

func HumanInput(conn net.Conn) string {
	SafeSend(conn, map[string]interface{}{}, "CHOICE")
	reader := bufio.NewReader(conn)
	input, _ := reader.ReadString('\n')
	return strings.TrimSpace(input)
}

func RobotInput(val string) string {
	time.Sleep(1 * time.Second)
	return val
}

func WelcomeHumansToPlayerList(playerOrder []string, ln net.Listener, players map[string]*Player) {
	for {
		remaining := []string{}
		for _, name := range playerOrder {
			if _, exists := players[name]; !exists {
				remaining = append(remaining, name)
			}
		}

		if len(remaining) == 0 {
			break
		}

		conn, err := ln.Accept()
		if err != nil {
			continue
		}

		name := remaining[0]
		players[name] = &Player{
			Name: name,
			Conn: conn,
			Decide: func(s *GameState) string {
				return HumanInput(conn)
			},
		}
		fmt.Printf("[Server] Player %s connected.\n", name)
	}
}

func PlayerConnections(players map[string]*Player) []net.Conn {
	conns := []net.Conn{}
	for _, p := range players {
		if p.Conn != nil {
			conns = append(conns, p.Conn)
		}
	}
	return conns
}
