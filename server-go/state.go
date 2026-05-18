package main

import (
	"math"
	"math/rand"
	"slices"
	"time"
)

type GameState struct {
	Data map[string]interface{}
}

func getInt(state *GameState, key string) int {
	if val, ok := state.Data[key].(int); ok {
		return val
	}
	return 0
}

func getString(state *GameState, key string) string {
	if val, ok := state.Data[key].(string); ok {
		return val
	}
	return ""
}

func getStringList(state *GameState, key string) []string {
	if val, ok := state.Data[key].([]string); ok {
		return val
	}
	return []string{}
}

func getBool(state *GameState, key string) bool {
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

func (s *GameState) PropedSet(key string, val int) {
	if val < 0 {
		val = 0
	}
	s.Data[key] = val
}

func (s *GameState) GetActingPlayer() string {
	return getString(s, "acting_player")
}

func (s *GameState) PlayerDotKey(key string) string {
	p := s.GetActingPlayer()
	return p + "." + key
}

func (s *GameState) LocalDotKey(key string) string {
	loc := getString(s, s.PlayerDotKey("location"))
	return loc + "." + key
}

func (s *GameState) LogEvent(msg string) {
	logs := getStringList(s, "event_logs")
	s.Data["event_logs"] = append(logs, msg)
}

func (s *GameState) EndTurn() {
	order := getStringList(s, "player_order")
	current := s.GetActingPlayer()
	nextIdx := 0
	for i, name := range order {
		if name == current {
			nextIdx = (i + 1) % len(order)
			break
		}
	}
	s.Data["acting_player"] = order[nextIdx]
}

func (s *GameState) RemovePlayers(playerKeys []string) {
	order := getStringList(s, "player_order")

	var newOrder []string
	for _, player := range order {
		if !slices.Contains(playerKeys, player) {
			newOrder = append(newOrder, player)
		}
	}

	s.Data["player_order"] = newOrder
}

func (s *GameState) AddEnergy(amount int, deathMsg string) bool {
	key := s.PlayerDotKey("energy")
	
	current := getInt(s, key)
	max := getInt(s, s.PlayerDotKey("max_energy"))
	
	newVal := current + amount
	if newVal <= 0 {
		s.PropedSet(key, 0)
		s.LogEvent(deathMsg)
		s.RemovePlayers([]string{s.GetActingPlayer()})
		s.EndTurn()
		return true
	}
	
	if newVal > max {
		newVal = max
	}
	s.PropedSet(key, newVal)
	return false
}

func (s *GameState) SunLightLevel() int {
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

func (s *GameState) LightLevel() int {
	sunLightLevel := s.SunLightLevel()

	fire_light_level := 0
	if getInt(s, s.LocalDotKey("fire_minutes")) > 0 {
		fire_light_level = 4
	}

	return sunLightLevel + fire_light_level
}

func (s *GameState) TimePasses(minutes int) bool {
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
				s.PropedSet(fireKey, newFire)
			}
		}
	}
	baseEnergySpent := -10.0 * (float64(minutes) / 60.0)
	if getBool(s, s.PlayerDotKey("sleeping")) {
		baseEnergySpent /= 5.0
	}

	if getBool(s, s.LocalDotKey("shelter")) {
		baseEnergySpent /= 2.0
	}

	if getInt(s, s.LocalDotKey("fire_minutes")) > 0 {
		baseEnergySpent /= 2.0
	}

	finalEnergySpent := int(math.Round(baseEnergySpent))
	return s.AddEnergy(finalEnergySpent, "Player died of hunger.")
}

func (s *GameState) ClearEventLogs() {
	s.Data["event_logs"] = []string{}
}

func InitState() *GameState {
	return &GameState{
		Data: map[string]interface{}{
			"status":        "RUNNING",
			"event_logs":    []string{},
			"invisible_keys": []string{"invisible_keys", "event_logs"},
			"round":         1,
			"player_order":  slices.Concat(humanKeys(), robotKeys()),
			"acting_player": "anar",
			"date_time":     time.Now().Format("2006-01-02 00:00:00"),
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
			Decide: func(s *GameState) string {
				return RobotInput("end_of_round")
			},
		},
	}
}

