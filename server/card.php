<?php

class Card {
    public string $name;
    public array $narration;
    public Closure $action;
    public Closure $conditions;

    public function __construct(string $name, callable $conditions, callable $action) {
        $this->name = $name;
        $this->conditions = $conditions(...); 
        $this->action = $action(...); 
    }

    public function is_playable(array &$state) {
        return ($this->conditions)($state);
    }

    public function play(array &$state): string | null {
        if (!$this->is_playable($state)) { return null; }
        return ($this->action)($state);
    }
}

function playable_cards(array $deck, array $state): array {
    $default_card = [
        'skip' => new Card(
            'skip',
            function() { return true; },
            function(&$state) {
                clear_event_logs($state);
                end_turn($state);
                log_event($state, "Let's see what else is happening.");
            }
        ),
    ];
    $playable_cards = array_filter($deck, function ($card) use($state) { return $card->is_playable($state); });
    return count($playable_cards) > 0 ? $playable_cards : $default_card;
}