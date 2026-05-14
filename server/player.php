<?php

function acting_player(array $state): string {
    return $state['acting_player'];
}

function robot_input(string $string): string {
    sleep(1);
    return $string;
}

function human_input(Socket $client_socket): string {
    $packet = json_encode([
        'type' => 'CHOICE',
        'data' => []
    ]);
    socket_write($client_socket, $packet . "\n"); 
    return trim(socket_read($client_socket, 1024));
}

function robot_socket(Socket $server_socket, string $address, int $port) {
    $robot_client_sock = socket_create(AF_INET, SOCK_STREAM, SOL_TCP);
    $GLOBALS['robot_client_sockets'][] = $robot_client_sock;
    socket_connect($robot_client_sock, $address, $port);
    return socket_accept($server_socket);
}

class Player {
    public string $name;
    public Socket $client_socket;
    public Closure $brain;

    public function __construct(string $name, Socket $client_socket, callable $brain) {
        $this->name = $name;
        $this->client_socket = $client_socket;
        $this->brain = $brain(...); 
    }

    public function decide(array $state): string {
        return ($this->brain)($state);
    }
}

function human_players(array $state, Socket $server_socket, array &$players) {
    $required_players = array_values(array_diff($state['player_order'], array_values(array_keys($players))));
    while (count($required_players) > 0) {
        $client_socket = socket_accept($server_socket);
        if ($client_socket === false) { continue; }

        $name = $required_players[0];
        $players[$name] = new Player(
            $name,
            $client_socket,
            function() use ($client_socket) {
                return human_input($client_socket);
            }
        );
        
        $required_players = array_values(array_diff($state['player_order'], array_values(array_keys($players))));
    };
}

function choose_card(Player $player, array $playable_cards, array $state): Card {
    $card_key = $player->decide($state);
    
    if (isset($playable_cards[$card_key])) {
        return $playable_cards[$card_key];
    }

    return $playable_cards['skip'];
}