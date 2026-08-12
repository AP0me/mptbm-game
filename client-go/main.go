package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"strings"
)

type Packet struct {
	Type string `json:"type"`
	Data any    `json:"data"`
}

func ConnectToServer() net.Conn {
	reader := bufio.NewReader(os.Stdin)
	fmt.Println("--- Server Connection ---")

	fmt.Print("Address: (e.g: 127.0.0.1) ")
	address, _ := reader.ReadString('\n')
	address = strings.TrimSpace(address)
	if address == "" {
		address = "127.0.0.1"
	}

	fmt.Print("Port: (e.g: 8080) ")
	port, _ := reader.ReadString('\n')
	port = strings.TrimSpace(port)
	if port == "" {
		port = "8080"
	}

	target := net.JoinHostPort(address, port)
	fmt.Printf("Attempting to connect to %s...\n", target)

	conn, err := net.Dial("tcp", target)
	if err != nil {
		PrintError("Connection refused: " + err.Error())
		return ConnectToServer()
	}

	fmt.Printf("%sConnected successfully!%s\n", Green, Reset)
	fmt.Printf("%sWelcome to the Story Game!%s\n", Yellow, Reset)
	return conn
}

func main() {
	conn := ConnectToServer()
	defer conn.Close()

	scanner := bufio.NewScanner(conn)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	stdinReader := bufio.NewReader(os.Stdin)

	for scanner.Scan() {
		line := scanner.Text()
		var packet Packet
		if err := json.Unmarshal([]byte(line), &packet); err != nil {
			continue
		}

		switch packet.Type {
		case "STATE":
			if data, ok := packet.Data.(map[string]any); ok {
				PrintTree(data) // ← was PrintState
			}
		case "CARDS":
			if data, ok := packet.Data.(map[string]any); ok {
				PrintCards(data)
			}
		case "CHOICE":
			fmt.Printf("%s%s➤ Enter option key (e.g. begin, recruit, confront): %s",
				Yellow, Bold, Reset)
			choice, _ := stdinReader.ReadString('\n')
			conn.Write([]byte(choice))
		case "MESSAGE":
			if data, ok := packet.Data.([]any); ok {
				PrintCardResponse(data)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		PrintError("Connection lost: " + err.Error())
	}
}
