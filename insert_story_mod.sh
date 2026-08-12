#!/bin/bash
set -e

# =============================================================================
# Write all files to their correct locations.
# Contents are inserted exactly as provided (AS IS, no changes),
# except for client-go/display.go which gains the required "time" import.
# =============================================================================

# -----------------------------------------------------------------------------
# server-go/mods/shared/tree.go
# -----------------------------------------------------------------------------
mkdir -p server-go/mods/shared
cat > server-go/mods/shared/tree.go <<'TREE_EOF'
package shared

import (
	"fmt"
	"strings"
)

// ANSI constants used by the tree renderer
const (
	Dim          = "\033[2m"
	BrightCyan   = "\033[96m"
	BrightYellow = "\033[93m"
	BrightGreen  = "\033[92m"
	BrightWhite  = "\033[97m"
	BrightBlue   = "\033[94m"
	BrightRed    = "\033[91m"
)

// wrap applies ANSI codes then resets.
func wrap(text string, codes ...string) string {
	return strings.Join(codes, "") + text + Reset
}

// DirectedTree mirrors the Python DirectedTree.
// Branches maps a branch name to an ordered list of milestone strings.
// Selections maps a branch name to the current cursor index (-1 = nothing done).
// BranchOrder preserves the intended display order of branches.
type DirectedTree struct {
	Branches    map[string][]string `json:"branches"`
	Selections  map[string]int       `json:"selections"`
	BranchOrder []string             `json:"branch_order"`
}

func NewDirectedTree(branches map[string][]string, selections map[string]int, order []string) *DirectedTree {
	return &DirectedTree{
		Branches:    branches,
		Selections:  selections,
		BranchOrder: order,
	}
}

// indexOf returns the index of childKey within the branch, or -1 if not found.
func (t *DirectedTree) indexOf(layer1Key, childKey string) int {
	kids, ok := t.Branches[layer1Key]
	if !ok {
		return -1
	}
	for i, v := range kids {
		if v == childKey {
			return i
		}
	}
	return -1
}

// IsAdvanced returns true if the branch cursor has reached or passed childKey.
func (t *DirectedTree) IsAdvanced(layer1Key, childKey string) bool {
	current := t.Selections[layer1Key]
	target := t.indexOf(layer1Key, childKey)
	return current >= target
}

// AdvanceTo moves the branch cursor forward to childKey (never backward).
func (t *DirectedTree) AdvanceTo(layer1Key, childKey string) {
	if !t.IsAdvanced(layer1Key, childKey) {
		t.Selections[layer1Key] = t.indexOf(layer1Key, childKey)
	}
}

// CurrentChild returns the milestone at the current cursor for the branch.
func (t *DirectedTree) CurrentChild(layer1Key string) string {
	idx := t.Selections[layer1Key]
	kids := t.Branches[layer1Key]
	if idx < 0 || idx >= len(kids) {
		return ""
	}
	return kids[idx]
}

// Render produces the full ANSI tree visualization, matching the Python output.
func (t *DirectedTree) Render(title string) string {
	var lines []string
	lines = append(lines, "")
	bar := strings.Repeat("=", 60)

	lines = append(lines, wrap(bar, BrightCyan, Bold))
	lines = append(lines, wrap("  "+title, BrightCyan, Bold))
	lines = append(lines, wrap(bar, BrightCyan, Bold))
	lines = append(lines, "")

	// Root node
	lines = append(lines, wrap("  root", BrightGreen, Bold))
	lines = append(lines, "   │")

	for bIdx, layer1Key := range t.BranchOrder {
		kids := t.Branches[layer1Key]
		currentIdx := t.Selections[layer1Key]
		isLastBranch := bIdx == len(t.BranchOrder)-1

		connector := "├──"
		if isLastBranch {
			connector = "└──"
		}
		lines = append(lines,
			"  "+wrap(connector+" ", Cyan)+wrap(layer1Key, BrightYellow, Bold))

		vpipe := "│  "
		if isLastBranch {
			vpipe = "   "
		}
		vpipeStyled := wrap(vpipe, Cyan)

		for i, kid := range kids {
			isLastKid := i == len(kids)-1
			kidConnector := "├──"
			if isLastKid {
				kidConnector = "└──"
			}
			connectorStyled := wrap(kidConnector, Cyan)

			var marker, kidName string
			if i <= currentIdx {
				marker = wrap("[✓]", Dim, Green)
				kidName = wrap(kid, Green)
			} else {
				marker = wrap("[ ]", Dim, Gray)
				kidName = wrap(kid, Dim)
			}

			lines = append(lines,
				"  "+vpipeStyled+connectorStyled+" "+marker+" "+kidName)
		}

		if !isLastBranch {
			lines = append(lines, wrap("  │", Cyan))
		}
	}

	lines = append(lines, "")
	// Legend
	lines = append(lines,
		wrap("  Legend: ", Dim)+
			wrap("[✓]", Dim, Green)+
			wrap(" done   ", Dim)+
			wrap("[ ]", Dim, Gray)+
			wrap(" not done", Dim))
	lines = append(lines, wrap(bar, BrightCyan, Bold))
	lines = append(lines, "")

	return strings.Join(lines, "\n")
}

