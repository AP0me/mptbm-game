<?php

function print_state(array $state) {
    // ANSI clear screen and home
    echo "\e[H\e[J";
    // echo "\033[2J\033[H";

    // Colors (inline for simplicity)
    $bold = "\033[1m";
    $reset = "\033[0m";
    $cyan = "\033[36m";
    $yellow = "\033[33m";
    $green = "\033[32m";
    $blue = "\033[34m";

    // Box width - keep consistent
    $box_width = 60;
    
    echo "{$cyan}{$bold}╔" . str_repeat("═", $box_width) . "╗{$reset}\n";
    echo "{$bold}{$cyan}║" . center_text("~ GAME STATE ~", $box_width) . "║{$reset}\n";
    echo "{$cyan}{$bold}╠" . str_repeat("═", $box_width) . "╣{$reset}\n";

    foreach ($state as $key => $value) {
        $value_str = json_encode($value, JSON_UNESCAPED_SLASHES | JSON_UNESCAPED_UNICODE);
        if (mb_strlen($value_str) > 45) {
            $value_str = mb_substr($value_str, 0, 42) . '...';
        }
        
        $key_part = "{$bold}{$yellow}{$key}{$reset}: ";
        $val_part = "{$green}{$value_str}{$reset}";
        
        // Strip ANSI codes for length calculation
        $clean_text = $key . ': ' . $value_str;
        $available_width = $box_width - 2; // Account for side walls
        $padding = max(0, $available_width - mb_strwidth($clean_text));
        
        echo "{$cyan}{$bold}║{$reset} {$key_part}{$val_part}" . str_repeat(" ", $padding+1) . "{$cyan}{$bold}║{$reset}\n";
    }

    echo "{$cyan}{$bold}╚" . str_repeat("═", $box_width) . "╝{$reset}\n";
}

function print_cards($cards) {
    // Colors
    $bold = "\033[1m";
    $reset = "\033[0m";
    $magenta = "\033[35m";
    $yellow = "\033[33m";
    $cyan = "\033[36m";
    $gray = "\033[90m";

    // Box width for cards
    $box_width = 44;
    
    echo "\n{$magenta}{$bold}╔" . str_repeat("═", $box_width) . "╗{$reset}\n";
    echo "{$bold}{$magenta}║" . center_text("~ PLAYABLE CARDS ~", $box_width) . "║{$reset}\n";
    echo "{$magenta}{$bold}╠" . str_repeat("═", $box_width) . "╣{$reset}\n";

    foreach ($cards as $card_key => $card) {
        // Handle both numeric and string keys
        $idx_str = is_numeric($card_key) ? (string)$card_key : "'{$card_key}'";
        
        // Strip ANSI codes for length calculation
        $clean_idx_length = mb_strlen($idx_str);
        $clean_name_length = mb_strlen($card['name']);
        $total_clean_length = $clean_idx_length + 2 + $clean_name_length; // +2 for ": "
        
        $available_width = $box_width - 3; // Account for side walls and space after text
        $padding = max(0, $available_width - $total_clean_length);
        
        $idx_colored = "{$bold}{$yellow}{$idx_str}{$reset}";
        $name_colored = "{$bold}{$cyan}{$card['name']}{$reset}";
        
        echo "{$magenta}{$bold}║{$reset} {$idx_colored}: {$name_colored}" . str_repeat(" ", $padding+2) . "{$magenta}{$bold}║{$reset}\n";
    }

    echo "{$magenta}{$bold}╚" . str_repeat("═", $box_width) . "╝{$reset}\n\n";
}

function center_text($text, $width) {
    $text_length = mb_strwidth($text);
    if ($text_length >= $width) {
        return $text;
    }
    $padding_left = floor(($width - $text_length) / 2);
    $padding_right = $width - $text_length - $padding_left;
    return str_repeat(" ", $padding_left) . $text . str_repeat(" ", $padding_right);
}

// Helper function to get visible width (strips ANSI codes)
function mb_strwidth_no_ansi($string) {
    // Remove ANSI escape sequences
    $clean_string = preg_replace('/\033\[[0-9;]*m/', '', $string);
    return mb_strwidth($clean_string);
}

function type_write(string $text, int $ms = 30000) {
    $chars = mb_str_split($text);
    foreach ($chars as $char) {
        echo $char;
        usleep($ms); // 30ms per character
    }
    echo "\n";
}

function print_card_response($message) {
    $yellow = "\033[33m";
    $white = "\033[37m";
    $reset = "\033[0m";
    $bold = "\033[1m";

    type_write("{$white}{$bold}» $message{$reset}");
    sleep(1);
}

$address = '127.0.0.1';
$port    = 8080;

// 2. Create and Connect Socket
$socket = socket_create(AF_INET, SOCK_STREAM, SOL_TCP);
if ($socket === false) {
    die("socket_create() failed: " . socket_strerror(socket_last_error()) . "\n");
}

echo "Attempting to connect to $address on port $port...\n";
$result = socket_connect($socket, $address, $port);
if ($result === false) {
    die("socket_connect() failed: " . socket_strerror(socket_last_error($socket)) . "\n");
}

// 3. Handshake: Send Player Name
echo "\033[1;33mWelcome to the Game!\033[0m\n";
echo "Enter your character name: ";
$name = trim(fgets(STDIN));
socket_write($socket, $name . "\n"); 

// 4. Main Event Loop
while (true) {
    /** 
     * PHP_NORMAL_READ reads until it hits \n. 
     * This matches the server's socket_write($packet . "\n") logic.
     */
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
            // Use your fancy type_write function for narrative flavor
            print_card_response($packet['data']);
            break;
    }
}

socket_close($socket);