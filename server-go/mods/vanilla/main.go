package main

import (
	"encoding/json"
	"game/server/mods/shared"
	"math"
	"math/rand"
	"slices"
	"time"
	"unsafe"
)

func humanKeys() []string {
	return []string{"anar", "elshad"}
}

func robotKeys() []string {
	keys := []string{}
	for k := range InitRobots() {
		keys = append(keys, k)
	}
	return keys
}

func RobotInput(val string) string {
	time.Sleep(1 * time.Second)
	return val
}

func isHuman(p string) bool {
	return slices.Contains(humanKeys(), p)
}

func PropedSet(s *shared.GameState, key string, val int) {
	if val < 0 {
		val = 0
	}
	s.Data[key] = val
}

func PlayerDotKey(s *shared.GameState, key string) string {
	p := shared.GetActingPlayer(s)
	return p + "." + key
}

func LocalDotKey(s *shared.GameState, key string) string {
	loc := shared.GetString(s, PlayerDotKey(s, "location"))
	return loc + "." + key
}

func LogEvent(s *shared.GameState, msg string) {
	logs := shared.GetStringList(s, "event_logs")
	s.Data["event_logs"] = append(logs, msg)
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
	s.Data["acting_player"] = order[nextIdx]
}

func RemovePlayers(s *shared.GameState, playerKeys []string) {
	order := shared.GetStringList(s, "player_order")

	var newOrder []string
	for _, player := range order {
		if !slices.Contains(playerKeys, player) {
			newOrder = append(newOrder, player)
		}
	}

	s.Data["player_order"] = newOrder
}

func AddEnergy(s *shared.GameState, amount int, deathMsg string) bool {
	key := PlayerDotKey(s, "energy")
	
	current := shared.GetInt(s, key)
	max := shared.GetInt(s, PlayerDotKey(s, "max_energy"))
	
	newVal := current + amount
	if newVal <= 0 {
		PropedSet(s, key, 0)
		LogEvent(s, deathMsg)
		RemovePlayers(s, []string{shared.GetActingPlayer(s)})
		EndTurn(s)
		return true
	}
	
	if newVal > max {
		newVal = max
	}
	PropedSet(s, key, newVal)
	return false
}

func SunLightLevel(s *shared.GameState) int {
	t, _ := time.Parse("2006-01-02 15:04:05", shared.GetString(s, "date_time"))
	hour := t.Hour()
	
	// Simplified sun logic: Peak at 12:00, 0 at night
	if hour >= 6 && hour <= 18 {
		progress := float64(hour-6) / 12.0
		level := math.Sin(progress * math.Pi)
		return int(math.Round(level * 10))
	}
	return 0
}

func LightLevel(s *shared.GameState) int {
	sunLightLevel := SunLightLevel(s)

	fire_light_level := 0
	if shared.GetInt(s, LocalDotKey(s, "fire_minutes")) > 0 {
		fire_light_level = 4
	}

	return sunLightLevel + fire_light_level
}

func TimePasses(s *shared.GameState, minutes int) bool {
	t, _ := time.Parse("2006-01-02 15:04:05", shared.GetString(s, "date_time"))
	newTime := t.Add(time.Duration(minutes) * time.Minute)
	s.Data["date_time"] = newTime.Format("2006-01-02 15:04:05")

	// 2. Decay fires across all human player locations
	humans := []string{"anar", "elshad"}
	for _, pKey := range humans {
		playerLoc := shared.GetString(s, pKey+".location")
		if playerLoc != "" {
			fireKey := playerLoc + ".fire_minutes"
			currentFire := shared.GetInt(s, fireKey)
			if currentFire > 0 {
				newFire := currentFire - minutes
				if newFire < 0 {
					newFire = 0
				}
				PropedSet(s, fireKey, newFire)
			}
		}
	}
	baseEnergySpent := -10.0 * (float64(minutes) / 60.0)
	if shared.GetBool(s, PlayerDotKey(s, "sleeping")) {
		baseEnergySpent /= 5.0
	}

	if shared.GetBool(s, LocalDotKey(s, "shelter")) {
		baseEnergySpent /= 2.0
	}

	if shared.GetInt(s, LocalDotKey(s, "fire_minutes")) > 0 {
		baseEnergySpent /= 2.0
	}

	finalEnergySpent := int(math.Round(baseEnergySpent))
	return AddEnergy(s, finalEnergySpent, "shared.Player died of hunger.")
}