// Dump is a plain-text debug dump (no ANSI).
func (t *DirectedTree) Dump() string {
	var b strings.Builder
	for _, k := range t.BranchOrder {
		idx := t.Selections[k]
		for i, c := range t.Branches[k] {
			if i == idx {
				fmt.Fprintf(&b, "  %s -> %s *\n", k, c)
			} else if i < idx {
				fmt.Fprintf(&b, "  %s -> %s (learned)\n", k, c)
			} else {
				fmt.Fprintf(&b, "  %s -> %s\n", k, c)
			}
		}
	}
	return b.String()
}
TREE_EOF

# -----------------------------------------------------------------------------
# server-go/mods/shared/main.go
# -----------------------------------------------------------------------------
cat > server-go/mods/shared/main.go <<'SHARED_MAIN_EOF'
package shared

import (
	"net"
)

// GameState is the top-level state container.
// It wraps a DirectedTree (progression) and EventLogs (messages).
type GameState struct {
	Tree      *DirectedTree `json:"tree"`
	EventLogs []string      `json:"event_logs"`

	// — Engine-level fields (kept for compatibility) —
	Status       string   `json:"status,omitempty"`
	PlayerOrder  []string `json:"player_order,omitempty"`
	ActingPlayer string   `json:"acting_player,omitempty"`
}

// Player represents a connected human or a robot (WASM-side decision).
type Player struct {
	Name   string
	Conn   net.Conn
	Decide func(state *GameState) string
}

// Card is the engine's representation of a playable option.
type Card struct {
	Name       string
	Conditions func(state *GameState) bool
	Action     func(state *GameState)
}

// HostPlayer is sent to the client so it knows which seats are robots.
type HostPlayer struct {
	Name string `json:"name"`
}

// CardProfile is the slim card description sent to clients for display.
type CardProfile struct {
	Name string `json:"name"`
}

// ---------------------------------------------------------------------------
// Event-log helpers (the only state mutations the story mod needs)
// ---------------------------------------------------------------------------

func LogEvent(s *GameState, msg string) {
	s.EventLogs = append(s.EventLogs, msg)
}

func ClearEventLogs(s *GameState) {
	s.EventLogs = []string{}
}

// ---------------------------------------------------------------------------
// Deprecated flat-map helpers — kept for backward compatibility but the
// story mod does NOT use them.  If you need typed access, use the tree.
// ---------------------------------------------------------------------------

func GetStringList(s *GameState, key string) []string {
	// Only works for top-level keys that are []string in a hypothetical
	// flat map.  The tree-based state doesn't store flat maps, so this
	// is a no-op stub.  Kept so the engine compiles if it references it.
	_ = key
	return []string{}
}

func GetActingPlayer(s *GameState) string {
	return s.ActingPlayer
}
SHARED_MAIN_EOF

# -----------------------------------------------------------------------------
# server-go/mods/shared/wasm.go
# -----------------------------------------------------------------------------
cat > server-go/mods/shared/wasm.go <<'SHARED_WASM_EOF'
package shared

import (
	"encoding/json"
	"unsafe"
)

// ---------------------------------------------------------------------------
// Memory allocation pinning (so the Go GC doesn't reclaim buffers we
// returned to the host)
// ---------------------------------------------------------------------------

var activeAllocations = make(map[uint32][]byte)

//go:wasmexport Allocate
func Allocate(size uint32) uint32 {
	if size == 0 {
		return 0
	}
	buf := make([]byte, size)
	ptr := uint32(uintptr(unsafe.Pointer(unsafe.SliceData(buf))))
	activeAllocations[ptr] = buf
	return ptr
}

