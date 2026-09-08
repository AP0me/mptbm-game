package shared

import (
	"encoding/json"
	"unsafe"
)


func InitRobotsNames(robots map[string]*Player) uint64 {
	names := map[string]HostPlayer{}
	for name, player := range robots {
		names[name] = HostPlayer{Name: player.Name}
	}

	buf, err := json.Marshal(names)
	if err != nil {
		return 0
	}

	ptr := uint32(uintptr(unsafe.Pointer(&buf[0])))
	size := uint32(len(buf))
	return (uint64(ptr) << 32) | uint64(size)
}

var (
	lastPlayableCardsOutput []byte
	lastChooseCardOutput    []byte
)

// Global map to pin memory allocated for INCOMING host payloads
var activeAllocations = make(map[uint32][]byte)

//go:wasmexport Allocate
func Allocate(size uint32) uint32 {
	if size == 0 {
		return 0
	}

	// Allocate slice safely
	buf := make([]byte, size)
	ptr := uint32(uintptr(unsafe.Pointer(unsafe.SliceData(buf))))

	// PIN THE MEMORY: Store it globally so the GC cannot reclaim it
	activeAllocations[ptr] = buf

	return ptr
}

func PlayableCards(deck map[string]*Card, statePtr uint32, stateSize uint32) uint64 {
	stateBytes := unsafe.Slice((*byte)(unsafe.Pointer(uintptr(statePtr))), stateSize)

	var state GameState
	if err := json.Unmarshal(stateBytes, &state); err != nil {
		return 0
	}

	// CLEANUP: We parsed the JSON, the host input data is no longer needed
	delete(activeAllocations, statePtr)

	// Run game logic
	playable := make(map[string]CardProfile)
	for key, card := range deck {
		if card.Conditions(&state) {
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

	// Keep alive globally, return packed pointer
	lastPlayableCardsOutput = buf
	ptr := uint32(uintptr(unsafe.Pointer(unsafe.SliceData(lastPlayableCardsOutput))))
	size := uint32(len(lastPlayableCardsOutput))
	return (uint64(ptr) << 32) | uint64(size)
}

func ChooseCard(localRobots map[string]*Player, deck map[string]*Card, namePtr uint32, nameSize uint32, statePtr uint32, stateSize uint32) uint64 {
	// 1. Parse Player Name and GameState data
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

	// CLEANUP: Free host input structures
	delete(activeAllocations, namePtr)
	delete(activeAllocations, statePtr)

	// 3. Find the local robot execution logic
	robot, exists := localRobots[playerName]
	if !exists || robot.Decide == nil {
		return returnRawString("skip")
	}

	// 4. Execute decision logic safely INSIDE the WASM context
	key := robot.Decide(&state)

	// Check what cards are valid right now
	playable := make(map[string]CardProfile)
	for k, card := range deck {
		if card.Conditions(&state) {
			playable[k] = CardProfile{Name: card.Name}
		}
	}

	// Validation: If the robot decided on an unplayable card, force a "skip"
	if _, ok := playable[key]; !ok {
		key = "skip"
	}

	return returnRawString(key)
}

// Helper to wrap a raw string into a packed uint64 safely without JSON overhead
func returnRawString(val string) uint64 {
	lastChooseCardOutput = []byte(val)
	ptr := uint32(uintptr(unsafe.Pointer(unsafe.SliceData(lastChooseCardOutput))))
	size := uint32(len(lastChooseCardOutput))
	return (uint64(ptr) << 32) | uint64(size)
}

// Add this near your other last*Output variables
var lastPlayCardOutput []byte

// Add this function to handle the action execution
func PlayCardAction(deck map[string]*Card, keyPtr uint32, keySize uint32, statePtr uint32, stateSize uint32) uint64 {
	// 1. Parse Card Key
	keyBytes := unsafe.Slice((*byte)(unsafe.Pointer(uintptr(keyPtr))), keySize)
	var cardKey string
	if err := json.Unmarshal(keyBytes, &cardKey); err != nil {
		return 0
	}

	// 2. Parse GameState
	stateBytes := unsafe.Slice((*byte)(unsafe.Pointer(uintptr(statePtr))), stateSize)
	var state GameState
	if err := json.Unmarshal(stateBytes, &state); err != nil {
		return 0
	}

	// CLEANUP: Free host input structures
	delete(activeAllocations, keyPtr)
	delete(activeAllocations, statePtr)

	// 3. Execute the Action
	card, exists := deck[cardKey];
	if !exists {
		card, exists = deck["skip"];
	}
	if !exists {
		return 0;
	}

	if card.Action != nil {
		card.Action(&state)
	}

	// 4. Marshal and return the updated state
	buf, err := json.Marshal(state)
	if err != nil {
		return 0
	}

	lastPlayCardOutput = buf
	ptr := uint32(uintptr(unsafe.Pointer(unsafe.SliceData(lastPlayCardOutput))))
	size := uint32(len(lastPlayCardOutput))
	return (uint64(ptr) << 32) | uint64(size)
}
