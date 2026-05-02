<?php

function acting_player(array $state): string {
    return $state['acting_player'];
}

function robot_input(Socket $client_socket, string $string): string {
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

function init_players(array $state, string $address, int $port): array | false {
    if (!filter_var($address, FILTER_VALIDATE_IP)) {
        return false;
    }

    $server_socket = socket_create(AF_INET, SOCK_STREAM, SOL_TCP);
    socket_set_option($server_socket, SOL_SOCKET, SO_REUSEADDR, 1);
    socket_bind($server_socket, $address, $port);
    socket_listen($server_socket, 5);
    echo "Server started on $address:$port. Waiting for players...\n";

    $players = [
        'round' => new Player(
            'round',
            robot_socket($server_socket, $address, $port),
            function(&$state) {
                $robot_client_sock = $this->client_socket;
                $input = robot_input($robot_client_sock, 'end_of_round');
                return $input;
            }
        )
    ];

    $required_players = array_diff($state['player_order'], array_values(array_keys($players)));
    while (count($required_players) > 0) {
        $client_socket = socket_accept($server_socket);
        if ($client_socket === false) { continue; }

        $name = $required_players[0];
        $players[$name] = new Player(
            $name,
            $client_socket,
            function(&$state) use ($client_socket) {
                return human_input($client_socket);
            }
        );
        
        $required_players = array_diff($state['player_order'], array_values(array_keys($players)));
    };

    return $players;
}

function choose_card(Player $player, array $playable_cards, array $state): Card {
    $card_key = $player->decide($state);
    
    if (isset($playable_cards[$card_key])) {
        return $playable_cards[$card_key];
    }

    return $playable_cards['skip'];
}