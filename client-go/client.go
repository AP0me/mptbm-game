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
	Type string                 `json:"type"`
	Data interface{}            `json:"data"`
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

	target := fmt.Sprintf("%s:%s", address, port)
	fmt.Printf("Attempting to connect to %s...\n", target)

	conn, err := net.Dial("tcp", target)
	if err != nil {
		PrintError("Connection refused: " + err.Error())
		return ConnectToServer()
	}

	fmt.Printf("%sConnected successfully!%s\n", Green, Reset)
	fmt.Printf("%sWelcome to the Game!%s\n", Yellow, Reset)
	fmt.Println("Waiting for the other players...")

	return conn
}

func main() {
	conn := ConnectToServer()
	defer conn.Close()

	scanner := bufio.NewScanner(conn)
	// Use a large buffer for game states
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	stdinReader := bufio.NewReader(os.Stdin)

	for scanner.Scan() {
		line := scanner.Text()
		var packet Packet
		err := json.Unmarshal([]byte(line), &packet)
		if err != nil {
			continue
		}

		switch packet.Type {
		case "STATE":
			if data, ok := packet.Data.(map[string]interface{}); ok {
				PrintState(data)
			}
		case "CARDS":
			if data, ok := packet.Data.(map[string]interface{}); ok {
				PrintCards(data)
			}
		case "CHOICE":
			fmt.Printf("%s%s➤ Enter card key (e.g. skip, hunt): %s", Yellow, Bold, Reset)
			choice, _ := stdinReader.ReadString('\n')
			conn.Write([]byte(choice))
		case "MESSAGE":
			if data, ok := packet.Data.([]interface{}); ok {
				PrintCardResponse(data)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		PrintError("Connection lost to server: " + err.Error())
		main() // Reconnect
	}
}