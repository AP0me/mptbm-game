package vanilla

import (
	"game/server/mods/shared"
	"math"
	"math/rand"
	"net"
	"slices"
	"time"
)

type Player struct {
	Name   string
	Conn   net.Conn
	Decide func(state *shared.GameState) string
}

type Card struct {
	Name       string
	Conditions func(state *shared.GameState) bool
	Action     func(state *shared.GameState)
}

func getInt(state *shared.GameState, key string) int {
	if val, ok := state.Data[key].(int); ok {
		return val
	}
	return 0
}

func getString(state *shared.GameState, key string) string {
	if val, ok := state.Data[key].(string); ok {
		return val
	}
	return ""
}

func getStringList(state *shared.GameState, key string) []string {
	if val, ok := state.Data[key].([]string); ok {
		return val
	}
	return []string{}
}

func getBool(state *shared.GameState, key string) bool {
	if val, ok := state.Data[key].(bool); ok {
		return val
	}
	return false
}

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

func isHuman(p string) bool {
	return slices.Contains(humanKeys(), p)
}

func PropedSet(s *shared.GameState, key string, val int) {
	if val < 0 {
		val = 0
	}
	s.Data[key] = val
}

func GetActingPlayer(s *shared.GameState) string {
	return getString(s, "acting_player")
}

func PlayerDotKey(s *shared.GameState, key string) string {
	p := GetActingPlayer(s)
	return p + "." + key
}

func LocalDotKey(s *shared.GameState, key string) string {
	loc := getString(s, PlayerDotKey(s, "location"))
	return loc + "." + key
}

func LogEvent(s *shared.GameState, msg string) {
	logs := getStringList(s, "event_logs")
	s.Data["event_logs"] = append(logs, msg)
}

