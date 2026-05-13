package main

type Card struct {
	Name       string
	Conditions func(state *GameState) bool
	Action     func(state *GameState)
}

func PlayableCards(deck map[string]*Card, state *GameState) map[string]*Card {
	playable := make(map[string]*Card)
	for key, card := range deck {
		if card.Conditions(state) {
			playable[key] = card
		}
	}

	if len(playable) == 0 {
		return map[string]*Card{"skip": deck["skip"]}
	}
	return playable
}

func ChooseCard(p *Player, playable map[string]*Card, state *GameState) *Card {
	key := p.Decide(state)
	if card, ok := playable[key]; ok {
		return card
	}
	return playable["skip"]
}