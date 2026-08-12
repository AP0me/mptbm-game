package core

import (
	"bufio"
	"fmt"
	"net"
	"strings"
)

func HumanInput(conn net.Conn) string {
	SafeSend(conn, map[string]any{}, "CHOICE")
	reader := bufio.NewReader(conn)
	input, _ := reader.ReadString('\n')
	return strings.TrimSpace(input)
}

// AcceptHumanConnection blocks until one human client connects, then returns
// the connection.  The story mod is single-player, so we only need one.
func AcceptHumanConnection(ln net.Listener) net.Conn {
	fmt.Println("Waiting for a player to connect...")
	conn, err := ln.Accept()
	if err != nil {
		fmt.Printf("[Server] Accept error: %v\n", err)
		return AcceptHumanConnection(ln)
	}
	fmt.Printf("[Server] Player connected from %s\n", conn.RemoteAddr())
	return conn
}
