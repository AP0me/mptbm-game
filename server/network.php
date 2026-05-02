<?php

function safe_send(Socket $socket, array $data) {
    $packet = json_encode($data) . "\n";
    $result = @socket_write($socket, $packet, strlen($packet));
    
    if ($result === false) {
        $error = socket_last_error($socket);
        if ($error == 104 || $error == 32) {
            echo "[Server] Client disconnected unexpectedly.\n";
        } else {
            echo "[Server] Write error: " . socket_strerror($error) . "\n";
        }
        return false;
    }
    return true;
}

function packet_multi_send(array $client_sockets, array $packet) {
    $failed_index = -1;
    foreach ($client_sockets as $client_socket) {
        $failed_index++;

        if (!safe_send($client_socket, $packet))
            { return $failed_index;}
    }
    
    return $failed_index;
}

function send_state(array $state, array $client_sockets) {
    packet_multi_send($client_sockets, [
        'type' => 'STATE',
        'data' => $state
    ]);
}

function send_cards(array $playable_cards, array $client_sockets) {
    $card_name_by_key = [];
    foreach ($playable_cards as $key => $card) {
        $card_name_by_key[$key] = [
            'name' => $card->name,
        ];
    }

    packet_multi_send($client_sockets, [
        'type' => 'CARDS',
        'data' => $card_name_by_key
    ]);
}

function send_message(string $message, array $client_sockets) {
    packet_multi_send($client_sockets, [
        'type' => 'MESSAGE',
        'data' => $message
    ]);
}

function client_socket_list(array $player_list) {
    $client_socket_list = [];
    foreach ($player_list as $player) {
        $client_socket_list[] = $player->client_socket;
    }
    return $client_socket_list;
}