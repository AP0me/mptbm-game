package main

import (
	"fmt"
	"game/server/core"
	"game/server/mods/shared"
	"log"
	"net"
	"plugin"
)

func main() {
	p, err := plugin.Open("mods/vanilla/vanilla.so")
	if err != nil {
		log.Fatalf("Error opening plugin: %v", err)
	}
	
	symInitState, err := p.Lookup("InitState")
	if err != nil {
		log.Fatalf("Lookup InitState failed: %v", err)
	}
	initStateDLL := symInitState.(func() *shared.GameState)

	symInitDeck, err := p.Lookup("InitDeck")
	if err != nil {
		log.Fatalf("Lookup InitDeck failed: %v", err)
	}
	initDeckDLL := symInitDeck.(func(*shared.GameState) map[string]*shared.Card)

	symInitRobots, err := p.Lookup("InitRobots")
	if err != nil {
		log.Fatalf("Lookup InitRobots failed: %v", err)
	}
	initRobotsDLL := symInitRobots.(func() map[string]*shared.Player)

	ln, _ := net.Listen("tcp", ":8080")
	fmt.Println("Server started on :8080")

	state := initStateDLL()
	deck := initDeckDLL(state)
	players := initRobotsDLL()

	core.WelcomeHumansToPlayerList(state.Data["player_order"].([]string), ln, players)
	conns := core.PlayerConnections(players)

	for state.Data["status"] == "RUNNING" {
		acting_name := shared.GetActingPlayer(state)
		player := players[acting_name]
		core.SendState(state, conns)

		playable_cards := core.PlayableCards(deck, state)

		if player.Conn != nil {
			core.SendCards(playable_cards, player.Conn)
		}

		card := core.ChooseCard(player, playable_cards, state)
		card.Action(state)

		core.SendMessages(state.Data["event_logs"].([]string), conns)
		state.Data["event_logs"] = []string{}
	}
}
