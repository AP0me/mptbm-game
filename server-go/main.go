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
	WelcomeHumansToPlayerList(state.Data["player_order"].([]string), ln, players)
	deck := InitDeck(state)
	conns := PlayerConnections(players)

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
