<?php
require_once 'state.php';

class Card {
    public string $name;
    public array $narration;
    public Closure $action;
    public Closure $conditions;

    public function __construct(string $name, callable $conditions, callable $action) {
        $this->name = $name;
        $this->conditions = $conditions(...); 
        $this->action = $action(...); 
    }

    public function is_playable(array &$state) {
        return ($this->conditions)($state);
    }

    function type_write(string $text, int $ms = 30000) {
        $chars = mb_str_split($text);
        foreach ($chars as $char) {
            echo $char;
            usleep($ms); // 30ms per character
        }
        echo "\n";
    }

    public function play(array &$state) {
        if (!$this->is_playable($state)) { return; }
        $message = ($this->action)($state);

        $yellow = "\033[33m";
        $white = "\033[37m";
        $reset = "\033[0m";
        $bold = "\033[1m";

        $this->type_write("{$white}{$bold}» $message{$reset}");
        sleep(1);
    }
}

function playable_cards(array $deck, array $state): array {
    $default_card = [
        'skip' => new Card(
            'skip',
            function() { return true; },
            function(&$state) {
                end_turn($state);
                return "Let's see what else is happening.";
            }
        )
    ];
    $playable_cards = array_filter($deck, function ($card) use($state) { return $card->is_playable($state); });
    return count($playable_cards) > 0 ? $playable_cards : $default_card;
}

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

    echo "{$cyan}{$bold}╚" . str_repeat("═", $box_width) . "╝{$reset}";
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
        $clean_name_length = mb_strlen($card->name);
        $total_clean_length = $clean_idx_length + 2 + $clean_name_length; // +2 for ": "
        
        $available_width = $box_width - 3; // Account for side walls and space after text
        $padding = max(0, $available_width - $total_clean_length);
        
        $idx_colored = "{$bold}{$yellow}{$idx_str}{$reset}";
        $name_colored = "{$bold}{$cyan}{$card->name}{$reset}";
        
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


function choose_card(Player $player, array $playable_cards, array $state): Card {
    $bold = "\033[1m";
    $reset = "\033[0m";
    $yellow = "\033[33m";
    $red = "\033[31m";
    $white = "\033[37m";

    while (1) {
        echo "{$bold}{$yellow}➤ Enter card key (e.g. 0, 'skip') {$white}▸ {$reset}";
        $card_key = $player->decide($state);
        if (is_string($card_key) && isset($playable_cards[$card_key])) {
            return $playable_cards[$card_key];
        }
        echo "\n{$red}❌ Invalid selection: {$bold}'{$card_key}'{$reset}{$red}. Try again!{$reset}\n\n";
    }

    return $playable_cards['skip'];
}

while ($state["status"] === "RUNNING") {
    print_state($state);
    $playable_cards = playable_cards($deck, $state);
    print_cards($playable_cards);
    
    $player = $players[acting_player($state)];
    $card = choose_card($player, $playable_cards, $state);
    $card->play($state);
}
print_state($state);