func packPtr(b []byte) uint64 {
	if len(b) == 0 {
		return 0
	}
	ptr := uint32(uintptr(unsafe.Pointer(unsafe.SliceData(b))))
	size := uint32(len(b))
	return (uint64(ptr) << 32) | uint64(size)
}

// ---------------------------------------------------------------------------
// PlayableCards — given a deck and the current state, return which cards
// are currently playable as a JSON map[string]CardProfile.
// ---------------------------------------------------------------------------

func PlayableCards(deck map[string]*Card, statePtr uint32, stateSize uint32) uint64 {
	stateBytes := unsafe.Slice((*byte)(unsafe.Pointer(uintptr(statePtr))), stateSize)

	var state GameState
	if err := json.Unmarshal(stateBytes, &state); err != nil {
		return 0
	}
	delete(activeAllocations, statePtr)

	playable := make(map[string]CardProfile)
	for key, card := range deck {
		if card.Conditions != nil && card.Conditions(&state) {
			playable[key] = CardProfile{Name: card.Name}
		}
	}
	if len(playable) == 0 {
		playable["skip"] = CardProfile{Name: "Skip"}
	}

	buf, err := json.Marshal(playable)
	if err != nil {
		return 0
	}
	lastPlayableCardsOutput = buf
	return packPtr(lastPlayableCardsOutput)
}

var lastPlayableCardsOutput []byte

// ---------------------------------------------------------------------------
// PlayCardAction — execute the action for cardKey against state, return
// the updated state as JSON.
// ---------------------------------------------------------------------------

func PlayCardAction(deck map[string]*Card, keyPtr uint32, keySize uint32, statePtr uint32, stateSize uint32) uint64 {
	keyBytes := unsafe.Slice((*byte)(unsafe.Pointer(uintptr(keyPtr))), keySize)
	var cardKey string
	if err := json.Unmarshal(keyBytes, &cardKey); err != nil {
		return 0
	}

	stateBytes := unsafe.Slice((*byte)(unsafe.Pointer(uintptr(statePtr))), stateSize)
	var state GameState
	if err := json.Unmarshal(stateBytes, &state); err != nil {
		return 0
	}
	delete(activeAllocations, keyPtr)
	delete(activeAllocations, statePtr)

	if card, exists := deck[cardKey]; exists && card.Action != nil {
		card.Action(&state)
	}

	buf, err := json.Marshal(state)
	if err != nil {
		return 0
	}
	lastPlayCardOutput = buf
	return packPtr(lastPlayCardOutput)
}

var lastPlayCardOutput []byte

// ---------------------------------------------------------------------------
// InitRobotsNames — returns an empty map for the story mod (no robots).
// ---------------------------------------------------------------------------

func InitRobotsNames(robots map[string]*Player) uint64 {
	names := map[string]HostPlayer{}
	for name, player := range robots {
		names[name] = HostPlayer{Name: player.Name}
	}
	buf, err := json.Marshal(names)
	if err != nil {
		return 0
	}
	lastRobotsOutput = buf
	return packPtr(lastRobotsOutput)
}

var lastRobotsOutput []byte

// ---------------------------------------------------------------------------
// ChooseCard — robot decision (not used by story mod, kept for compat).
// ---------------------------------------------------------------------------

func ChooseCard(localRobots map[string]*Player, deck map[string]*Card, namePtr uint32, nameSize uint32, statePtr uint32, stateSize uint32) uint64 {
	nameBytes := unsafe.Slice((*byte)(unsafe.Pointer(uintptr(namePtr))), nameSize)
	var playerName string
	if err := json.Unmarshal(nameBytes, &playerName); err != nil {
		return returnRawString("skip")
	}

	stateBytes := unsafe.Slice((*byte)(unsafe.Pointer(uintptr(statePtr))), stateSize)
	var state GameState
	if err := json.Unmarshal(stateBytes, &state); err != nil {
		return returnRawString("skip")
	}
	delete(activeAllocations, namePtr)
	delete(activeAllocations, statePtr)

	robot, exists := localRobots[playerName]
	if !exists || robot.Decide == nil {
		return returnRawString("skip")
	}

	key := robot.Decide(&state)

	playable := make(map[string]CardProfile)
	for k, card := range deck {
		if card.Conditions != nil && card.Conditions(&state) {
			playable[k] = CardProfile{Name: card.Name}
		}
	}
	if _, ok := playable[key]; !ok {
		key = "skip"
	}
	return returnRawString(key)
}

