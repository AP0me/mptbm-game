package core

import (
	"encoding/json"
	"fmt"
	"game/server/mods/shared"
	"net"
)

type Packet struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

func SafeSend(conn net.Conn, data interface{}, pType string) bool {
	packet := Packet{Type: pType, Data: data}
	encoded, err := json.Marshal(packet)
	if err != nil {
		return false
	}

	_, err = conn.Write(append(encoded, '\n'))
	if err != nil {
		fmt.Printf("[Server] Write error: %v\n", err)
		return false
	}
	return true
}

func PacketMultiSend(conns []net.Conn, data interface{}, pType string) {
	for _, conn := range conns {
		SafeSend(conn, data, pType)
	}
}

func SendState(state shared.GameState, conns []net.Conn) {
	PacketMultiSend(conns, state.Data, "STATE")
}

func SendCards(playableCards map[string]*shared.Card, conn net.Conn) {
	cardNames := make(map[string]map[string]string)
	for key, card := range playableCards {
		cardNames[key] = map[string]string{"name": card.Name}
	}
	SafeSend(conn, cardNames, "CARDS")
}

func SendMessages(logs []string, conns []net.Conn) {
	PacketMultiSend(conns, logs, "MESSAGE")
}