func ClearEventLogs(s *shared.GameState) {
	s.Data["event_logs"] = []string{}
}

//go:wasmexport InitState
func InitState() uint64 {
	state := &shared.GameState{
		Data: map[string]interface{}{
			"status":          "RUNNING",
			"event_logs":      []string{},
			"invisible_keys":  []string{"invisible_keys", "event_logs"},
			"round":           1,
			"player_order":    slices.Concat(humanKeys(), robotKeys()),
			"acting_player":   "anar",
			"date_time":       time.Now().Format("2006-01-02 00:00:00"),
			"anar.location":   "forest",
			"anar.energy":     100,
			"anar.max_energy": 100,
			"forest.wood":     0,
			"forest.food":     0,
		},
	}
	
	buf, err := json.Marshal(state)
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

func InitRobots() map[string]*shared.Player {
	return map[string]*shared.Player{
		"round": {
			Name: "round",
			Decide: func(s *shared.GameState) string {
				return RobotInput("end_of_round")
			},
		},
	}
}

//go:wasmexport InitRobotsNames
func InitRobotsNames() uint64 {
    robots := InitRobots()
    names := map[string]shared.HostPlayer{};
    for name, player := range robots {
        names[name] = shared.HostPlayer{Name: player.Name}
    }

    buf, err := json.Marshal(names)
    if err != nil {
        return 0
    }

    ptr := uint32(uintptr(unsafe.Pointer(&buf[0])))
    size := uint32(len(buf))
    return (uint64(ptr) << 32) | uint64(size)
}

func InitDeck(s *shared.GameState) map[string]*shared.Card {
	return map[string]*shared.Card{
		"skip": {
			Name: "Skip",
			Conditions: func(state *shared.GameState) bool { return true },
			Action: func(state *shared.GameState) {
				ClearEventLogs(state)
				EndTurn(state)
				LogEvent(state, "Let's see what else is happening.")
			},
		},
		"eat": {
			Name: "Eat 10 food",
			Conditions: func(state *shared.GameState) bool {
				return isHuman(shared.GetActingPlayer(state)) && shared.GetInt(state, LocalDotKey(state, "food")) > 0
			},
			Action: func(state *shared.GameState) {
				ClearEventLogs(state)

				if TimePasses(state, 30) {
					return
				}

				cooked := 20
				if shared.GetInt(state, LocalDotKey(state, "fire_minutes")) > 0 {
					cooked = 50
				}

				if AddEnergy(state, cooked, "Died eating") {
					return
				}

				PropedSet(state, LocalDotKey(state, "food"), shared.GetInt(state, LocalDotKey(state, "food")) - 15)

				message := "The player ate raw food."
				if cooked > 20 {
					message = "The player ate a cooked meal."
				}
				LogEvent(state, message)
			},
		},
		"hunt": {
			Name: "Hunt game",
			Conditions: func(state *shared.GameState) bool {
				return isHuman(shared.GetActingPlayer(state))
			},
			Action: func(state *shared.GameState) {
				ClearEventLogs(state)
				isDay := SunLightLevel(state) > 0
				yield := 3
				if isDay {
					yield = 12
				}
				PropedSet(state, LocalDotKey(state, "food"), shared.GetInt(state, LocalDotKey(state, "food")) + yield)

				if AddEnergy(state, -10, "Died hunting") {
					return
				}
				if TimePasses(state, 60) {
					return
				}

				message := "Hunting at night was difficult."
				if isDay {
					message = "The daytime hunt was highly productive."
				}
				LogEvent(state, message)
			},
		},
		"wood": {
			Name: "Collect wood",
			Conditions: func(state *shared.GameState) bool {
				return isHuman(shared.GetActingPlayer(state)) && shared.GetInt(state, PlayerDotKey(state, "energy")) > 5
			},
			Action: func(state *shared.GameState) {
				ClearEventLogs(state)
				if AddEnergy(state, -5, "Died collecting wood") {
					return
				}

				hasLight := LightLevel(state) > 3
				yield := 3
				if hasLight {
					yield = 5
				}

				PropedSet(state, LocalDotKey(state, "wood"), shared.GetInt(state, LocalDotKey(state, "wood")) + yield)

				if TimePasses(state, 60) {
					return
				}

				message := "The player foraged for wood. Lack of visibility made foraging challenging."
				if hasLight {
					message = "The player foraged for wood."
				}
				LogEvent(state, message)
			},
		},
		"shelter": {
			Name: "Build a shelter (50 wood)",
			Conditions: func(state *shared.GameState) bool {
				return isHuman(shared.GetActingPlayer(state)) && shared.GetInt(state, LocalDotKey(state, "wood")) >= 50
			},
			Action: func(state *shared.GameState) {
				ClearEventLogs(state)

				if AddEnergy(state, -45, "Died building shelter") {
					return
				}
				if TimePasses(state, 60) {
					return
				}

				state.Data[LocalDotKey(state, "shelter")] = true
				PropedSet(state, LocalDotKey(state, "wood"), shared.GetInt(state, LocalDotKey(state, "wood")) - 50)

				LogEvent(state, "The player built a shelter.")
			},
		},
		"boat": {
			Name: "Build a boat (250 wood)",
			Conditions: func(state *shared.GameState) bool {
				return isHuman(shared.GetActingPlayer(state)) && shared.GetInt(state, LocalDotKey(state, "wood")) >= 250
			},
			Action: func(state *shared.GameState) {
				ClearEventLogs(state)

				if AddEnergy(state, -45, "Died building boat") {
					return
				}
				if TimePasses(state, 60) {
					return
				}

				state.Data[LocalDotKey(state, "boat")] = true
				PropedSet(state, LocalDotKey(state, "wood"), shared.GetInt(state, LocalDotKey(state, "wood")) - 250)

				LogEvent(state, "The player built a boat.")
			},
		},
		"fish": {
			Name: "Go fishing",
			Conditions: func(state *shared.GameState) bool {
				return isHuman(shared.GetActingPlayer(state)) && shared.GetBool(state, LocalDotKey(state, "boat"))
			},
			Action: func(state *shared.GameState) {
				ClearEventLogs(state)
				lucky := rand.Intn(2) == 1
				yield := 3
				if lucky {
					yield = 12
				}

				PropedSet(state, LocalDotKey(state, "food"), shared.GetInt(state, LocalDotKey(state, "food")) + yield)

				if TimePasses(state, 60) {
					return
				}

				if SunLightLevel(state) > 6 && !shared.GetBool(state, "bottle_map") {
					state.Data["bottle_map"] = true
					LogEvent(state, "The player fishes out a map in a bottle!")
					return
				}

				message := "The fishing trip was unlucky."
				if lucky {
					message = "The player caught a big fish."
				}
				LogEvent(state, message)
			},
		},
		"fire": {
			Name: "Make fire (up to 10 wood)",
			Conditions: func(state *shared.GameState) bool {
				return isHuman(shared.GetActingPlayer(state)) && shared.GetInt(state, LocalDotKey(state, "wood")) > 0
			},
			Action: func(state *shared.GameState) {
				ClearEventLogs(state)
				fromScratch := false

				if shared.GetInt(state, LocalDotKey(state, "fire_minutes")) <= 0 {
					fromScratch = true
					if AddEnergy(state, -45, "Died making fire from scratch") {
						return
					}
					if TimePasses(state, 60) {
						return
					}
				}

				currentWood := shared.GetInt(state, LocalDotKey(state, "wood"))
				woodToBurn := currentWood
				if woodToBurn > 10 {
					woodToBurn = 10
				}

				burnMinutes := woodToBurn * 90
				PropedSet(state, LocalDotKey(state, "fire_minutes"), shared.GetInt(state, LocalDotKey(state, "fire_minutes")) + burnMinutes)
				PropedSet(state, LocalDotKey(state, "wood"), currentWood - woodToBurn)

				message := "The player stokes the fire with more wood."
				if fromScratch {
					message = "The player rubs sticks together to make fire. It was exhausting and time consuming."
				}
				LogEvent(state, message)
			},
		},
		"follow_the_map": {
			Name: "Follow the map the player fished out.",
			Conditions: func(state *shared.GameState) bool {
				return isHuman(shared.GetActingPlayer(state)) && shared.GetBool(state, "bottle_map")
			},
			Action: func(state *shared.GameState) {

				food := shared.GetInt(state, LocalDotKey(state, "food"))
				wood := shared.GetInt(state, LocalDotKey(state, "wood"))
				fire := shared.GetInt(state, LocalDotKey(state, "fire_minutes"))

				PropedSet(state, LocalDotKey(state, "food"), food - 10)
				PropedSet(state, LocalDotKey(state, "wood"), wood - 10)
				PropedSet(state, LocalDotKey(state, "fire_minutes"), fire - 60)

				newLoc := "caves"
				state.Data[PlayerDotKey(state, "location")] = newLoc

				PropedSet(state, newLoc+".food", shared.GetInt(state, newLoc+".food") + food)
				PropedSet(state, newLoc+".wood", shared.GetInt(state, newLoc+".wood") + wood)
				PropedSet(state, newLoc+".fire_minutes", shared.GetInt(state, newLoc+".fire_minutes") + fire)

				LogEvent(state, "The player enters the caves with all the supplies they could carry.")
			},
		},
		"sleep": {
			Name: "Sleep 8 hours",
			Conditions: func(state *shared.GameState) bool {
				return isHuman(shared.GetActingPlayer(state)) && SunLightLevel(state) <= 0
			},
			Action: func(state *shared.GameState) {
				ClearEventLogs(state)

				state.Data[PlayerDotKey(state, "sleeping")] = true
				hasFire := shared.GetInt(state, LocalDotKey(state, "fire_minutes")) > 0

				if TimePasses(state, 8 * 60) {
					return
				}

				delete(state.Data, PlayerDotKey(state, "sleeping"))

				message := "The player slept in the cold."
				if hasFire {
					message = "The player slept in warmth."
				}
				LogEvent(state, message)
			},
		},
		"wait": {
			Name: "Wait 1 hour",
			Conditions: func(state *shared.GameState) bool {
				return isHuman(shared.GetActingPlayer(state))
			},
			Action: func(state *shared.GameState) {
				ClearEventLogs(state)
				if TimePasses(state, 60) {
					return
				}
				LogEvent(state, "shared.Player does nothing for 1 hour.")
			},
		},
		"end_of_round": {
			Name: "End of the round",
			Conditions: func(state *shared.GameState) bool {
				return shared.GetActingPlayer(state) == "round"
			},
			Action: func(state *shared.GameState) {
				ClearEventLogs(state)
				PropedSet(state, "round", shared.GetInt(state, "round") + 1)
				EndTurn(state)
				LogEvent(state, "End of the round.")
			},
		},
	}
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

//go:wasmexport PlayableCards
func PlayableCards(statePtr uint32, stateSize uint32) uint64 {
    stateBytes := unsafe.Slice((*byte)(unsafe.Pointer(uintptr(statePtr))), stateSize)
    
    var state shared.GameState
    if err := json.Unmarshal(stateBytes, &state); err != nil {
        return 0
    }
    
    // CLEANUP: We parsed the JSON, the host input data is no longer needed
    delete(activeAllocations, statePtr)

    // Run game logic
    playable := make(map[string]shared.CardProfile)
    deck := InitDeck(&state)
    for key, card := range deck {
        if card.Conditions(&state) {
            playable[key] = shared.CardProfile{Name: card.Name}
        }
    }

    if len(playable) == 0 {
        playable["skip"] = shared.CardProfile{Name: "Skip"}
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

// Cache the robots registry so we don't rebuild the map on every single turn
var localRobots map[string]*shared.Player

//go:wasmexport ChooseCard
func ChooseCard(namePtr uint32, nameSize uint32, statePtr uint32, stateSize uint32) uint64 {
    // 1. Parse Player Name and GameState data
    nameBytes := unsafe.Slice((*byte)(unsafe.Pointer(uintptr(namePtr))), nameSize)
    var playerName string
    if err := json.Unmarshal(nameBytes, &playerName); err != nil {
        return returnRawString("skip")
    }

    stateBytes := unsafe.Slice((*byte)(unsafe.Pointer(uintptr(statePtr))), stateSize)
    var state shared.GameState
    if err := json.Unmarshal(stateBytes, &state); err != nil {
        return returnRawString("skip")
    }

    // CLEANUP: Free host input structures
    delete(activeAllocations, namePtr)
    delete(activeAllocations, statePtr)

    // 2. Instantiate local robots registry if it doesn't exist yet
    if localRobots == nil {
        localRobots = InitRobots()
    }

    // 3. Find the local robot execution logic
    robot, exists := localRobots[playerName]
    if !exists || robot.Decide == nil {
        return returnRawString("skip")
    }

    // 4. Execute decision logic safely INSIDE the WASM context
    key := robot.Decide(&state)
    
    // Check what cards are valid right now
    playable := make(map[string]shared.CardProfile)
    deck := InitDeck(&state)
    for k, card := range deck {
        if card.Conditions(&state) {
            playable[k] = shared.CardProfile{Name: card.Name}
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

//go:wasmexport Return2
func Return2() int32 {
	return 2;
}

func main() {}