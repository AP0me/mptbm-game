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
