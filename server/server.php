<?php
require_once 'state.php';
require_once 'network.php';

$state = init_state();
$deck = init_deck();
$players = init_players($state);

$client_socket_list = client_socket_list($players);

while ($state["status"] === "RUNNING") {
    $player = $players[acting_player($state)];

    $failed_index = send_state($state, $client_socket_list);
    
    $playable_cards = playable_cards($deck, $state);
    $failed_index = send_cards($playable_cards, [$player->client_socket]);
    handle_disconnect($players, $failed_index);
    
    $card = choose_card($player, $playable_cards, $state);
    
    $message = $card->play($state);

    if (is_string($message)) {
        $failed_index = send_message($message, $client_socket_list);
        handle_disconnect($players, $failed_index);
    }
}

$failed_index = send_state($state, $client_socket_list);
handle_disconnect($players, $failed_index);
