<?php
require_once 'state.php';

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
                end_turn($state);
                return "Let's see what else is happening.";
            }
        )
    ];
    $playable_cards = array_filter($deck, function ($card) use($state) { return $card->is_playable($state); });
    return count($playable_cards) > 0 ? $playable_cards : $default_card;
}

function send_state(array $state, array $client_sockets) {
    $packet = json_encode([
        'type' => 'STATE',
        'data' => $state
    ]);
    foreach ($client_sockets as $client_socket) {
        socket_write($client_socket, $packet . "\n"); 
    }
}

function send_cards(array $playable_cards, array $client_sockets) {
    $card_name_by_key = [];
    foreach ($playable_cards as $key => $card) {
        $card_name_by_key[$key] = [
            'name' => $card->name,
        ];
    }

    $packet = json_encode([
        'type' => 'CARDS',
        'data' => $card_name_by_key
    ]);
    foreach ($client_sockets as $client_socket) {
        socket_write($client_socket, $packet . "\n"); 
    }
}

function send_message(string $message, array $client_sockets) {
    $packet = json_encode([
        'type' => 'MESSAGE',
        'data' => $message
    ]);
    foreach ($client_sockets as $client_socket) {
        socket_write($client_socket, $packet . "\n"); 
    }
}

function client_socket_list(array $player_list) {
    $client_socket_list = [];
    foreach ($player_list as $player) {
        $client_socket_list[] = $player->client_socket;
    }
    return $client_socket_list;
}

$state = init_state();
$deck = init_deck();
$players = init_players($state);

$client_socket_list = client_socket_list($players);

while ($state["status"] === "RUNNING") {
    $player = $players[acting_player($state)];

    send_state($state, $client_socket_list);
    
    $playable_cards = playable_cards($deck, $state);
    send_cards($playable_cards, [$player->client_socket]);
    
    $card = choose_card($player, $playable_cards, $state);
    
    $message = $card->play($state);

    if (is_string($message)) {
        send_message($message, $client_socket_list);
    }
}

send_state($state, $client_socket_list);