var lastChooseCardOutput []byte

func returnRawString(val string) uint64 {
	lastChooseCardOutput = []byte(val)
	return packPtr(lastChooseCardOutput)
}
SHARED_WASM_EOF

# -----------------------------------------------------------------------------
# server-go/mods/vanilla/main.go
# -----------------------------------------------------------------------------
mkdir -p server-go/mods/vanilla
cat > server-go/mods/vanilla/main.go <<'VANILLA_MAIN_EOF'
package main

import (
	"encoding/json"
	"unsafe"

	"game/server/mods/shared"
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
		"begin":       begin,
		"recruit":     recruit,
		"confront":    confront,
		"rumor":       rumor,
		"clue":        clue,
		"artifact":    artifact,
		"skip":        skip,
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
VANILLA_MAIN_EOF

# -----------------------------------------------------------------------------
# server-go/core/wasm.go
# -----------------------------------------------------------------------------
mkdir -p server-go/core
cat > server-go/core/wasm.go <<'CORE_WASM_EOF'
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
CORE_WASM_EOF

# -----------------------------------------------------------------------------
# server-go/core/network.go
# -----------------------------------------------------------------------------
cat > server-go/core/network.go <<'CORE_NET_EOF'
package core

import (
	"encoding/json"
	"fmt"
	"game/server/mods/shared"
	"net"
)

type Packet struct {
	Type string `json:"type"`
	Data any    `json:"data"`
}

func SafeSend(conn net.Conn, data any, pType string) bool {
	packet := Packet{Type: pType, Data: data}
	encoded, err := json.Marshal(packet)
	if err != nil {
		return false
	}
	_, err = conn.Write(append(encoded, '\n'))
	if err != nil {
		fmt.Printf("[Server] Write error: %v\n", err)
		return false
	}
	return true
}

func PacketMultiSend(conns []net.Conn, data any, pType string) {
	for _, conn := range conns {
		SafeSend(conn, data, pType)
	}
}

// SendState sends the full tree-based GameState to all clients.
func SendState(state shared.GameState, conns []net.Conn) {
	PacketMultiSend(conns, state, "STATE")
}

func SendCards(playableCards map[string]shared.CardProfile, conn net.Conn) {
	cardNames := make(map[string]map[string]string)
	for key, card := range playableCards {
		cardNames[key] = map[string]string{"name": card.Name}
	}
	SafeSend(conn, cardNames, "CARDS")
}

func SendMessages(logs []string, conns []net.Conn) {
	if logs == nil {
		logs = []string{}
	}
	PacketMultiSend(conns, logs, "MESSAGE")
}
CORE_NET_EOF

# -----------------------------------------------------------------------------
# server-go/core/player.go
# -----------------------------------------------------------------------------
cat > server-go/core/player.go <<'CORE_PLAYER_EOF'
package core

import (
	"bufio"
	"fmt"
	"game/server/mods/shared"
	"net"
	"strings"
)

func HumanInput(conn net.Conn) string {
	SafeSend(conn, map[string]any{}, "CHOICE")
	reader := bufio.NewReader(conn)
	input, _ := reader.ReadString('\n')
	return strings.TrimSpace(input)
}

// AcceptHumanConnection blocks until one human client connects, then returns
// the connection.  The story mod is single-player, so we only need one.
func AcceptHumanConnection(ln net.Listener) net.Conn {
	fmt.Println("Waiting for a player to connect...")
	conn, err := ln.Accept()
	if err != nil {
		fmt.Printf("[Server] Accept error: %v\n", err)
		return AcceptHumanConnection(ln)
	}
	fmt.Printf("[Server] Player connected from %s\n", conn.RemoteAddr())
	return conn
}
CORE_PLAYER_EOF

# -----------------------------------------------------------------------------
# client-go/display.go (with the required "time" import added)
# -----------------------------------------------------------------------------
mkdir -p client-go
cat > client-go/display.go <<'DISPLAY_EOF'
package main

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	Reset   = "\033[0m"
	Bold    = "\033[1m"
	Dim     = "\033[2m"
	Red     = "\033[31m"
	Green   = "\033[32m"
	Yellow  = "\033[33m"
	Blue    = "\033[34m"
	Magenta = "\033[35m"
	Cyan    = "\033[36m"
	White   = "\033[37m"
	Gray    = "\033[90m"

	BrightRed    = "\033[91m"
	BrightGreen  = "\033[92m"
	BrightYellow = "\033[93m"
	BrightBlue   = "\033[94m"
	BrightCyan   = "\033[96m"
	BrightWhite  = "\033[97m"
)

