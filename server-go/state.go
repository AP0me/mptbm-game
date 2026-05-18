package main

import (
	"math"
	"math/rand"
	"time"
)

type GameState struct {
	Data map[string]interface{}
}

func (s *GameState) GetActingPlayer() string {
	return s.Data["acting_player"].(string)
}

func (s *GameState) PlDotKey(key string) interface{} {
	p := s.GetActingPlayer()
	plKey := p + "." + key
	return s.Data[plKey]
}

func (s *GameState) LogEvent(msg string) {
	logs := s.Data["event_logs"].([]string)
	s.Data["event_logs"] = append(logs, msg)
}

func (s *GameState) EndTurn() {
	order := s.Data["player_order"].([]string)
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

func (s *GameState) AddEnergy(amount int, deathMsg string) bool {
	p := s.GetActingPlayer()
	key := p + ".energy"
	maxKey := p + ".max_energy"
	
	current := s.Data[key].(int)
	max := s.Data[maxKey].(int)
	
	newVal := current + amount
	if newVal <= 0 {
		s.Data[key] = 0
		s.LogEvent(deathMsg)
		// Logic to remove player would go here
		s.EndTurn()
		return true
	}
	
	if newVal > max {
		newVal = max
	}
	s.Data[key] = newVal
	return false
}

func (s *GameState) SunLightLevel() int {
	t, _ := time.Parse("2006-01-02 15:04:05", s.Data["date_time"].(string))
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
	p := s.GetActingPlayer()
	locKey := p + ".location"
	loc := s.Data[locKey].(string)

	sunLightLevel := s.SunLightLevel()

	fire_light_level := 0
	if (s.Data[loc + ".fire_minutes"].(int) > 0) {
		fire_light_level = 4;
	}

	return sunLightLevel + fire_light_level
}

func (s *GameState) TimePasses(minutes int) bool {
	t, _ := time.Parse("2006-01-02 15:04:05", s.Data["date_time"].(string))
	newTime := t.Add(time.Duration(minutes) * time.Minute)
	s.Data["date_time"] = newTime.Format("2006-01-02 15:04:05")

	energySpent := int(math.Round(-10.0 * (float64(minutes) / 60.0)))
	return s.AddEnergy(energySpent, "Player died of hunger.")
}

func (s *GameState) ClearEventLogs() {
	s.Data["event_logs"] = []string{}
}

func InitState() *GameState {
	return &GameState{
		Data: map[string]interface{}{
			"status":        "RUNNING",
			"event_logs":    []string{},
			"round":         1,
			"player_order":  []string{"anar", "elshad", "round"},
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
	};
}

// InitDeck initializes and returns the game deck map.
func InitDeck(s *GameState) map[string]*Card {
	// Reusable type-safe helpers to replicate PHP's dynamic array access & null coalescing
	getInt := func(state *GameState, key string) int {
		if val, ok := state.Data[key].(int); ok {
			return val
		}
		return 0
	}

	getString := func(state *GameState, key string) string {
		if val, ok := state.Data[key].(string); ok {
			return val
		}
		return ""
	}

	getBool := func(state *GameState, key string) bool {
		if val, ok := state.Data[key].(bool); ok {
			return val
		}
		return false
	}

	isHuman := func(p string) bool {
		return p == "anar" || p == "elshad"
	}

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
				p := state.GetActingPlayer()
				loc := getString(state, p+".location")
				return isHuman(p) && getInt(state, loc+".food") > 0
			},
			Action: func(state *GameState) {
				state.ClearEventLogs()
				p := state.GetActingPlayer()
				loc := getString(state, p+".location")

				if state.TimePasses(30) {
					return
				}

				cooked := 20
				if getInt(state, loc+".fire_minutes") > 0 {
					cooked = 50
				}

				if state.AddEnergy(cooked, "Died eating") {
					return
				}

				state.Data[loc+".food"] = getInt(state, loc+".food") - 15

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

				p := state.GetActingPlayer()
				loc := getString(state, p+".location")
				state.Data[loc+".food"] = getInt(state, loc+".food") + yield

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
				p := state.GetActingPlayer()
				return isHuman(p) && getInt(state, p+".energy") > 5
			},
			Action: func(state *GameState) {
				state.ClearEventLogs()
				if state.AddEnergy(-5, "Died collecting wood") {
					return
				}

				p := state.GetActingPlayer()
				loc := getString(state, p+".location")

				hasLight := state.LightLevel() > 3
				yield := 3
				if hasLight {
					yield = 5
				}

				state.Data[loc+".wood"] = getInt(state, loc+".wood") + yield

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
				p := state.GetActingPlayer()
				loc := getString(state, p+".location")
				return isHuman(p) && getInt(state, loc+".wood") >= 50
			},
			Action: func(state *GameState) {
				state.ClearEventLogs()
				p := state.GetActingPlayer()
				loc := getString(state, p+".location")

				if state.AddEnergy(-45, "Died building shelter") {
					return
				}
				if state.TimePasses(60) {
					return
				}

				state.Data[loc+".shelter"] = true
				state.Data[loc+".wood"] = getInt(state, loc+".wood") - 50

				state.LogEvent("The player built a shelter.")
			},
		},
		"boat": {
			Name: "Build a boat (250 wood)",
			Conditions: func(state *GameState) bool {
				p := state.GetActingPlayer()
				loc := getString(state, p+".location")
				return isHuman(p) && getInt(state, loc+".wood") >= 250
			},
			Action: func(state *GameState) {
				state.ClearEventLogs()
				p := state.GetActingPlayer()
				loc := getString(state, p+".location")

				if state.AddEnergy(-45, "Died building boat") {
					return
				}
				if state.TimePasses(60) {
					return
				}

				state.Data[loc+".boat"] = true
				state.Data[loc+".wood"] = getInt(state, loc+".wood") - 250

				state.LogEvent("The player built a boat.")
			},
		},
		"fish": {
			Name: "Go fishing",
			Conditions: func(state *GameState) bool {
				p := state.GetActingPlayer()
				loc := getString(state, p+".location")
				return isHuman(p) && getBool(state, loc+".boat")
			},
			Action: func(state *GameState) {
				state.ClearEventLogs()
				lucky := rand.Intn(2) == 1
				yield := 3
				if lucky {
					yield = 12
				}

				p := state.GetActingPlayer()
				loc := getString(state, p+".location")
				state.Data[loc+".food"] = getInt(state, loc+".food") + yield

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
				p := state.GetActingPlayer()
				loc := getString(state, p+".location")
				return isHuman(p) && getInt(state, loc+".wood") > 0
			},
			Action: func(state *GameState) {
				state.ClearEventLogs()
				p := state.GetActingPlayer()
				loc := getString(state, p+".location")
				fromScratch := false

				if getInt(state, loc+".fire_minutes") <= 0 {
					fromScratch = true
					if state.AddEnergy(-45, "Died making fire from scratch") {
						return
					}
					if state.TimePasses(60) {
						return
					}
				}

				currentWood := getInt(state, loc+".wood")
				woodToBurn := currentWood
				if woodToBurn > 10 {
					woodToBurn = 10
				}

				// 60 * 1.5 = 90. Multiplying directly bypasses the need for float rounding math.
				burnMinutes := woodToBurn * 90
				state.Data[loc+".fire_minutes"] = getInt(state, loc+".fire_minutes") + burnMinutes
				state.Data[loc+".wood"] = currentWood - woodToBurn

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
				p := state.GetActingPlayer()
				return isHuman(p) && getBool(state, "bottle_map")
			},
			Action: func(state *GameState) {
				p := state.GetActingPlayer()
				loc := getString(state, p+".location")

				food := getInt(state, loc+".food")
				wood := getInt(state, loc+".wood")
				fire := getInt(state, loc+".fire_minutes")

				state.Data[loc+".food"] = food - 10
				state.Data[loc+".wood"] = wood - 10
				state.Data[loc+".fire_minutes"] = fire - 60

				newLoc := "caves"
				state.Data[p+".location"] = newLoc

				state.Data[newLoc+".food"] = getInt(state, newLoc+".food") + food
				state.Data[newLoc+".wood"] = getInt(state, newLoc+".wood") + wood
				state.Data[newLoc+".fire_minutes"] = getInt(state, newLoc+".fire_minutes") + fire

				state.LogEvent("The player enters the caves with all the supplies they could carry.")
			},
		},
		"sleep": {
			Name: "Sleep 8 hours",
			Conditions: func(state *GameState) bool {
				p := state.GetActingPlayer()
				return isHuman(p) && state.SunLightLevel() <= 0
			},
			Action: func(state *GameState) {
				state.ClearEventLogs()
				p := state.GetActingPlayer()
				loc := getString(state, p+".location")

				state.Data[p+".sleeping"] = true
				hasFire := getInt(state, loc+".fire_minutes") > 0

				if state.TimePasses(8 * 60) {
					return
				}

				energyGain := 60
				if hasFire {
					energyGain = 70
				}

				if state.AddEnergy(energyGain, "Died sleeping") {
					return
				}

				delete(state.Data, p+".sleeping")

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
				state.Data["round"] = getInt(state, "round") + 1
				state.EndTurn()
				state.LogEvent("End of the round.")
			},
		},
	}
}