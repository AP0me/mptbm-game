package main

import (
	"fmt"
	"game/server/core"
	"game/server/mods/vanilla"
	"net"
)

func main() {
	ln, _ := net.Listen("tcp", ":8080")
	fmt.Println("Server started on :8080")

	activeMod := vanilla.VanillaMod{}
	
	state := activeMod.InitState()
    deck := activeMod.InitDeck(state)
    players := activeMod.InitRobots()
	core.WelcomeHumansToPlayerList(state.Data["player_order"].([]string), ln, players)
	conns := core.PlayerConnections(players)

	for state.Data["status"] == "RUNNING" {
		acting_name := activeMod.GetActingPlayer(state)
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
