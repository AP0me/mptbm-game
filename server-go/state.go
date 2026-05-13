package main

import (
	"math"
	"time"
)

type GameState struct {
	Data map[string]interface{}
}

func (s *GameState) GetActingPlayer() string {
	return s.Data["acting_player"].(string)
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

func (s *GameState) TimePasses(minutes int) bool {
	t, _ := time.Parse("2006-01-02 15:04:05", s.Data["date_time"].(string))
	newTime := t.Add(time.Duration(minutes) * time.Minute)
	s.Data["date_time"] = newTime.Format("2006-01-02 15:04:05")

	energySpent := int(math.Round(-10.0 * (float64(minutes) / 60.0)))
	return s.AddEnergy(energySpent, "Player died of hunger.")
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