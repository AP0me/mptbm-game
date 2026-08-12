package core

import (
	"context"
	"encoding/json"
	"fmt"
	"game/server/mods/shared"
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

// ---------------------------------------------------------------------------
// Init state
// ---------------------------------------------------------------------------

func WasmInitState(instance api.Module, ctx context.Context) shared.GameState {
	var state shared.GameState

	fn := instance.ExportedFunction("InitMarshaledState")
	if fn == nil {
		fmt.Println("Exported function 'InitMarshaledState' not found")
		return state
	}

	results, err := fn.Call(ctx)
	if err != nil {
		fmt.Printf("Error calling InitMarshaledState: %v\n", err)
		return state
	}

	if len(results) == 0 {
		return state
	}

	packed := results[0]
	ptr := uint32(packed >> 32)
	size := uint32(packed)
	bytes, ok := instance.Memory().Read(ptr, size)
	if !ok {
		fmt.Printf("Failed to read WASM memory at ptr=%d size=%d\n", ptr, size)
		return state
	}
	if err := json.Unmarshal(bytes, &state); err != nil {
		fmt.Printf("Failed to unmarshal state: %v\n", err)
		return state
	}
	return state
}

// ---------------------------------------------------------------------------
// Init robot names (empty for story mod, kept for compatibility)
// ---------------------------------------------------------------------------

func WasmInitRobotsNames(instance api.Module, ctx context.Context) map[string]shared.HostPlayer {
	var robotMap map[string]shared.HostPlayer

	fn := instance.ExportedFunction("InitRobotsNames")
	if fn == nil {
		return robotMap
	}

	res, err := fn.Call(ctx)
	if err != nil {
		fmt.Printf("Error calling InitRobotsNames: %v\n", err)
		return robotMap
	}
	if len(res) == 0 {
		return robotMap
	}

	packed := res[0]
	ptr := uint32(packed >> 32)
	size := uint32(packed)
	bytes, ok := instance.Memory().Read(ptr, size)
	if !ok {
		return robotMap
	}
	if err := json.Unmarshal(bytes, &robotMap); err != nil {
		fmt.Printf("Failed to unmarshal robot names: %v\n", err)
	}
	return robotMap
}

// ---------------------------------------------------------------------------
// Helper: write a Go value into WASM memory via Allocate
// ---------------------------------------------------------------------------

func writeToWasm(ctx context.Context, instance api.Module, data any) (uint32, uint32, error) {
	allocator := instance.ExportedFunction("Allocate")
	if allocator == nil {
		return 0, 0, fmt.Errorf("Allocate export not found")
	}

	bytes, err := json.Marshal(data)
	if err != nil {
		return 0, 0, err
	}
	size := uint32(len(bytes))

	res, err := allocator.Call(ctx, uint64(size))
	if err != nil {
		return 0, 0, err
	}
	ptr := uint32(res[0])

	instance.Memory().Write(ptr, bytes)
	return ptr, size, nil
}

// ---------------------------------------------------------------------------
// PlayableCards — ask WASM which cards are playable for the current state
// ---------------------------------------------------------------------------

func CallPlayableCards(ctx context.Context, instance api.Module, state shared.GameState) (map[string]shared.CardProfile, error) {
	fn := instance.ExportedFunction("PlayableCards")
	if fn == nil {
		return nil, fmt.Errorf("PlayableCards export not found")
	}

	ptr, size, err := writeToWasm(ctx, instance, state)
	if err != nil {
		return nil, err
	}

	res, err := fn.Call(ctx, uint64(ptr), uint64(size))
	if err != nil {
		return nil, err
	}

	outPtr := uint32(res[0] >> 32)
	outSize := uint32(res[0])
	outBytes, _ := instance.Memory().Read(outPtr, outSize)

	var playable map[string]shared.CardProfile
	if err := json.Unmarshal(outBytes, &playable); err != nil {
		return nil, err
	}
	return playable, nil
}

// ---------------------------------------------------------------------------
// PlayCardAction — execute a card action, return the updated state
// ---------------------------------------------------------------------------

func CallPlayCardAction(ctx context.Context, instance api.Module, cardKey string, state shared.GameState) (shared.GameState, error) {
	fn := instance.ExportedFunction("PlayCardAction")
	if fn == nil {
		return shared.GameState{}, fmt.Errorf("PlayCardAction export not found")
	}

	kPtr, kSize, err := writeToWasm(ctx, instance, cardKey)
	if err != nil {
		return shared.GameState{}, err
	}
	sPtr, sSize, err := writeToWasm(ctx, instance, state)
	if err != nil {
		return shared.GameState{}, err
	}

	res, err := fn.Call(ctx, uint64(kPtr), uint64(kSize), uint64(sPtr), uint64(sSize))
	if err != nil {
		return shared.GameState{}, err
	}

	outPtr := uint32(res[0] >> 32)
	outSize := uint32(res[0])
	outBytes, ok := instance.Memory().Read(outPtr, outSize)
	if !ok {
		return shared.GameState{}, fmt.Errorf("failed to read updated state memory")
	}

	var updated shared.GameState
	if err := json.Unmarshal(outBytes, &updated); err != nil {
		return shared.GameState{}, err
	}
	return updated, nil
}
