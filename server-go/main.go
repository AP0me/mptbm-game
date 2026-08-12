package main

import (
	"context"
	"fmt"
	"game/server/core"
	"game/server/mods/shared"
	"net"
)

func isGameOver(s *shared.GameState) bool {
	return s.Tree.IsAdvanced("main_quest", "victory") ||
		s.Tree.IsAdvanced("main_quest_detail", "player_defeated")
}

func main() {
	ctx := context.Background()

	// Load the story WASM module
	runtime, instance, err := core.LoadWasmInstance(ctx, "mods/vanilla/story.wasm")
	if err != nil {
		fmt.Printf("Failed to load wasm: %v\n", err)
		return
	}
	defer runtime.Close(ctx)

	// Initialize state
	state := core.WasmInitState(instance, ctx)

	// Start TCP server
	ln, err := net.Listen("tcp", ":8080")
	if err != nil {
		fmt.Printf("Failed to listen: %v\n", err)
		return
	}
	fmt.Println("Story server started on :8080")

	// Accept one human player
	conn := core.AcceptHumanConnection(ln)
	defer conn.Close()

	conns := []net.Conn{conn}

	// Main game loop
	for !isGameOver(&state) {
		// 1. Send current tree state
		core.SendState(state, conns)

		// 2. Get playable options from WASM
		playable, err := core.CallPlayableCards(ctx, instance, state)
		if err != nil {
			fmt.Printf("Error getting playable cards: %v\n", err)
			continue
		}

		// 3. Send options to client
		core.SendCards(playable, conn)

		// 4. Get human's choice
		cardKey := core.HumanInput(conn)
		if cardKey == "" {
			cardKey = "skip"
		}

		// 5. Execute the chosen action in WASM
		newState, err := core.CallPlayCardAction(ctx, instance, cardKey, state)
		if err != nil {
			fmt.Printf("Error playing card action: %v\n", err)
			continue
		}
		state = newState

		// 6. Send event messages
		core.SendMessages(state.EventLogs, conns)
	}

	// Final state + messages
	core.SendState(state, conns)
	core.SendMessages(state.EventLogs, conns)

	fmt.Println("Game over!")
}
