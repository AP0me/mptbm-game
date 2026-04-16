<?php
require_once 'player.php';
require_once 'state.php';

class Card {
    public string $name;
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

    public function play(array &$state) {
        if (!$this->is_playable($state)) { return; }
        ($this->action)($state);
    }
}

function playable_cards(array $deck, array $state): array {
    $default_card = [
        'skip' => new Card(
            'skip',
            function() { return true; },
            function(&$state) { end_turn($state); }
        )
    ];
    $playable_cards = array_filter($deck, function ($card) use($state) { return $card->is_playable($state); });
    return count($playable_cards) > 0 ? $playable_cards : $default_card;
}

function print_state(array $state) {
    echo "\n## State\n";
    foreach ($state as $key => $value) {
        $value = json_encode($value);
        echo "- $key: {$value}\n";
    }
}

function choose_card(array $playable_cards, array $state): Card {
    $default_card = new Card(
        'skip',
        function() { return true; },
        function(&$state) { end_turn($state); }
    );
    if (empty($playable_cards)) { return $default_card; }

    echo "## Playable cards\n";
    foreach ($playable_cards as $index => $card) {
        echo "$index | $card->name \n";
    }

    while (1) {
        echo "### Choose a card (number): ";
        $player = current_player($state);
        $card_key = $player->decide($state);
        if (is_string($card_key) && isset($playable_cards[$card_key])) {
            return $playable_cards[$card_key];
        }
        echo "Invalid selection. $card_key\n";
    }

    return $default_card;
}

while ($state["status"] === "RUNNING") {
    print_state($state);
    $playable_cards = playable_cards($deck, $state);
    $card = choose_card($playable_cards, $state);
    $card->play($state);
}
print_state($state);
