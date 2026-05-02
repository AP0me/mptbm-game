<?php

function acting_player(array $state): string {
    return $state['acting_player'];
}

function robot_input(string $string) {
    echo "$string\n";
    sleep(1);
    return $string;
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

function init_players(array &$state): array {
    $address = '0.0.0.0';
    $port = 8080;
    $server_socket = socket_create(AF_INET, SOCK_STREAM, SOL_TCP);
    socket_set_option($server_socket, SOL_SOCKET, SO_REUSEADDR, 1);
    socket_bind($server_socket, $address, $port);
    socket_listen($server_socket, 5);
    echo "Server started on $address:$port. Waiting for players...\n";

    $players = [];
    $required_players = 2;

    while (count($players) < $required_players) {
        $client_socket = socket_accept($server_socket);
        if ($client_socket) {
            $name = trim(socket_read($client_socket, 1024));
            
            $players[strtolower($name)] = new Player(
                $name,
                $client_socket,
                function(&$state) use ($client_socket) {
                    return trim(socket_read($client_socket, 1024));
                }
            );
            
            echo "Player $name joined the game.\n";
        }
    }

    $robot_client_sock = socket_create(AF_INET, SOCK_STREAM, SOL_TCP);
    socket_connect($robot_client_sock, $address, $port);
    $robot_server_sock = socket_accept($server_socket);
    socket_write($robot_client_sock, "Round\n");
    $name = trim(socket_read($robot_server_sock, 1024));

    $players['round'] = new Player(
        $name,
        $robot_server_sock,
        function(&$state) use ($robot_client_sock) {
            $input = robot_input('end_of_round');
            socket_write($robot_client_sock, $input . "\n");
            return $input;
        }
    );

    $state['player_order'] = array_keys($players);
    $state['acting_player'] = $state['player_order'][0];

    return $players;
}

function choose_card(Player $player, array $playable_cards, array $state): Card {
    $card_key = $player->decide($state);
    
    if (isset($playable_cards[$card_key])) {
        return $playable_cards[$card_key];
    }

    return $playable_cards['skip'];
}