func InitDeck(s *GameState) map[string]*Card {
	return map[string]*Card{
		"skip": {
			Name: "Skip",
			Conditions: func(state *GameState) bool { return true },
			Action: func(state *GameState) {
				state.ClearEventLogs()
				state.EndTurn()
				state.LogEvent("Let's see what else is happening.")
			},
		},
		"eat": {
			Name: "Eat 10 food",
			Conditions: func(state *GameState) bool {
				return isHuman(state.GetActingPlayer()) && getInt(state, state.LocalDotKey("food")) > 0
			},
			Action: func(state *GameState) {
				state.ClearEventLogs()

				if state.TimePasses(30) {
					return
				}

				cooked := 20
				if getInt(state, state.LocalDotKey("fire_minutes")) > 0 {
					cooked = 50
				}

				if state.AddEnergy(cooked, "Died eating") {
					return
				}

				state.PropedSet(state.LocalDotKey("food"), getInt(state, state.LocalDotKey("food")) - 15)

				message := "The player ate raw food."
				if cooked > 20 {
					message = "The player ate a cooked meal."
				}
				state.LogEvent(message)
			},
		},
		"hunt": {
			Name: "Hunt game",
			Conditions: func(state *GameState) bool {
				return isHuman(state.GetActingPlayer())
			},
			Action: func(state *GameState) {
				state.ClearEventLogs()
				isDay := state.SunLightLevel() > 0
				yield := 3
				if isDay {
					yield = 12
				}
				state.PropedSet(state.LocalDotKey("food"), getInt(state, state.LocalDotKey("food")) + yield)

				if state.AddEnergy(-10, "Died hunting") {
					return
				}
				if state.TimePasses(60) {
					return
				}

				message := "Hunting at night was difficult."
				if isDay {
					message = "The daytime hunt was highly productive."
				}
				state.LogEvent(message)
			},
		},
		"wood": {
			Name: "Collect wood",
			Conditions: func(state *GameState) bool {
				return isHuman(state.GetActingPlayer()) && getInt(state, state.PlayerDotKey("energy")) > 5
			},
			Action: func(state *GameState) {
				state.ClearEventLogs()
				if state.AddEnergy(-5, "Died collecting wood") {
					return
				}

				hasLight := state.LightLevel() > 3
				yield := 3
				if hasLight {
					yield = 5
				}

				state.PropedSet(state.LocalDotKey("wood"), getInt(state, state.LocalDotKey("wood")) + yield)

				if state.TimePasses(60) {
					return
				}

				message := "The player foraged for wood. Lack of visibility made foraging challenging."
				if hasLight {
					message = "The player foraged for wood."
				}
				state.LogEvent(message)
			},
		},
		"shelter": {
			Name: "Build a shelter (50 wood)",
			Conditions: func(state *GameState) bool {
				return isHuman(state.GetActingPlayer()) && getInt(state, state.LocalDotKey("wood")) >= 50
			},
			Action: func(state *GameState) {
				state.ClearEventLogs()

				if state.AddEnergy(-45, "Died building shelter") {
					return
				}
				if state.TimePasses(60) {
					return
				}

				state.Data[state.LocalDotKey("shelter")] = true
				state.PropedSet(state.LocalDotKey("wood"), getInt(state, state.LocalDotKey("wood")) - 50)

				state.LogEvent("The player built a shelter.")
			},
		},
		"boat": {
			Name: "Build a boat (250 wood)",
			Conditions: func(state *GameState) bool {
				return isHuman(state.GetActingPlayer()) && getInt(state, state.LocalDotKey("wood")) >= 250
			},
			Action: func(state *GameState) {
				state.ClearEventLogs()

				if state.AddEnergy(-45, "Died building boat") {
					return
				}
				if state.TimePasses(60) {
					return
				}

				state.Data[state.LocalDotKey("boat")] = true
				state.PropedSet(state.LocalDotKey("wood"), getInt(state, state.LocalDotKey("wood")) - 250)

				state.LogEvent("The player built a boat.")
			},
		},
		"fish": {
			Name: "Go fishing",
			Conditions: func(state *GameState) bool {
				return isHuman(state.GetActingPlayer()) && getBool(state, state.LocalDotKey("boat"))
			},
			Action: func(state *GameState) {
				state.ClearEventLogs()
				lucky := rand.Intn(2) == 1
				yield := 3
				if lucky {
					yield = 12
				}

				state.PropedSet(state.LocalDotKey("food"), getInt(state, state.LocalDotKey("food")) + yield)

				if state.TimePasses(60) {
					return
				}

				if state.SunLightLevel() > 6 && !getBool(state, "bottle_map") {
					state.Data["bottle_map"] = true
					state.LogEvent("The player fishes out a map in a bottle!")
					return
				}

				message := "The fishing trip was unlucky."
				if lucky {
					message = "The player caught a big fish."
				}
				state.LogEvent(message)
			},
		},
		"fire": {
			Name: "Make fire (up to 10 wood)",
			Conditions: func(state *GameState) bool {
				return isHuman(state.GetActingPlayer()) && getInt(state, state.LocalDotKey("wood")) > 0
			},
			Action: func(state *GameState) {
				state.ClearEventLogs()
				fromScratch := false

				if getInt(state, state.LocalDotKey("fire_minutes")) <= 0 {
					fromScratch = true
					if state.AddEnergy(-45, "Died making fire from scratch") {
						return
					}
					if state.TimePasses(60) {
						return
					}
				}

				currentWood := getInt(state, state.LocalDotKey("wood"))
				woodToBurn := currentWood
				if woodToBurn > 10 {
					woodToBurn = 10
				}

				burnMinutes := woodToBurn * 90
				state.PropedSet(state.LocalDotKey("fire_minutes"), getInt(state, state.LocalDotKey("fire_minutes")) + burnMinutes)
				state.PropedSet(state.LocalDotKey("wood"), currentWood - woodToBurn)

				message := "The player stokes the fire with more wood."
				if fromScratch {
					message = "The player rubs sticks together to make fire. It was exhausting and time consuming."
				}
				state.LogEvent(message)
			},
		},
		"follow_the_map": {
			Name: "Follow the map the player fished out.",
			Conditions: func(state *GameState) bool {
				return isHuman(state.GetActingPlayer()) && getBool(state, "bottle_map")
			},
			Action: func(state *GameState) {

				food := getInt(state, state.LocalDotKey("food"))
				wood := getInt(state, state.LocalDotKey("wood"))
				fire := getInt(state, state.LocalDotKey("fire_minutes"))

				state.PropedSet(state.LocalDotKey("food"), food - 10)
				state.PropedSet(state.LocalDotKey("wood"), wood - 10)
				state.PropedSet(state.LocalDotKey("fire_minutes"), fire - 60)

				newLoc := "caves"
				state.Data[state.PlayerDotKey("location")] = newLoc

				state.PropedSet(newLoc+".food", getInt(state, newLoc+".food") + food)
				state.PropedSet(newLoc+".wood", getInt(state, newLoc+".wood") + wood)
				state.PropedSet(newLoc+".fire_minutes", getInt(state, newLoc+".fire_minutes") + fire)

				state.LogEvent("The player enters the caves with all the supplies they could carry.")
			},
		},
		"sleep": {
			Name: "Sleep 8 hours",
			Conditions: func(state *GameState) bool {
				return isHuman(state.GetActingPlayer()) && state.SunLightLevel() <= 0
			},
			Action: func(state *GameState) {
				state.ClearEventLogs()

				state.Data[state.PlayerDotKey("sleeping")] = true
				hasFire := getInt(state, state.LocalDotKey("fire_minutes")) > 0

				if state.TimePasses(8 * 60) {
					return
				}

				delete(state.Data, state.PlayerDotKey("sleeping"))

				message := "The player slept in the cold."
				if hasFire {
					message = "The player slept in warmth."
				}
				state.LogEvent(message)
			},
		},
		"wait": {
			Name: "Wait 1 hour",
			Conditions: func(state *GameState) bool {
				return isHuman(state.GetActingPlayer())
			},
			Action: func(state *GameState) {
				state.ClearEventLogs()
				if state.TimePasses(60) {
					return
				}
				state.LogEvent("Player does nothing for 1 hour.")
			},
		},
		"end_of_round": {
			Name: "End of the round",
			Conditions: func(state *GameState) bool {
				return state.GetActingPlayer() == "round"
			},
			Action: func(state *GameState) {
				state.ClearEventLogs()
				state.PropedSet("round", getInt(state, "round") + 1)
				state.EndTurn()
				state.LogEvent("End of the round.")
			},
		},
	}
}