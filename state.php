<?php
require_once 'player.php';

function next_player_key(array &$state): string {
    $currentIndex = array_search($state['acting_player'], $state['player_order']);
    if ($currentIndex === false) {
        return $state['player_order'][0];
    }
    return $state['player_order'][($currentIndex + 1) % count($state['player_order'])];
}

function end_turn(array &$state) {
    $state['acting_player'] = next_player_key($state);
}

function add_health(array &$state, int $add) {
    $state['health'] += $add;
    if ($state['health'] <= 0) {
        $state['health'] = 0;
        $state['player_order'] = array_values(array_diff($state['player_order'], ['anar']));
    }
    $state['health'] = $state['health'] > $state['max_health'] ? $state['max_health'] : $state['health'];
}

$state = [
    'status' => 'RUNNING',
    'round' => 1,
    'player_order' => ['anar', 'sun', 'round'],
    'acting_player' => 'anar',

    'light_level' => 2,
    'max_health' => 100,
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
                acting_player($state) === 'anar' &&
                $state['light_level'] <= 3
            );
        },
        function(&$state) {
            add_health($state, -60);
            end_turn($state);
        }
    ),
    'day_cycle' => new Card(
        'Day cycle',
        function(&$state) {
            return acting_player($state) === 'sun';
        },
        function(&$state) {
            $state['light_level'] = ($state['light_level'] + 3) % 11;
            end_turn($state);
        }
    ),
    'end_of_round' => new Card(
        'End of the round',
        function(&$state) {
            return acting_player($state) === 'round';
        },
        function(&$state) {
            $state['round']++;
            end_turn($state);
        }
    ),
];