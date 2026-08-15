package main

import (
	"encoding/json"
	"fmt"
	"unsafe"

	"game/server/mods/shared"
)

func humanKeys() []string {
	return []string{"anar"}
}

// ---------------------------------------------------------------------------
// DirectedTree helpers
//
// The whole Python DirectedTree lives in GameState.Data["tree"] as a flat map
// with dot-notation keys:
//
//	"branches.<layer1>"   -> []string (children in order)
//	"selections.<layer1>" -> int     (advanced index, -1 = none)
//
// After a JSON round-trip, []string becomes []any and int becomes float64,
// so the getters below accept both shapes.
// ---------------------------------------------------------------------------

func treeData(state *shared.GameState) map[string]any {
	if t, ok := state.Data["tree"].(map[string]any); ok {
		return t
	}
	return nil
}

func getBranch(state *shared.GameState, layer1Key string) []string {
	t := treeData(state)
	if t == nil {
		return nil
	}

	switch val := t["branches."+layer1Key].(type) {
	case []string:
		return val
	case []any:
		out := make([]string, len(val))
		for i, v := range val {
			out[i] = fmt.Sprintf("%v", v)
		}
		return out
	}
	return nil
}

func getSelection(state *shared.GameState, layer1Key string) int {
	t := treeData(state)
	if t == nil {
		return -1
	}

	switch val := t["selections."+layer1Key].(type) {
	case int:
		return val
	case float64:
		return int(val)
	}
	return -1
}

func setSelection(state *shared.GameState, layer1Key string, idx int) {
	if t := treeData(state); t != nil {
		t["selections."+layer1Key] = idx
	}
}

func indexOfChild(children []string, childKey string) int {
	for i, child := range children {
		if child == childKey {
			return i
		}
	}
	return -1
}

// advanceTo matches Python DirectedTree.advance_to: never moves backwards.
func advanceTo(state *shared.GameState, layer1Key string, childKey string) {
	hasNotBeenAdvanced := isAdvanced(
		state,
		advNot(advIs(layer1Key, childKey)),
	)
	if hasNotBeenAdvanced {
		setSelection(state, layer1Key, indexOfChild(getBranch(state, layer1Key), childKey))
	}
}

// notDefeated mirrors PlayerNotDefeatedOption.check_condition.
func notDefeated(state *shared.GameState) bool {
	return !isAdvanced(state, advIs("main_quest_detail", "player_defeated"))
}

// ---------------------------------------------------------------------------
// Game setup
// ---------------------------------------------------------------------------

func InitState() *shared.GameState {
	return &shared.GameState{
		Data: map[string]any{
			"display_order": []string{
				"status",
				"acting_player",
				"player_order",
			},
			"status":        "RUNNING",
			"event_logs":    []string{},
			"acting_player": "anar",
			"player_order":  humanKeys(),
			"tree": map[string]any{
				"branches.main_quest":          []string{"gathering_allies", "confronting_villain", "victory"},
				"branches.main_quest_detail":   []string{"confronted_villain_without_artifact", "player_defeated"},
				"branches.side_quest":          []string{"rumor_heard", "clue_found", "artifact_recovered"},
				"selections.main_quest":        -1,
				"selections.main_quest_detail": -1,
				"selections.side_quest":        -1,
			},
		},
	}
}

func InitRobots() map[string]*shared.Player {
	// The Python story game has no robot players.
	return map[string]*shared.Player{}
}

func playerNotDefeated() AdvancedExpression {
	return advNot(advIs("main_quest_detail", "player_defeated"))
}

