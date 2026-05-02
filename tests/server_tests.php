<?php
if (!class_exists('Socket')) { class Socket {} }

require_once './server/state.php';
require_once './server/player.php';
require_once './server/card.php';
require_once './server/network.php';
require_once './client/display.php';

$tests_passed = 0;
$tests_failed = 0;

function it(string $description, bool $condition) {
    global $tests_passed, $tests_failed;
    if ($condition) {
        echo "✅ PASS: $description\n";
        $tests_passed++;
    } else {
        echo "❌ FAIL: $description\n";
        $tests_failed++;
    }
}

function capture_output(callable $fn): string {
    ob_start();
    $fn();
    return ob_get_clean();
}

// --- 1. STATE & PLAYER ROTATION EDGE CASES ---
echo "--- Testing State & Rotation Boundaries ---\n";

$state = init_state();
it("init_state() defaults energy to max_energy", $state['energy'] === $state['max_energy']);

// Edge Case: Next player with only one player
$state['player_order'] = ['anar'];
$state['acting_player'] = 'anar';
it("next_player_key() returns same player if alone", next_player_key($state) === 'anar');

// Edge Case: remove_players when player isn't in list
$original_count = count($state['player_order']);
remove_players($state, ['ghost_player']);
it("remove_players() ignores non-existent keys", count($state['player_order']) === $original_count);

// Edge Case: end_turn rotation
$state['player_order'] = ['p1', 'p2', 'p3'];
$state['acting_player'] = 'p3';
end_turn($state);
it("end_turn() wraps back to index 0", $state['acting_player'] === 'p1');

// --- 2. SURVIVAL LOGIC EDGE CASES ---
echo "\n--- Testing Survival Logic Boundaries ---\n";

$state = init_state();
// Boundary: Exactly 0 energy
add_energy($state, -$state['energy']); 
it("add_energy(0) triggers LOST status immediately", $state['status'] === 'LOST');

// Boundary: Negative energy reset
$state = init_state();
add_energy($state, -500);
it("add_energy() clamps negative values to 0", $state['energy'] === 0);

// Boundary: Food cannot be negative
$state['food'] = 5;
add_food($state, -10);
it("add_food() clamps to 0", $state['food'] === 0);

// Boundary: Time wrap-around
$state = init_state();
$state['minutes_left_per_player'] = 10;
time_passes(20, $state); 
it("time_passes() resets minutes_left to 1440 after turn end", $state['minutes_left_per_player'] === 1440);
it("time_passes() resets clock to 00:00:00 on new turn", str_contains($state['date_time'], '00:00:00'));

// --- 3. ENVIRONMENTAL CALCULATIONS ---
echo "\n--- Testing Environment (Sun & Fire) ---\n";

// Test Midnight
$state['date_time'] = '2026-05-02 00:00:00';
it("sun_light_level() is 0 at midnight", sun_light_level($state) === 0);

// Test Midday (approximate max light)
$state['date_time'] = '2026-05-02 12:00:00';
it("sun_light_level() is > 5 at midday", sun_light_level($state) > 5);

// Test Fire Logic
$state['fire_minutes'] = 0;
$state['date_time'] = '2026-05-02 00:00:00';
it("light_level() is 0 without fire at night", light_level($state) === 0);
$state['fire_minutes'] = 60;
it("light_level() is exactly 4 with fire at night", light_level($state) === 4);

// --- 4. STRING & UTILITY HARDENING ---
echo "\n--- Testing Utility Edge Cases ---\n";

it("snake_case() handles empty string", snake_case("") === "");
it("snake_case() strips special characters", snake_case("Player! @Name") === "player_name");
it("center_text() handles text longer than width", center_text("SuperLongText", 5) === "SuperLongText");
it("mb_strwidth_no_ansi() handles plain text", mb_strwidth_no_ansi("Plain") === 5);

// --- 5. CARD & DECK SYSTEM ---
echo "\n--- Testing Card & Deck Mechanics ---\n";

$deck = init_deck();
$state = init_state();

// Test: Card action return value
$message = $deck['wait']->play($state);
it("Card->play() returns a narrative string", is_string($message));

// Test: is_playable logic
$state['energy'] = 1;
it("'hunt' is not playable with low energy", $deck['hunt']->is_playable($state) === false);

// Test: Playable cards fallback
$state['acting_player'] = 'invalid_player';
$pcards = playable_cards($deck, $state);
it("playable_cards() falls back to 'skip' if no conditions are met", isset($pcards['skip']));

// --- 6. DISPLAY VERIFICATION ---
echo "\n--- Testing Display Formatting ---\n";

$state_out = capture_output(function() use ($state) { print_state($state); });
it("print_state() contains energy value", str_contains($state_out, (string)$state['energy']));

$cards_out = capture_output(function() use ($deck) { print_cards(['wait' => $deck['wait']]); });
it("print_cards() formats key correctly", str_contains($cards_out, "'wait'"));

// Final Summary
summary();