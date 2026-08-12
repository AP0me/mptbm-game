package core

// import "game/server/mods/shared"

// func PlayableCards(state shared.GameState) map[string]shared.CardProfile {
// 	playable := make(map[string]shared.CardProfile) // TODO
// 	for key, card := range deck {
// 		if card.Conditions(&state) {
// 			playable[key] = card
// 		}
// 	}

// 	if len(playable) == 0 {
// 		return map[string]shared.CardProfile{"skip": {Name: "Skip"}}
// 	}
// 	return playable
// }

// func ChooseCard(p *shared.Player, playable map[string]*shared.Card, state shared.GameState) *shared.Card {
// 	key := p.Decide(&state) // TODO
// 	if card, ok := playable[key]; ok {
// 		return card
// 	}
// 	return playable["skip"]
// }

