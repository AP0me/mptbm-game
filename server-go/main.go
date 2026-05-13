package main

import (
	"fmt"
	"net"
)

func main() {
	ln, _ := net.Listen("tcp", ":8080")
	fmt.Println("Server started on :8080")

	state := InitState()
	players := make(map[string]*Player)

	// Robot Player for Round Management
	players["round"] = &Player{
		Name: "round",
		Decide: func(s *GameState) string {
			return RobotInput("end_of_round")
		},
	}

	// Wait for human players
	HumanPlayers(state, ln, players)

	deck := InitDeck(state) // Assume this returns map[string]*Card using logic from PHP init_deck
	
	conns := []net.Conn{}
	for _, p := range players {
		if p.Conn != nil {
			conns = append(conns, p.Conn)
		}
	}

	for state.Data["status"] == "RUNNING" {
		actingName := state.GetActingPlayer()
		player := players[actingName]

		SendState(state, conns)

		playable := PlayableCards(deck, state)
		
		// If human, send them their specific options
		if player.Conn != nil {
			SendCards(playable, player.Conn)
		}

		card := ChooseCard(player, playable, state)
		card.Action(state)

		SendMessages(state.Data["event_logs"].([]string), conns)
		state.Data["event_logs"] = []string{} // Clear logs after sending
	}
}

func InitDeck(s *GameState) map[string]*Card {
	// Re-implementation of the PHP init_deck()
	deck := make(map[string]*Card)

	deck["skip"] = &Card{
		Name: "Skip",
		Conditions: func(state *GameState) bool { return true },
		Action: func(state *GameState) {
			state.EndTurn()
			state.LogEvent("Let's see what else is happening.")
		},
	}

	deck["hunt"] = &Card{
		Name: "Hunt game",
		Conditions: func(state *GameState) bool {
			p := state.GetActingPlayer()
			return p == "anar" || p == "elshad"
		},
		Action: func(state *GameState) {
			isDay := state.SunLightLevel() > 0
			yield := 3
			if isDay { yield = 12 }
			
			loc := state.Data[state.GetActingPlayer()+".location"].(string)
			state.Data[loc+".food"] = state.Data[loc+".food"].(int) + yield
			
			if !state.AddEnergy(-10, "Died hunting") {
				state.TimePasses(60)
				state.LogEvent("Hunting was completed.")
			}
		},
	}
    
    deck["end_of_round"] = &Card{
        Name: "End of round",
        Conditions: func(state *GameState) bool { return state.GetActingPlayer() == "round" },
        Action: func(state *GameState) {
            state.Data["round"] = state.Data["round"].(int) + 1
            state.EndTurn()
            state.LogEvent("End of the round.")
        },
    }

	return deck
}