func EndTurn(s *shared.GameState) {
	order := getStringList(s, "player_order")
	current := GetActingPlayer(s)
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
	order := getStringList(s, "player_order")

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
	
	current := getInt(s, key)
	max := getInt(s, PlayerDotKey(s, "max_energy"))
	
	newVal := current + amount
	if newVal <= 0 {
		PropedSet(s, key, 0)
		LogEvent(s, deathMsg)
		RemovePlayers(s, []string{GetActingPlayer(s)})
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
	t, _ := time.Parse("2006-01-02 15:04:05", getString(s, "date_time"))
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
	if getInt(s, LocalDotKey(s, "fire_minutes")) > 0 {
		fire_light_level = 4
	}

	return sunLightLevel + fire_light_level
}

func TimePasses(s *shared.GameState, minutes int) bool {
	t, _ := time.Parse("2006-01-02 15:04:05", getString(s, "date_time"))
	newTime := t.Add(time.Duration(minutes) * time.Minute)
	s.Data["date_time"] = newTime.Format("2006-01-02 15:04:05")

	// 2. Decay fires across all human player locations
	humans := []string{"anar", "elshad"}
	for _, pKey := range humans {
		playerLoc := getString(s, pKey+".location")
		if playerLoc != "" {
			fireKey := playerLoc + ".fire_minutes"
			currentFire := getInt(s, fireKey)
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
	if getBool(s, PlayerDotKey(s, "sleeping")) {
		baseEnergySpent /= 5.0
	}

	if getBool(s, LocalDotKey(s, "shelter")) {
		baseEnergySpent /= 2.0
	}

	if getInt(s, LocalDotKey(s, "fire_minutes")) > 0 {
		baseEnergySpent /= 2.0
	}

	finalEnergySpent := int(math.Round(baseEnergySpent))
	return AddEnergy(s, finalEnergySpent, "Player died of hunger.")
}

func ClearEventLogs(s *shared.GameState) {
	s.Data["event_logs"] = []string{}
}

func InitState() *shared.GameState {
	return &shared.GameState{
		Data: map[string]interface{}{
			"status":		"RUNNING",
			"event_logs":	[]string{},
			"invisible_keys": []string{"invisible_keys", "event_logs"},
			"round":		 1,
			"player_order":  slices.Concat(humanKeys(), robotKeys()),
			"acting_player": "anar",
			"date_time":	 time.Now().Format("2006-01-02 00:00:00"),
			"anar.location": "forest",
			"anar.energy":   100,
			"anar.max_energy": 100,
			"elshad.location": "forest",
			"elshad.energy":   100,
			"elshad.max_energy": 100,
			"forest.wood": 0,
			"forest.food": 0,
		},
	}
}

func InitRobots() map[string]*Player {
	return map[string]*Player{
		"round": {
			Name: "round",
			Decide: func(s *shared.GameState) string {
				return RobotInput("end_of_round")
			},
		},
	}
}

func InitDeck(s *shared.GameState) map[string]*Card {
	return map[string]*Card{
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
				return isHuman(GetActingPlayer(state)) && getInt(state, LocalDotKey(state, "food")) > 0
			},
			Action: func(state *shared.GameState) {
				ClearEventLogs(state)

				if TimePasses(state, 30) {
					return
				}

				cooked := 20
				if getInt(state, LocalDotKey(state, "fire_minutes")) > 0 {
					cooked = 50
				}

				if AddEnergy(state, cooked, "Died eating") {
					return
				}

				PropedSet(state, LocalDotKey(state, "food"), getInt(state, LocalDotKey(state, "food")) - 15)

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
				return isHuman(GetActingPlayer(state))
			},
			Action: func(state *shared.GameState) {
				ClearEventLogs(state)
				isDay := SunLightLevel(state) > 0
				yield := 3
				if isDay {
					yield = 12
				}
				PropedSet(state, LocalDotKey(state, "food"), getInt(state, LocalDotKey(state, "food")) + yield)

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
				return isHuman(GetActingPlayer(state)) && getInt(state, PlayerDotKey(state, "energy")) > 5
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

				PropedSet(state, LocalDotKey(state, "wood"), getInt(state, LocalDotKey(state, "wood")) + yield)

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
				return isHuman(GetActingPlayer(state)) && getInt(state, LocalDotKey(state, "wood")) >= 50
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
				PropedSet(state, LocalDotKey(state, "wood"), getInt(state, LocalDotKey(state, "wood")) - 50)

				LogEvent(state, "The player built a shelter.")
			},
		},
		"boat": {
			Name: "Build a boat (250 wood)",
			Conditions: func(state *shared.GameState) bool {
				return isHuman(GetActingPlayer(state)) && getInt(state, LocalDotKey(state, "wood")) >= 250
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
				PropedSet(state, LocalDotKey(state, "wood"), getInt(state, LocalDotKey(state, "wood")) - 250)

				LogEvent(state, "The player built a boat.")
			},
		},
		"fish": {
			Name: "Go fishing",
			Conditions: func(state *shared.GameState) bool {
				return isHuman(GetActingPlayer(state)) && getBool(state, LocalDotKey(state, "boat"))
			},
			Action: func(state *shared.GameState) {
				ClearEventLogs(state)
				lucky := rand.Intn(2) == 1
				yield := 3
				if lucky {
					yield = 12
				}

				PropedSet(state, LocalDotKey(state, "food"), getInt(state, LocalDotKey(state, "food")) + yield)

				if TimePasses(state, 60) {
					return
				}

				if SunLightLevel(state) > 6 && !getBool(state, "bottle_map") {
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
				return isHuman(GetActingPlayer(state)) && getInt(state, LocalDotKey(state, "wood")) > 0
			},
			Action: func(state *shared.GameState) {
				ClearEventLogs(state)
				fromScratch := false

				if getInt(state, LocalDotKey(state, "fire_minutes")) <= 0 {
					fromScratch = true
					if AddEnergy(state, -45, "Died making fire from scratch") {
						return
					}
					if TimePasses(state, 60) {
						return
					}
				}

				currentWood := getInt(state, LocalDotKey(state, "wood"))
				woodToBurn := currentWood
				if woodToBurn > 10 {
					woodToBurn = 10
				}

				burnMinutes := woodToBurn * 90
				PropedSet(state, LocalDotKey(state, "fire_minutes"), getInt(state, LocalDotKey(state, "fire_minutes")) + burnMinutes)
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
				return isHuman(GetActingPlayer(state)) && getBool(state, "bottle_map")
			},
			Action: func(state *shared.GameState) {

				food := getInt(state, LocalDotKey(state, "food"))
				wood := getInt(state, LocalDotKey(state, "wood"))
				fire := getInt(state, LocalDotKey(state, "fire_minutes"))

				PropedSet(state, LocalDotKey(state, "food"), food - 10)
				PropedSet(state, LocalDotKey(state, "wood"), wood - 10)
				PropedSet(state, LocalDotKey(state, "fire_minutes"), fire - 60)

				newLoc := "caves"
				state.Data[PlayerDotKey(state, "location")] = newLoc

				PropedSet(state, newLoc+".food", getInt(state, newLoc+".food") + food)
				PropedSet(state, newLoc+".wood", getInt(state, newLoc+".wood") + wood)
				PropedSet(state, newLoc+".fire_minutes", getInt(state, newLoc+".fire_minutes") + fire)

				LogEvent(state, "The player enters the caves with all the supplies they could carry.")
			},
		},
		"sleep": {
			Name: "Sleep 8 hours",
			Conditions: func(state *shared.GameState) bool {
				return isHuman(GetActingPlayer(state)) && SunLightLevel(state) <= 0
			},
			Action: func(state *shared.GameState) {
				ClearEventLogs(state)

				state.Data[PlayerDotKey(state, "sleeping")] = true
				hasFire := getInt(state, LocalDotKey(state, "fire_minutes")) > 0

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
				return isHuman(GetActingPlayer(state))
			},
			Action: func(state *shared.GameState) {
				ClearEventLogs(state)
				if TimePasses(state, 60) {
					return
				}
				LogEvent(state, "Player does nothing for 1 hour.")
			},
		},
		"end_of_round": {
			Name: "End of the round",
			Conditions: func(state *shared.GameState) bool {
				return GetActingPlayer(state) == "round"
			},
			Action: func(state *shared.GameState) {
				ClearEventLogs(state)
				PropedSet(state, "round", getInt(state, "round") + 1)
				EndTurn(state)
				LogEvent(state, "End of the round.")
			},
		},
	}
}