func PrintError(message string) {
	fmt.Printf("\n%s%s[ERROR]: %s%s\n\n", Bold, Red, message, Reset)
}

func CenterText(text string, width int) string {
	length := utf8.RuneCountInString(text)
	if length >= width {
		return text
	}
	paddingLeft := (width - length) / 2
	paddingRight := width - length - paddingLeft
	return strings.Repeat(" ", paddingLeft) + text + strings.Repeat(" ", paddingRight)
}

// ---------------------------------------------------------------------------
// TreeState — local types for unmarshaling the STATE packet.
// The client doesn't import shared, so we mirror the JSON shape here.
// ---------------------------------------------------------------------------

type treeState struct {
	Tree struct {
		Branches    map[string][]string `json:"branches"`
		Selections  map[string]int      `json:"selections"`
		BranchOrder []string             `json:"branch_order"`
	} `json:"tree"`
	EventLogs []string `json:"event_logs"`
}

func wrap(text string, codes ...string) string {
	return strings.Join(codes, "") + text + Reset
}

// PrintTree replaces the old PrintState. It unmarshals the flat
// map[string]any back into a typed treeState and renders the ANSI tree.
func PrintTree(data map[string]any) {
	// Clear screen and home cursor
	fmt.Print("\033[H\033[J")

	// Re-marshal to JSON, then unmarshal into typed struct
	raw, err := json.Marshal(data)
	if err != nil {
		fmt.Println("Error marshaling state:", err)
		return
	}

	var ts treeState
	if err := json.Unmarshal(raw, &ts); err != nil {
		fmt.Println("Error parsing tree state:", err)
		return
	}

	// Render tree
	fmt.Print(renderTree(&ts))
}

func renderTree(ts *treeState) string {
	var lines []string
	lines = append(lines, "")
	bar := strings.Repeat("=", 60)

	lines = append(lines, wrap(bar, BrightCyan, Bold))
	lines = append(lines, wrap("  Story Tree", BrightCyan, Bold))
	lines = append(lines, wrap(bar, BrightCyan, Bold))
	lines = append(lines, "")

	// Root node
	lines = append(lines, wrap("  root", BrightGreen, Bold))
	lines = append(lines, "   │")

	for bIdx, layer1Key := range ts.Tree.BranchOrder {
		kids := ts.Tree.Branches[layer1Key]
		currentIdx := ts.Tree.Selections[layer1Key]
		isLastBranch := bIdx == len(ts.Tree.BranchOrder)-1

		connector := "├──"
		if isLastBranch {
			connector = "└──"
		}
		lines = append(lines,
			"  "+wrap(connector+" ", Cyan)+wrap(layer1Key, BrightYellow, Bold))

		vpipe := "│  "
		if isLastBranch {
			vpipe = "   "
		}
		vpipeStyled := wrap(vpipe, Cyan)

		for i, kid := range kids {
			isLastKid := i == len(kids)-1
			kidConnector := "├──"
			if isLastKid {
				kidConnector = "└──"
			}
			connectorStyled := wrap(kidConnector, Cyan)

			var marker, kidName string
			if i <= currentIdx {
				marker = wrap("[✓]", Dim, Green)
				kidName = wrap(kid, Green)
			} else {
				marker = wrap("[ ]", Dim, Gray)
				kidName = wrap(kid, Dim)
			}

			lines = append(lines,
				"  "+vpipeStyled+connectorStyled+" "+marker+" "+kidName)
		}

		if !isLastBranch {
			lines = append(lines, wrap("  │", Cyan))
		}
	}

	lines = append(lines, "")
	// Legend
	lines = append(lines,
		wrap("  Legend: ", Dim)+
			wrap("[✓]", Dim, Green)+
			wrap(" done   ", Dim)+
			wrap("[ ]", Dim, Gray)+
			wrap(" not done", Dim))
	lines = append(lines, wrap(bar, BrightCyan, Bold))
	lines = append(lines, "")

	return strings.Join(lines, "\n")
}

// ---------------------------------------------------------------------------
// PrintCards — shows available options (unchanged logic, restyled)
// ---------------------------------------------------------------------------

