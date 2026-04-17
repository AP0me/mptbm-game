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

function add_energy(array &$state, int $add) {
    $state['energy'] += $add;
    if ($state['energy'] <= 0) {
        $state['energy'] = 0;
        $state['player_order'] = array_values(array_diff($state['player_order'], ['anar']));
    }
    $state['energy'] = $state['energy'] > $state['max_energy'] ? $state['max_energy'] : $state['energy'];
}

function add_food(array &$state, int $add) {
    $state['food'] += $add;
    if ($state['food'] < 0) {
        $state['food'] = 0;
    }
}

function date_time_stamp(array $state) {
    return strtotime($state['date_time']);
}

function sun_light_level(array &$state) {
    $timestamp = date_time_stamp($state); 
    $lat = 51.4; $long = 0;
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

function time_passes($minutes, array $state): string {
    return date('Y-m-d H:i:s', strtotime("+$minutes minutes", date_time_stamp($state)));
}

$state = [
    'status' => 'RUNNING',
    'round' => 1,
    'player_order' => ['anar', 'round'],
    'acting_player' => 'anar',

    'date_time' => date('Y-m-d H:i:s'),
    'max_energy' => 10000,
    'energy' => 10000,
    'food' => 0,
];

$deck = [
    'skip' => new Card(
        'Skip turn',
        function(&$state) { return true; },
        function(&$state) { end_turn($state); }
    ),
    'eat' => new Card(
        'Eat food',
        function(&$state) {
            return acting_player($state) === 'anar' && $state['food'] > 0;
        },
        function(&$state) {
            $state['date_time'] = time_passes(30, $state);
            add_energy($state, 5);
            add_food($state, -15);
            end_turn($state);
        }
    ),
    'hunt' => new Card(
        'Hunt game',
        function(&$state) {
            return acting_player($state) === 'anar' && $state['energy'] > 30;
        },
        function(&$state) {
            $light = light_level($state);
            $state['date_time'] = time_passes(4 * 60, $state);
            $yield = ($light >= 4) ? 25 : 5; 
            add_food($state, $yield);
            add_energy($state, -35);
            end_turn($state);
        }
    ),
    'sleep' => new Card(
        'Sleep',
        function(&$state) {
            return (
                acting_player($state) === 'anar' &&
                light_level($state) <= 0
            );
        },
        function(&$state) {
            $state['date_time'] = time_passes(8 * 60, $state);
            if ($state['food'] >= 10) {
                add_energy($state, 60);
                add_food($state, -10);
            } else {
                add_energy($state, 10);
                add_food($state, -$state['food']);
            }
            end_turn($state);
        }
    ),
    'wait' => new Card(
        'Wait 1 hour',
        function(&$state) { return acting_player($state) === 'anar'; },
        function(&$state) {
            $state['date_time'] = time_passes(60, $state);
            add_energy($state, -10);
            add_food($state, -2);
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