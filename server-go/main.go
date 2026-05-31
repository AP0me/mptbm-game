package main

import (
	"context"
	"encoding/json"
	"fmt"
	"game/server/core"
	"game/server/mods/shared"
	"net"
	"os"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
)

func LoadWasmInstance(ctx context.Context, path string) (wazero.Runtime, api.Module, error) {
	wasmBytes, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, err
	}

	r := wazero.NewRuntime(ctx)
	wasi_snapshot_preview1.MustInstantiate(ctx, r)
	config := wazero.NewModuleConfig().WithStartFunctions("_initialize")
	mod, err := r.InstantiateWithConfig(ctx, wasmBytes, config)
	if err != nil {
		r.Close(ctx)
		return nil, nil, err
	}

	return r, mod, nil
}

func WasmInitState(instance api.Module, ctx context.Context) shared.GameState {
	var state shared.GameState

	InitState := instance.ExportedFunction("InitState")
	if InitState == nil {
		fmt.Println("Exported function 'InitState' not found")
		return state
	}

	results, err := InitState.Call(ctx)
	if err != nil {
		fmt.Printf("Error calling InitState: %v\n", err)
		return state
	}

	if len(results) > 0 {
		packedResult := results[0]
		ptr := uint32(packedResult >> 32)
		size := uint32(packedResult)
		bytes, ok := instance.Memory().Read(ptr, size)
		if !ok {
			fmt.Printf("Failed to read WASM memory at pointer %d with size %d\n", ptr, size)
			return state
		}
		if err := json.Unmarshal(bytes, &state); err != nil {
			fmt.Printf("Failed to parse JSON string: %v\n", err)
			return state
		}
	} else {
		fmt.Println("Function executed successfully (no return values)")
	}

	return state
}

func WasmInitRobotsNames(instance api.Module, ctx context.Context) map[string]shared.HostPlayer {
	var robotMap map[string]shared.HostPlayer

	InitRobotsNames := instance.ExportedFunction("InitRobotsNames")
	if InitRobotsNames == nil {
		fmt.Println("Exported function 'InitRobotsNames' not found")
		return robotMap
	}

	res, err := InitRobotsNames.Call(ctx)
	if err != nil {
		fmt.Printf("Error calling InitRobotsNames: %v\n", err)
		return robotMap
	}

	if len(res) > 0 {
		// 1. Unpack the pointer and size
		packedResult := res[0]
		ptr := uint32(packedResult >> 32)
		size := uint32(packedResult)

		// 2. Read the raw bytes from WASM sandbox memory
		bytes, ok := instance.Memory().Read(ptr, size)
		if !ok {
			fmt.Println("Failed to read WASM memory boundary")
			return robotMap
		}

		// 3. Parse the JSON back into a Go map
		if err := json.Unmarshal(bytes, &robotMap); err != nil {
			fmt.Printf("Failed to unmarshal robot names: %v\n", err)
			return robotMap
		}
	}

	return robotMap
}

func main() {
	ctx := context.Background()
	runtime, instance, err := LoadWasmInstance(ctx, "mods/vanilla/vanilla.wasm")
	if err != nil {
		fmt.Printf("Failed to load wasm instance: %v\n", err)
		return
	}
	defer runtime.Close(ctx)

	ln, _ := net.Listen("tcp", ":8080")
	fmt.Println("Server started on :8080")

	state := WasmInitState(instance, ctx)
	robots := WasmInitRobotsNames(instance, ctx)
	// deck := initDeck()

	players := map[string]*shared.Player{}
	core.WelcomeHumansToPlayerList(shared.GetStringList(&state, "player_order"), ln, players, robots)
	conns := core.PlayerConnections(players)

	for state.Data["status"] == "RUNNING" {
		acting_name := shared.GetActingPlayer(&state)
		// player := players[acting_name]
		_ = players[acting_name]
		core.SendState(state, conns)

		// playable_cards := core.PlayableCards(deck, state) // TODO

		// if player.Conn != nil {
		// 	core.SendCards(playable_cards, player.Conn)
		// }

		// card := core.ChooseCard(player, playable_cards, state)
		// card.Action(&state) // TODO

		core.SendMessages(shared.GetStringList(&state, "event_logs"), conns)
		state.Data["event_logs"] = []string{}
	}
}
