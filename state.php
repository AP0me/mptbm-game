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

function sun_light_level(array &$state) {
    $timestamp = $state['date_time']; 
    $lat = 40.4; $long = 49.8;
    $sun_info = date_sun_info($timestamp, $lat, $long);
    
    $sunrise = $sun_info['sunrise'];
    $sunset = $sun_info['sunset'];

    if ($timestamp > $sunrise && $timestamp < $sunset) {
        $day_length = $sunset - $sunrise;
        $progress = ($timestamp - $sunrise) / $day_length;
        $light_level = sin($progress * M_PI);

        return round($light_level * 10);
    } 

    return 0;
}

function light_level(array &$state) {
    return sun_light_level($state);
}

$state = [
    'status' => 'RUNNING',
    'round' => 1,
    'player_order' => ['anar', 'round'],
    'acting_player' => 'anar',

    'date_time' => time(),
    'max_health' => 100,
    'health' => 100,
    'coin' => 0,
];

$deck = [
    'skip' => new Card(
        'Skip turn',
        function(&$state) { return true; },
        function(&$state) { end_turn($state); }
    ),
    'wait' => new Card(
        'Wait 1 hour',
        function(&$state) {
            return acting_player($state) === 'anar';
        },
        function(&$state) {
            $state['date_time'] = strtotime('+1 hour', $state['date_time']);
            end_turn($state);
        }
    ),
    'sleep' => new Card(
        'Sleep',
        function(&$state) {
            return (
                acting_player($state) === 'anar' &&
                light_level($state) <= 3
            );
        },
        function(&$state) {
            add_health($state, $state['max_health']);
            $state['date_time'] = strtotime('+8 hour', $state['date_time']);
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