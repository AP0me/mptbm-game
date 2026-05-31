package core

import (
	"bufio"
	"fmt"
	"game/server/mods/shared"
	"net"
	"strings"
)

func HumanInput(conn net.Conn) string {
	SafeSend(conn, map[string]interface{}{}, "CHOICE")
	reader := bufio.NewReader(conn)
	input, _ := reader.ReadString('\n')
	return strings.TrimSpace(input)
}

func WelcomeHumansToPlayerList(playerOrder []string, ln net.Listener, players map[string]*shared.Player, robots map[string]shared.HostPlayer) {
	for {
		remaining := []string{}
		for _, name := range playerOrder {
			_, isRobot := robots[name]
			_, exists := players[name]
			if !isRobot && !exists {
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
		players[name] = &shared.Player{
			Name: name,
			Conn: conn,
			Decide: nil,
		}

		for robot_key, robot := range robots {
			players[robot_key] = &shared.Player{Name: robot.Name, Conn: nil, Decide: nil}
		}

		fmt.Printf("[Server] Player %s connected.\n", name)
	}
}

func PlayerConnections(players map[string]*shared.Player) []net.Conn {
	conns := []net.Conn{}
	for _, p := range players {
		if p.Conn != nil {
			conns = append(conns, p.Conn)
		}
	}
	return conns
}
