<?php

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