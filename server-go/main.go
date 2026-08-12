package main

import (
	"context"
	"fmt"
	"game/server/core"
	"game/server/mods/shared"
	"net"
)

func main() {
	ctx := context.Background()
	runtime, instance, err := core.LoadWasmInstance(ctx, "mods/vanilla/vanilla.wasm")
	if err != nil {
		fmt.Printf("Failed to load wasm instance: %v\n", err)
		return
	}
	defer runtime.Close(ctx)

	ln, _ := net.Listen("tcp", ":8080")
	fmt.Println("Server started on :8080")

	state := core.WasmInitState(instance, ctx)
	robots := core.WasmInitRobotsNames(instance, ctx)

	players := map[string]*shared.Player{}
	core.WelcomeHumansToPlayerList(shared.GetStringList(&state, "player_order"), ln, players, robots)
	conns := core.PlayerConnections(players)

	for state.Data["status"] == "RUNNING" {
		acting_name := shared.GetActingPlayer(&state)
		player := players[acting_name]
		core.SendState(state, conns)

		playable_cards, _ := core.CallPlayableCards(ctx, instance, state)

		if player.Conn != nil {
			core.SendCards(playable_cards, player.Conn)
		}

		card_key := "skip"
		if _, ok := robots[acting_name]; ok {
			card_key, err = core.CallChooseCard(ctx, instance, acting_name, state)
		} else {
			card_key = core.HumanInput(player.Conn)
		}

		newState, err := core.CallPlayCardAction(ctx, instance, card_key, state)
		if err != nil {
			fmt.Printf("Error playing card action: %v\n", err)
			continue
		}
		state = newState

		core.SendMessages(shared.GetStringList(&state, "event_logs"), conns)
	}
}
