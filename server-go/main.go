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

func writeToWasm(ctx context.Context, instance api.Module, allocator api.Function, data any) (uint32, uint32, error) {
	bytes, err := json.Marshal(data)
	if err != nil {
		return 0, 0, err
	}
	size := uint32(len(bytes))

	// Ask WASM to allocate memory for this payload
	res, err := allocator.Call(ctx, uint64(size))
	if err != nil {
		return 0, 0, err
	}
	ptr := uint32(res[0])

	// Write the JSON bytes into that allocated sandbox slot
	instance.Memory().Write(ptr, bytes)
	return ptr, size, nil
}

func CallPlayableCards(ctx context.Context, instance api.Module, hostState any) (map[string]shared.CardProfile, error) {
	allocator := instance.ExportedFunction("Allocate")
	playableCardsFn := instance.ExportedFunction("PlayableCards")

	ptr, size, err := writeToWasm(ctx, instance, allocator, hostState)
	if err != nil {
		return nil, err
	}

	res, err := playableCardsFn.Call(ctx, uint64(ptr), uint64(size))
	if err != nil {
		return nil, err
	}

	outPtr := uint32(res[0] >> 32)
	outSize := uint32(res[0])
	outBytes, _ := instance.Memory().Read(outPtr, outSize)

	// Unmarshal directly into the correct type
	var playable map[string]shared.CardProfile
	if err := json.Unmarshal(outBytes, &playable); err != nil {
		return nil, err
	}

	return playable, nil
}

func CallChooseCard(ctx context.Context, instance api.Module, playerName string, hostState any) (string, error) {
    allocator := instance.ExportedFunction("Allocate")
    chooseCardFn := instance.ExportedFunction("ChooseCard")

    // 1. Write the player NAME string and state into the memory heap
    pPtr, pSize, _ := writeToWasm(ctx, instance, allocator, playerName)
    sPtr, sSize, _ := writeToWasm(ctx, instance, allocator, hostState)

    // 2. Invoke the choice calculation function
    res, err := chooseCardFn.Call(ctx, uint64(pPtr), uint64(pSize), uint64(sPtr), uint64(sSize))
    if err != nil {
        return "", err
    }

    // 3. Read back the single chosen card key
    outPtr := uint32(res[0] >> 32)
    outSize := uint32(res[0])
    outBytes, _ := instance.Memory().Read(outPtr, outSize)

    // Convert raw bytes directly to string—no JSON parsing required!
    return string(outBytes), nil
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

	players := map[string]*shared.Player{}
	core.WelcomeHumansToPlayerList(shared.GetStringList(&state, "player_order"), ln, players, robots)
	conns := core.PlayerConnections(players)

	for state.Data["status"] == "RUNNING" {
		acting_name := shared.GetActingPlayer(&state)
		player := players[acting_name]
		core.SendState(state, conns)

		playable_cards, _ := CallPlayableCards(ctx, instance, state)

		if player.Conn != nil {
			core.SendCards(playable_cards, player.Conn)
		}

		card_key := "skip"
		if player.Decide == nil {
			card_key = core.HumanInput(player.Conn)
		} else {
			card_key, _ = CallChooseCard(ctx, instance, acting_name, state)
		}
		fmt.Println(acting_name, card_key)
		// card.Action(&state) // TODO

		core.SendMessages(shared.GetStringList(&state, "event_logs"), conns)
		state.Data["event_logs"] = []string{}
	}
}
