<?php
function end_turn(array &$state) {
    $currentIndex = array_search($state['acting_player'], $state['player_order']);
    $nextIndex = ($currentIndex + 1) % count($state['player_order']);
    $state['acting_player'] = $state['player_order'][$nextIndex];
}

$state = [
    'status' => 'RUNNING',
    'round' => 1,
    'players' => $players,
    'player_order' => ['anar', 'sun', 'round'],
    'acting_player' => 'anar',

    'light_level' => 2,
    'health' => 100,
    'coin' => 0,
];

$deck = [
    'skip' => new Card(
        'Skip turn',
        function() { return true; },
        function(&$state) { end_turn($state); }
    ),
    'sleep' => new Card(
        'Sleep',
        function(&$state) {
            return (
                current_player($state) === $state['players']['anar'] &&
                $state['light_level'] <= 3
            );
        },
        function(&$state) {
            $state['health'] += 10;
            end_turn($state);
        }
    ),
    'day_cycle' => new Card(
        'Day cycle',
        function(&$state) {
            return current_player($state) === $state['players']['sun'];
        },
        function(&$state) {
            $state['light_level'] = ($state['light_level'] + 3) % 11;
            end_turn($state);
        }
    ),
    'end_of_round' => new Card(
        'End of the round',
        function(&$state) {
            return current_player($state) === $state['players']['round'];
        },
        function(&$state) {
            $state['round']++;
            end_turn($state);
        }
    ),
];