<?php
require_once 'display.php';

echo "Address: ";
$address = trim(fgets(STDIN));
echo "Port: ";
$port = trim(fgets(STDIN));

$socket = socket_create(AF_INET, SOCK_STREAM, SOL_TCP);
if ($socket === false) {
    die("socket_create() failed: " . socket_strerror(socket_last_error()) . "\n");
}

echo "Attempting to connect to $address on port $port...\n";
$result = socket_connect($socket, $address, $port);
if ($result === false) {
    die("socket_connect() failed: " . socket_strerror(socket_last_error($socket)) . "\n");
}

echo "\033[1;33mWelcome to the Game!\033[0m\n";
echo "Enter your character name: ";
$name = trim(fgets(STDIN));
socket_write($socket, $name . "\n"); 
echo "Accepted. \n";

while (true) {
    /** 
     * PHP_NORMAL_READ reads until it hits \n. 
     * This matches the server's socket_write($packet . "\n") logic.
     */
    echo "Waiting for the other players... \n";
    $buffer = socket_read($socket, 8192, PHP_NORMAL_READ);
    
    if ($buffer === false || $buffer === "") {
        echo "\n\033[1;31mConnection lost to server.\033[0m\n";
        break;
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
            socket_write($socket, $choice . "\n");
            break;
            
        case 'MESSAGE':
            print_card_response($packet['data']);
            break;
    }
}

socket_close($socket);