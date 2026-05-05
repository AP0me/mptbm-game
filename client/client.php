<?php
require_once 'display.php';

function connect_to_server() {
    echo "--- Server Connection ---\n";
    $example_address = "0.0.0.0";
    echo "Address: (e.g: $example_address)";
    $address = trim(fgets(STDIN));
    $example_port = "8080";
    echo "Port: (e.g: $example_port)";
    $port = trim(fgets(STDIN));

    if (!filter_var($address, FILTER_VALIDATE_IP)) {
        $address = $example_address;
    }
    if (!ctype_digit($port)) {
        $port = $example_port;
    }

    // Create Socket
    $socket = @socket_create(AF_INET, SOCK_STREAM, SOL_TCP);
    if ($socket === false) {
        print_error("Could not create socket: " . socket_strerror(socket_last_error()));
        return connect_to_server();
    }

    echo "Attempting to connect to $address on port $port...\n";

    // Connect to Socket
    $result = @socket_connect($socket, $address, $port);
    if ($result === false) {
        print_error("Connection refused: " . socket_strerror(socket_last_error($socket)));
        return connect_to_server();
    }

    echo "\033[1;32mConnected successfully!\033[0m\n";
    echo "\033[1;33mWelcome to the Game!\033[0m\n";
    echo "Waiting for the other players... \n";

    return $socket;
}

$socket = connect_to_server();

while (true) {
    $buffer = @socket_read($socket, 8192, PHP_NORMAL_READ);
    
    if ($buffer === false || $buffer === "") {
        print_error("Connection lost to server.");
        $socket = connect_to_server();
        continue;
    }

    $packet = json_decode(trim($buffer), true);
    if (!$packet) continue;

    switch ($packet['type']) {
        case 'STATE':
            print_state($packet['data']);
            break;

        case 'CARDS':
            print_cards($packet['data']);
            break;

        case 'CHOICE':
            echo "\033[1;33m➤ Enter card key (e.g. skip, hunt): \033[0m";
            $choice = trim(fgets(STDIN));
            @socket_write($socket, $choice . "\n");
            break;
            
        case 'MESSAGE':
            print_card_response($packet['data']);
            break;
    }
}

socket_close($socket);