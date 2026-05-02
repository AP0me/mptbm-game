<?php
require_once 'state.php';
require_once 'network.php';

$state = init_state();
$deck = init_deck();
$players = init_players($state, '0.0.0.0', 8080);

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
