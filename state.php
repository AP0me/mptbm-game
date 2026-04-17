<?php
require_once 'player.php';

function next_player_key(array &$state): string {
    $currentIndex = array_search($state['acting_player'], $state['player_order']);
    if ($currentIndex === false) {
        return $state['player_order'][0];
    }
    return $state['player_order'][($currentIndex + 1) % count($state['player_order'])];
}

function remove_players(array &$state, array $player_keys) {
    $state['player_order'] = array_values(array_diff($state['player_order'], $player_keys));
}

function add_players(array &$state, array $player_keys) {
    $state['player_order'] = array_merge($player_keys, $state['player_order']);
}

function end_turn(array &$state) {
    $state['acting_player'] = next_player_key($state);
}

function add_energy(array &$state, int $add) {
    $state['energy'] += $add;
    if ($state['energy'] <= 0) {
        $state['energy'] = 0;
        remove_players($state, ['anar']);
        $state['status'] = 'LOST';
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

function time_passes($minutes, array &$state) {
    $state['date_time'] = date('Y-m-d H:i:s', strtotime("+$minutes minutes", date_time_stamp($state)));
    
    if (isset($state['fire_minutes'])) {
        $state['fire_minutes'] = max($state['fire_minutes'] - $minutes, 0);
        if ($state['fire_minutes'] === 0) {
            unset($state['fire_minutes']);
        }
    }

    add_energy($state, round(-10 * ($minutes / 60)));
}

$state = [
    'status' => 'RUNNING',
    'round' => 1,
    'player_order' => ['anar', 'round'],
    'acting_player' => 'anar',

    'date_time' => date('Y-m-d H:i:s'),
    'max_energy' => 100,
    'energy' => 100,
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
            add_energy($state, ($state['fire_minutes'] ?? 0) > 0 ? 40 : 20);
            add_food($state, -15);
            time_passes(30, $state);
            end_turn($state);
        }
    ),
    'hunt' => new Card(
        'Hunt game',
        function(&$state) {
            return acting_player($state) === 'anar' && $state['energy'] > 30;
        },
        function(&$state) {
            $light = sun_light_level($state);
            $yield = ($light >= 0) ? 25 : 5; 
            add_food($state, $yield);
            add_energy($state, -10);
            time_passes(2 * 60, $state);
            end_turn($state);
        }
    ),
    'wood' => new Card(
        'Collect wood',
        function(&$state) {
            return (
                acting_player($state) === 'anar' &&
                $state['energy'] > 10
            );
        },
        function(&$state) {
            add_energy($state, -5);
            $state['wood'] += light_level($state) > 3 ? 5 : 3;
            time_passes(60, $state);
            end_turn($state);
        }
    ),
    'fire' => new Card(
        'Make fire',
        function(&$state) {
            return (
                acting_player($state) === 'anar' &&
                ($state['energy'] > 50 || ($state['fire_minutes'] ?? 0) > 0) &&
                ($state['wood'] ?? 0) > 0
            );
        },
        function(&$state) {
            if (!isset($state['fire_minutes'])) {
                $state['fire_minutes'] = 0;
            }
            if (!($state['fire_minutes'] > 0)) {
                add_energy($state, -45);
            }

            $state['fire_minutes'] += ($state['wood'] ?? 0) * 60;
            $state['wood'] = max(0, ($state['wood'] ?? 0) - 10);
            time_passes(60, $state);
            end_turn($state);
        }
    ),
    'sleep' => new Card(
        'Sleep',
        function(&$state) {
            return (
                acting_player($state) === 'anar' &&
                sun_light_level($state) <= 0
            );
        },
        function(&$state) {
            $fire_bonus = ($state['fire_minutes'] ?? 0) > 0 ? 10 : 0;
            add_energy($state, 70 + $fire_bonus);
            time_passes(8 * 60, $state);
            end_turn($state);
        }
    ),
    'wait' => new Card(
        'Wait 1 hour',
        function(&$state) { return acting_player($state) === 'anar'; },
        function(&$state) {
            time_passes(60, $state);
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