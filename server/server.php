<?php
require_once 'state.php';
require_once 'network.php';

$address = '0.0.0.0';
$port = 8080;
$server_socket = server_socket($address, $port);
$robot_socket = robot_socket($server_socket, $address, $port);

$state = init_state();
$deck = init_deck();

$players = robot_players($robot_socket);
human_players($state, $server_socket, $players);

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
