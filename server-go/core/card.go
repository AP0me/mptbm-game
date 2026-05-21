package core

import "game/server/mods/shared"

func PlayableCards(deck map[string]*shared.Card, state *shared.GameState) map[string]*shared.Card {
	playable := make(map[string]*shared.Card)
	for key, card := range deck {
		if card.Conditions(state) {
			playable[key] = card
		}
	}

	if len(playable) == 0 {
		return map[string]*shared.Card{"skip": deck["skip"]}
	}
	return playable
}

func ChooseCard(p *shared.Player, playable map[string]*shared.Card, state *shared.GameState) *shared.Card {
	key := p.Decide(state)
	if card, ok := playable[key]; ok {
		return card
	}
	return playable["skip"]
}