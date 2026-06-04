# Multi-Player Text-Based Moddable Game

## What is this game?
idk ... it is meant to be a story generator.

## How to run?
- Install [golang](https://go.dev/doc/install).
- Open 3 terminal windows.
- Navigate to project root.
- Open server-go in terminal 1.
- Open client-go in terminal 2.
- Open client-go in terminal 3.
- Run `bash run.sh` on terminal 1.
- Run `go run .` on terminal 2.
- Run `go run .` on terminal 3.
- Play the game `:)`

## How to play?
- Open the terminal 2 and terminal 3 side by side. They are meant to be played by different players.
- Type in one of the displayed card options and press `Enter`.
- Observe the message describing the changes to the game world.
- You can do anything you want, but if you don't know what to do start by surviving a few days and gather wood.

## How to make mods?
- Open a new folder under `server-go/mods` folder.
- Create a `main.go` file under this folder.
- Copy the code below to the `main.go`:
```golang
package main

import (
	"encoding/json"
	"game/server/mods/shared"
	"unsafe"
)

//go:wasmexport InitMarshaledState
func InitMarshaledState() uint64 {
	buf, err := json.Marshal(InitState())
	if err != nil {
		return 0
	}

	ptr := uint32(uintptr(slicePtr(buf)))
	size := uint32(len(buf))
	return (uint64(ptr) << 32) | uint64(size)
}

func slicePtr(b []byte) unsafe.Pointer {
	if len(b) == 0 {
		return nil
	}
	return unsafe.Pointer(&b[0])
}

//go:wasmexport InitRobotsNames
func InitRobotsNames() uint64 {
	return shared.InitRobotsNames(InitRobots())
}

//go:wasmexport PlayableCards
func PlayableCards(statePtr uint32, stateSize uint32) uint64 {
	return shared.PlayableCards(InitDeck(), statePtr, stateSize)
}

//go:wasmexport ChooseCard
func ChooseCard(namePtr uint32, nameSize uint32, statePtr uint32, stateSize uint32) uint64 {
	return shared.ChooseCard(InitRobots(), InitDeck(), namePtr, nameSize, statePtr, stateSize)
}

//go:wasmexport PlayCardAction
func PlayCardAction(keyPtr uint32, keySize uint32, statePtr uint32, stateSize uint32) uint64 {
	return shared.PlayCardAction(InitDeck(), keyPtr, keySize, statePtr, stateSize)
}

func main() {}
``` 
- Implement InitRobots, InitDeck and InitState functions.

- Minimal implementation examples:
```golang
func InitState() *shared.GameState {
	state := &shared.GameState{
		Data: map[string]any{
			"display_order": []string{
				"status",
				"player_order",
				"acting_player",
			},
			"status":        "RUNNING",
			"event_logs":    []string{},
			"player_order":  []string{"steve"},
			"acting_player": "steve",
		},
	}

	return state
}

func InitRobots() map[string]*shared.Player {
	return map[string]*shared.Player{}
}

func StateInvisibleSet(s *shared.GameState, key string, val any) {
	if key == "" { return }

	if val == nil {
		delete(s.Data, key)
		return
	}

	s.Data[key] = val
}

func StateSet(s *shared.GameState, key string, val any) {
	display_order := shared.GetStringList(s, "display_order")
	if !slices.Contains(display_order, key) {
		s.Data["display_order"] = append(display_order, key)
	}
	StateInvisibleSet(s, key, val)
}

func ClearEventLogs(s *shared.GameState) {
	StateInvisibleSet(s, "event_logs", []string{})
}

func EndTurn(s *shared.GameState) {
	order := shared.GetStringList(s, "player_order")
	current := shared.GetActingPlayer(s)
	nextIdx := 0
	for i, name := range order {
		if name == current {
			nextIdx = (i + 1) % len(order)
			break
		}
	}
	StateSet(s, "acting_player", order[nextIdx])
}

func LogEvent(s *shared.GameState, msg string) {
	logs := shared.GetStringList(s, "event_logs")
	StateInvisibleSet(s, "event_logs", append(logs, msg))
}

func InitDeck() map[string]*shared.Card {
	return map[string]*shared.Card{
		"skip": {
			Name:       "Skip",
			Conditions: func(state *shared.GameState) bool { return true },
			Action: func(state *shared.GameState) {
				ClearEventLogs(state)
				EndTurn(state)
				LogEvent(state, "Let's see what else is happening.")
			},
		},
    }
}
```

- Run `GOOS=wasip1 GOARCH=wasm go build -buildmode=c-shared -o my_game_mod.wasm main.go`.
- Resulting `.wasm` file is your mod. 
