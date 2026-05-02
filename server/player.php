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

function snake_case(string $string): string {
    // 1. Regex to find word boundaries:
    // - Consecutive uppercase letters (acronyms)
    // - Standard CamelCase transitions
    // - Numbers or lowercase word chunks
    $pattern = '/[A-Z]{2,}(?=[A-Z][a-z]+[0-9]*|\b)|[A-Z]?[a-z]+[0-9]*|[A-Z]|[0-9]+/';
    
    preg_match_all($pattern, $string, $matches);
    
    // 2. Flatten the matches, lowercase them, and join with underscores
    $words = array_map('mb_strtolower', $matches[0]);
    
    $snake = implode('_', $words);

    return preg_replace('/[^a-z0-9_]/', '', $snake);
}

function handle_disconnect(array &$players, int $failed_index) { // VERY BAD FUNCTION
    $client_socket_list = client_socket_list($players);
    $disconnected_players = sockets_to_players([$client_socket_list[$failed_index]], $players);

    foreach ($disconnected_players as $disconnected_player) {
        unset($players[$disconnected_player->name]);
    }
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

function init_players(array &$state, string $address, int $port): array | false {
    if (!filter_var($address, FILTER_VALIDATE_IP)) {
        return false;
    }

    $server_socket = socket_create(AF_INET, SOCK_STREAM, SOL_TCP);
    socket_set_option($server_socket, SOL_SOCKET, SO_REUSEADDR, 1);
    socket_bind($server_socket, $address, $port);
    socket_listen($server_socket, 5);
    echo "Server started on $address:$port. Waiting for players...\n";

    $players = [];
    $required_players = 2;

    while (count($players) < $required_players) {
        $client_socket = socket_accept($server_socket);
        if ($client_socket === false) { continue; }

        $name = trim(socket_read($client_socket, 1024));
        $name = snake_case(strtolower($name));
        $players[$name] = new Player(
            $name,
            $client_socket,
            function(&$state) use ($client_socket) {
                return human_input($client_socket);
            }
        );
        
        echo "Player $name joined the game.\n";
    }

    $robot_client_sock = socket_create(AF_INET, SOCK_STREAM, SOL_TCP);
    $GLOBALS['robot_client_sockets'][] = $robot_client_sock;
    socket_connect($robot_client_sock, $address, $port);
    $robot_server_sock = socket_accept($server_socket);

    $players['round'] = new Player(
        'Round',
        $robot_server_sock,
        function(&$state) use ($robot_client_sock) {
            $input = robot_input($robot_client_sock, 'end_of_round');
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