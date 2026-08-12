package main

import (
	"encoding/json"
	"game/server/mods/shared"
	"unsafe"
)

// ---------------------------------------------------------------------------
// Branches & initial state — mirrors the Python toy example exactly.
// ---------------------------------------------------------------------------

func InitState() *shared.GameState {
	tree := shared.NewDirectedTree(
		map[string][]string{
			"main_quest":        {"inception", "gathering_allies", "confronting_villain", "victory"},
			"main_quest_detail": {"confronted_villain_without_artifact", "player_defeated"},
			"side_quest":        {"rumor_heard", "clue_found", "artifact_recovered"},
		},
		map[string]int{
			"main_quest":        -1,
			"main_quest_detail": -1,
			"side_quest":        -1,
		},
		[]string{"main_quest", "main_quest_detail", "side_quest"},
	)

	return &shared.GameState{
		Tree:      tree,
		EventLogs: []string{},
	}
}

// ---------------------------------------------------------------------------
// Deck — six options matching the Python Options() function.
// Each uses playerNotDefeatedCard to wrap its condition with the
// "not player_defeated" guard.
// ---------------------------------------------------------------------------

func playerNotDefeatedCard(label string, cond func(*shared.GameState) bool, act func(*shared.GameState)) *shared.Card {
	return &shared.Card{
		Name: label,
		Conditions: func(s *shared.GameState) bool {
			if s.Tree.IsAdvanced("main_quest_detail", "player_defeated") {
				return false
			}
			return cond(s)
		},
		Action: act,
	}
}

func InitDeck() map[string]*shared.Card {
	// --- Begin the adventure ---
	begin := playerNotDefeatedCard(
		"Begin the adventure",
		func(s *shared.GameState) bool {
			return !s.Tree.IsAdvanced("main_quest", "inception")
		},
		func(s *shared.GameState) {
			shared.ClearEventLogs(s)
			s.Tree.AdvanceTo("main_quest", "inception")
			shared.LogEvent(s, "You set out on your adventure. The road ahead is uncertain.")
		},
	)

	// --- Recruit heroes ---
	recruit := playerNotDefeatedCard(
		"Recruit heroes to aid your cause",
		func(s *shared.GameState) bool {
			return s.Tree.IsAdvanced("main_quest", "inception") &&
				!s.Tree.IsAdvanced("main_quest", "gathering_allies")
		},
		func(s *shared.GameState) {
			shared.ClearEventLogs(s)
			s.Tree.AdvanceTo("main_quest", "gathering_allies")
			shared.LogEvent(s, "Heroes rally to your banner. Your army grows stronger.")
		},
	)

	// --- Storm the villain's fortress (complex logic) ---
	confront := playerNotDefeatedCard(
		"Storm the villain's fortress",
		func(s *shared.GameState) bool {
			return (s.Tree.IsAdvanced("main_quest", "gathering_allies") &&
				!s.Tree.IsAdvanced("main_quest", "confronting_villain")) ||
				s.Tree.IsAdvanced("main_quest_detail", "confronted_villain_without_artifact")
		},
		func(s *shared.GameState) {
			shared.ClearEventLogs(s)
			s.Tree.AdvanceTo("main_quest", "confronting_villain")

			secondMeetingMsg := `The villain says: "This time, I will finish you for good".`

			if !s.Tree.IsAdvanced("side_quest", "artifact_recovered") {
				// No artifact — bad outcome
				if s.Tree.IsAdvanced("main_quest_detail", "confronted_villain_without_artifact") {
					// Second meeting without artifact — death
					msg := "You storm the fortress. " + secondMeetingMsg +
						" The fight is brutal and you lose. The villain's sword pierces your heart."
					s.Tree.AdvanceTo("main_quest_detail", "player_defeated")
					shared.LogEvent(s, msg)
				} else {
					// First meeting without artifact — spared
					s.Tree.AdvanceTo("main_quest_detail", "confronted_villain_without_artifact")
					msg := "You storm the fortress. The fight is brutal and you lose. " +
						"The villain lets you go, out of pity."
					shared.LogEvent(s, msg)
				}
			} else {
				// Artifact recovered — victory!
				victoryMsg := "The battle starts and with the help of your artifact, " +
					"you strike the villain down! The world enters the era of peace."
				if s.Tree.IsAdvanced("main_quest_detail", "confronted_villain_without_artifact") {
					msg := "With the artifact in hand, you storm the fortress. " +
						secondMeetingMsg + " " + victoryMsg
					shared.LogEvent(s, msg)
				} else {
					msg := "With the artifact in hand, you storm the fortress. " + victoryMsg
					shared.LogEvent(s, msg)
				}
				s.Tree.AdvanceTo("main_quest", "victory")
			}
		},
	)

	// --- Listen to whispers (side quest) ---
	rumor := playerNotDefeatedCard(
		"Listen to whispers at the tavern",
		func(s *shared.GameState) bool {
			return !s.Tree.IsAdvanced("side_quest", "rumor_heard")
		},
		func(s *shared.GameState) {
			shared.ClearEventLogs(s)
			s.Tree.AdvanceTo("side_quest", "rumor_heard")
			shared.LogEvent(s, "You overhear a tavern whisper about a lost artifact.")
		},
	)

	// --- Explore ancient ruins (side quest) ---
	clue := playerNotDefeatedCard(
		"Explore the ancient ruins for hidden lore",
		func(s *shared.GameState) bool {
			return s.Tree.IsAdvanced("side_quest", "rumor_heard") &&
				!s.Tree.IsAdvanced("side_quest", "clue_found")
		},
		func(s *shared.GameState) {
			shared.ClearEventLogs(s)
			s.Tree.AdvanceTo("side_quest", "clue_found")
			shared.LogEvent(s, "Hidden among the ruins, you find an ancient map leading to the artifact.")
		},
	)

	// --- Claim the artifact (side quest) ---
	artifact := playerNotDefeatedCard(
		"Claim the legendary artifact as your own",
		func(s *shared.GameState) bool {
			return s.Tree.IsAdvanced("side_quest", "clue_found") &&
				s.Tree.IsAdvanced("main_quest", "gathering_allies") &&
				!s.Tree.IsAdvanced("side_quest", "artifact_recovered")
		},
		func(s *shared.GameState) {
			shared.ClearEventLogs(s)
			s.Tree.AdvanceTo("side_quest", "artifact_recovered")
			shared.LogEvent(s, "You claim the legendary artifact. Its power hums in your hands.")
		},
	)

	// --- Skip (always available) ---
	skip := &shared.Card{
		Name:       "Skip",
		Conditions: func(s *shared.GameState) bool { return true },
		Action: func(s *shared.GameState) {
			shared.ClearEventLogs(s)
			shared.LogEvent(s, "You hesitate, unsure of what to do next...")
		},
	}

	return map[string]*shared.Card{
		"begin":    begin,
		"recruit":  recruit,
		"confront": confront,
		"rumor":    rumor,
		"clue":     clue,
		"artifact": artifact,
		"skip":     skip,
	}
}

// ---------------------------------------------------------------------------
// Robots — empty for the story mod (single-player).
// ---------------------------------------------------------------------------

func InitRobots() map[string]*shared.Player {
	return map[string]*shared.Player{}
}

// ---------------------------------------------------------------------------
// WASM exports
// ---------------------------------------------------------------------------

func slicePtr(b []byte) unsafe.Pointer {
	if len(b) == 0 {
		return nil
	}
	return unsafe.Pointer(&b[0])
}

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
