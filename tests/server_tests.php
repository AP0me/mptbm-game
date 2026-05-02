<?php
require_once 'test_boot.php';
require_once './server/state.php'; // Assuming your functions are in this file

run_test_suite("Core Logic & Utilities", function() {
    // Test snake_case
    assert_equals("player_one", snake_case("PlayerOne"), "Converts CamelCase to snake_case");
    assert_equals("anar", snake_case("ANAR"), "Handles all caps");

    // Test next_player_key
    $state = ['acting_player' => 'anar', 'player_order' => ['anar', 'round']];
    assert_equals('round', next_player_key($state), "Correctly cycles to next player");
    
    $state['acting_player'] = 'round';
    assert_equals('anar', next_player_key($state), "Correctly loops back to first player");
});

run_test_suite("Player & Communication", function() {
    // Create a dummy socket-like object or null for testing
    // In PHP, we can pass a dummy resource if needed, but here we test the brain logic
    
    $fake_socket = socket_create(AF_INET, SOCK_STREAM, SOL_TCP); // A real resource but unused
    
    $received_state = null;
    $test_player = new Player(
        "Tester",
        $fake_socket,
        function($state) use (&$received_state) {
            $received_state = $state;
            return "skip";
        }
    );

    $state = ['status' => 'RUNNING'];
    $choice = $test_player->decide($state);

    assert_equals("skip", $choice, "Player decide() returns the brain's output");
    assert_equals('RUNNING', $received_state['status'], "Player brain receives correct state");
    
    socket_close($fake_socket);
});