func PrintCards(cards map[string]any) {
	boxWidth := 44
	fmt.Printf("\n%s%s╔%s╗%s\n", Magenta, Bold, strings.Repeat("═", boxWidth), Reset)
	fmt.Printf("%s%s║%s║%s\n", Bold, Magenta, CenterText("~ AVAILABLE OPTIONS ~", boxWidth), Reset)
	fmt.Printf("%s%s╠%s╣%s\n", Magenta, Bold, strings.Repeat("═", boxWidth), Reset)

	for key, val := range cards {
		cardData := val.(map[string]any)
		name := fmt.Sprintf("%v", cardData["name"])

		cleanText := fmt.Sprintf("[%s] %s", key, name)
		padding := boxWidth - 3 - utf8.RuneCountInString(cleanText)
		if padding < 0 {
			padding = 0
		}

		fmt.Printf("%s%s║%s %s%s%s: %s%s%s%s %s%s%s ║%s\n",
			Magenta, Bold, Reset,
			Bold, Yellow, key,
			Bold, Cyan, name, Reset,
			strings.Repeat(" ", padding),
			Magenta, Bold, Reset)
	}

	fmt.Printf("%s%s╚%s╝%s\n\n", Magenta, Bold, strings.Repeat("═", boxWidth), Reset)
}

func TypeWrite(text string) {
	for _, char := range text {
		fmt.Print(string(char))
		time.Sleep(30 * time.Millisecond)
	}
	fmt.Println()
}

func PrintCardResponse(messages []any) {
	for _, msg := range messages {
		if m, ok := msg.(string); ok {
			fmt.Printf("%s%s» %s", White, Bold, Reset)
			TypeWrite(m)
			fmt.Print(Reset)
			time.Sleep(1 * time.Second)
		}
	}
}
DISPLAY_EOF

# -----------------------------------------------------------------------------
# client-go/main.go
# -----------------------------------------------------------------------------
cat > client-go/main.go <<'CLIENT_MAIN_EOF'
package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"strings"
)

type Packet struct {
	Type string `json:"type"`
	Data any    `json:"data"`
}

func ConnectToServer() net.Conn {
	reader := bufio.NewReader(os.Stdin)
	fmt.Println("--- Server Connection ---")

	fmt.Print("Address: (e.g: 127.0.0.1) ")
	address, _ := reader.ReadString('\n')
	address = strings.TrimSpace(address)
	if address == "" {
		address = "127.0.0.1"
	}

	fmt.Print("Port: (e.g: 8080) ")
	port, _ := reader.ReadString('\n')
	port = strings.TrimSpace(port)
	if port == "" {
		port = "8080"
	}

	target := net.JoinHostPort(address, port)
	fmt.Printf("Attempting to connect to %s...\n", target)

	conn, err := net.Dial("tcp", target)
	if err != nil {
		PrintError("Connection refused: " + err.Error())
		return ConnectToServer()
	}

	fmt.Printf("%sConnected successfully!%s\n", Green, Reset)
	fmt.Printf("%sWelcome to the Story Game!%s\n", Yellow, Reset)
	return conn
}

func main() {
	conn := ConnectToServer()
	defer conn.Close()

	scanner := bufio.NewScanner(conn)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	stdinReader := bufio.NewReader(os.Stdin)

	for scanner.Scan() {
		line := scanner.Text()
		var packet Packet
		if err := json.Unmarshal([]byte(line), &packet); err != nil {
			continue
		}

		switch packet.Type {
		case "STATE":
			if data, ok := packet.Data.(map[string]any); ok {
				PrintTree(data) // ← was PrintState
			}
		case "CARDS":
			if data, ok := packet.Data.(map[string]any); ok {
				PrintCards(data)
			}
		case "CHOICE":
			fmt.Printf("%s%s➤ Enter option key (e.g. begin, recruit, confront): %s",
				Yellow, Bold, Reset)
			choice, _ := stdinReader.ReadString('\n')
			conn.Write([]byte(choice))
		case "MESSAGE":
			if data, ok := packet.Data.([]any); ok {
				PrintCardResponse(data)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		PrintError("Connection lost: " + err.Error())
	}
}
CLIENT_MAIN_EOF

# -----------------------------------------------------------------------------
# server-go/main.go
# -----------------------------------------------------------------------------
cat > server-go/main.go <<'SERVER_MAIN_EOF'
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
SERVER_MAIN_EOF

echo "All files written successfully."