func InitDeck() map[string]*shared.Card {
	return map[string]*shared.Card{
		"skip": {
			Name:       "Skip",
			Conditions: func(state *shared.GameState) bool { return true },
			Action: func(state *shared.GameState) {
				shared.ClearEventLogs(state)
				shared.EndTurn(state)
				shared.LogEvent(state, "The hero pauses for a moment.")
			},
		},
		"recruit": {
			Name: "Recruit heroes to aid your cause",
			Conditions: func(state *shared.GameState) bool {
				return isAdvanced(
					state,
					advAnd(
						playerNotDefeated(),
						advNot(
							advIs("main_quest", "gathering_allies"),
						),
					),
				)
			},
			Action: func(state *shared.GameState) {
				shared.ClearEventLogs(state)
				advanceTo(state, "main_quest", "gathering_allies")
				shared.LogEvent(
					state,
					"Heroes rally to your banner. Your army grows stronger.",
				)
				shared.EndTurn(state)
			},
		},
		"confront": {
			Name: "Storm the villain's fortress",
			Conditions: func(state *shared.GameState) bool {
				return isAdvanced(
					state,
					advAnd(
						playerNotDefeated(),
						advOr(
							advAnd(
								advIs("main_quest", "gathering_allies"),
								advNot(
									advIs("main_quest", "confronting_villain"),
								),
							),
							advIs(
								"main_quest_detail",
								"confronted_villain_without_artifact",
							),
						),
					),
				)
			},
			Action: func(state *shared.GameState) {
				shared.ClearEventLogs(state)
				advanceTo(state, "main_quest", "confronting_villain")

				secondMeetingMsg := `The villain says: "This time, I will finish you for good".`

				if isAdvanced(
					state,
					advNot(
						advIs("side_quest", "artifact_recovered"),
					),
				) {
					if isAdvanced(
						state,
						advIs(
							"main_quest_detail",
							"confronted_villain_without_artifact",
						),
					) {
						msg := "You storm the fortress. " + secondMeetingMsg +
							" The fight is brutal and you lose. " +
							"The villain's sword pierces your heart."

						advanceTo(state, "main_quest_detail", "player_defeated")
						shared.LogEvent(state, msg)
						shared.StateSet(state, "status", "DEFEATED")
					} else {
						advanceTo(
							state,
							"main_quest_detail",
							"confronted_villain_without_artifact",
						)
						shared.LogEvent(
							state,
							"You storm the fortress. The fight is brutal and you lose. The villain lets you go, out of pity.",
						)
					}
				} else {
					victoryMsg := "The battle starts and with the help of your artifact, " +
						"you strike the villain down! The world enters the era of peace."

					var msg string

					if isAdvanced(
						state,
						advIs(
							"main_quest_detail",
							"confronted_villain_without_artifact",
						),
					) {
						msg = "With the artifact in hand, you storm the fortress. " +
							secondMeetingMsg + " " + victoryMsg
					} else {
						msg = "With the artifact in hand, you storm the fortress. " +
							victoryMsg
					}

					advanceTo(state, "main_quest", "victory")
					shared.LogEvent(state, msg)
					shared.StateSet(state, "status", "VICTORY")
				}

				shared.EndTurn(state)
			},
		},
		"rumor": {
			Name: "Listen to whispers at the tavern",
			Conditions: func(state *shared.GameState) bool {
				return isAdvanced(
					state,
					advAnd(
						playerNotDefeated(),
						advNot(
							advIs("side_quest", "rumor_heard"),
						),
					),
				)
			},
			Action: func(state *shared.GameState) {
				shared.ClearEventLogs(state)
				advanceTo(state, "side_quest", "rumor_heard")
				shared.LogEvent(
					state,
					"You overhear a tavern whisper about a lost artifact.",
				)
				shared.EndTurn(state)
			},
		},
		"clue": {
			Name: "Explore the ancient ruins for hidden lore",
			Conditions: func(state *shared.GameState) bool {
				return notDefeated(state) &&
					isAdvanced(
						state,
						advAnd(
							playerNotDefeated(),
							advIs("side_quest", "rumor_heard"),
							advNot(
								advIs("side_quest", "clue_found"),
							),
						),
					)
			},
			Action: func(state *shared.GameState) {
				shared.ClearEventLogs(state)
				advanceTo(state, "side_quest", "clue_found")
				shared.LogEvent(
					state,
					"Hidden among the ruins, you find an ancient map leading to the artifact.",
				)
				shared.EndTurn(state)
			},
		},
		"artifact": {
			Name: "Claim the legendary artifact as your own",
			Conditions: func(state *shared.GameState) bool {
				return isAdvanced(
					state,
					advAnd(
						playerNotDefeated(),
						advIs("side_quest", "clue_found"),
						advIs("main_quest", "gathering_allies"),
						advNot(
							advIs("side_quest", "artifact_recovered"),
						),
					),
				)
			},
			Action: func(state *shared.GameState) {
				shared.ClearEventLogs(state)
				advanceTo(state, "side_quest", "artifact_recovered")
				shared.LogEvent(
					state,
					"You claim the legendary artifact. Its power hums in your hands.",
				)
				shared.EndTurn(state)
			},
		},
	}
}

// ---------------------------------------------------------------------------
// WASM exports
// ---------------------------------------------------------------------------

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

//go:wasmexport Return2
func Return2() int32 {
	return 2
}

func main() {}
