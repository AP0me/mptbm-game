package main

import (
	"fmt"
	"net"
)

func main() {
	ln, _ := net.Listen("tcp", ":8080")
	fmt.Println("Server started on :8080")

	state := InitState()
	players := InitRobots()
	HumanPlayers(state, ln, players)
	deck := InitDeck(state)

	conns := []net.Conn{}
	for _, p := range players {
		if p.Conn != nil {
			conns = append(conns, p.Conn)
		}
	}

	for state.Data["status"] == "RUNNING" {
		acting_name := state.GetActingPlayer()
		player := players[acting_name]
		SendState(state, conns)

		playable_cards := PlayableCards(deck, state)

		if player.Conn != nil {
			SendCards(playable_cards, player.Conn)
		}

		card := ChooseCard(player, playable_cards, state)
		card.Action(state)

		SendMessages(state.Data["event_logs"].([]string), conns)
		state.Data["event_logs"] = []string{}
	}
}
