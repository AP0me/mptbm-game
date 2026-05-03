<?php
require_once 'player.php';
require_once 'card.php';

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
    $player_key = acting_player($state);
    $state["$player_key.energy"] += $add;
    if (pl_dotkey($state, 'energy') <= 0) {
        $state["$player_key.energy"] = 0;
        remove_players($state, ['anar']);
        $state['status'] = 'LOST';
    }
    $state["$player_key.energy"] = 
    pl_dotkey($state, 'energy') > $state["$player_key.max_energy"] ?
        $state["$player_key.max_energy"] :
        pl_dotkey($state, 'energy');
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
    $sun_light_level = sun_light_level($state);
    $fire_light_level = (($state['fire_minutes'] ?? 0) > 0) ? 4 : 0;
    return $sun_light_level + $fire_light_level;
}

function time_passes(int $minutes, array &$state) {
    $state['date_time'] = date('Y-m-d H:i:s', strtotime("+$minutes minutes", date_time_stamp($state)));
    
    $state['minutes_left_per_player'] = max($state['minutes_left_per_player'] - $minutes, 0);
    if ($state['minutes_left_per_player'] === 0) {
        end_turn($state);
        $state['minutes_left_per_player'] = 60 * 24;
        $state['date_time'] = date('Y-m-d 00:00:00', date_time_stamp($state));
    }

    if (isset($state['fire_minutes'])) {
        $state['fire_minutes'] = max($state['fire_minutes'] - $minutes, 0);
        if ($state['fire_minutes'] === 0) {
            unset($state['fire_minutes']);
        }
    }

    $energy_spent = round(-10 * ($minutes / 60));
    if ($state['sleeping'] ?? false) {
        $energy_spent = round(-2 * ($minutes / 60));
    }
    add_energy($state, $energy_spent);
}

function pl_dotkey(array $state, string $bare_key) {
    $player_key = acting_player($state);
    return $state["$player_key.$bare_key"];
}

function human_keys() {
    return ['anar'];
}

function init_state(): array {
    $state = [
        'status' => 'RUNNING',
        'round' => 1,
        'player_order' => array_values(array_merge(human_keys(), ['round'])),
        'acting_player' => human_keys()[0],

        'date_time' => date('Y-m-d 00:00:00'),
        'minutes_left_per_player' => 60 * 24,
    ];

    foreach (human_keys() as $human_key) {
        $state["$human_key.max_energy"] = 100;
        $state["$human_key.energy"] = 100;
    }

    return $state;
}

function robot_players(Socket $robot_socket) {
    return [
        'round' => new Player(
            'round',
            $robot_socket,
            function() {
                return robot_input('end_of_round');
            }
        )
    ];
}

function init_deck(): array {
    return [
        'skip' => new Card(
            'skip',
            function() { return true; },
            function(&$state) {
                end_turn($state);
                return "Let's see what else is happening.";
            }
        ),
        'eat' => new Card(
            'Eat 10 food',
            function(&$state) {
                return in_array(acting_player($state), human_keys()) && $state['food'] > 0;
            },
            function(&$state) {
                time_passes(30, $state);
                $cooked = ($state['fire_minutes'] ?? 0) > 0 ? 50 : 20;
                add_energy($state, $cooked);
                add_food($state, -15);
                
                return $cooked ? "The player stopped to eat a cooked meal." : "The player stopped to eat raw food.";
            }
        ),
        'hunt' => new Card(
            'Hunt game',
            function(&$state) {
                return in_array(acting_player($state), human_keys()) && pl_dotkey($state, 'energy') > 30;
            },
            function(&$state) {
                $light = sun_light_level($state);
                $is_day = ($light > 0);
                $yield = $is_day ? 12 : 3;
                add_food($state, $yield);
                add_energy($state, -10);
                time_passes(60, $state);
                return $is_day ? "The daytime hunt highly productive." : "Hunting at night was difficult.";
            }
        ),
        'wood' => new Card(
            'Collect wood',
            function(&$state) {
                return (
                    in_array(acting_player($state), human_keys()) &&
                    pl_dotkey($state, 'energy') > 5
                );
            },
            function(&$state) {
                add_energy($state, -5);
                $has_light = light_level($state) > 3;
                $state['wood'] = $state['wood'] ?? 0; 
                $state['wood'] += $has_light ? 5 : 3;
                time_passes(60, $state);

                return $has_light ?
                "The player foraged for wood." :
                "The player foraged for wood. The lack of visibility made it challenging.";
            }
        ),
        'fire' => new Card(
            'Make fire with 10 wood',
            function(&$state) {
                return (
                    in_array(acting_player($state), human_keys()) &&
                    (pl_dotkey($state, 'energy') > 45 || ($state['fire_minutes'] ?? 0) > 0) &&
                    ($state['wood'] ?? 0) > 0
                );
            },
            function(&$state) {
                if (!isset($state['fire_minutes'])) {
                    $state['fire_minutes'] = 0;
                }

                $from_scratch = false;
                if (!($state['fire_minutes'] > 0)) {
                    $from_scratch = true;
                    add_energy($state, -45);
                    time_passes(60, $state);
                }

                $state['fire_minutes'] = $state['fire_minutes'] ?? 0;
                $state['fire_minutes'] += round(($state['wood'] ?? 0) * 60 * 1.5);
                $state['wood'] = max(0, ($state['wood'] ?? 0) - 10);

                return $from_scratch ? 
                "The player rubs sticks together to make fire. It was exhausting and time consuming." : 
                "The player stokes the fire with more wood.";
            }
        ),
        'sleep' => new Card(
            'Sleep 8 hours',
            function(&$state) {
                return (
                    in_array(acting_player($state), human_keys()) &&
                    sun_light_level($state) <= 0
                );
            },
            function(&$state) {
                $state['sleeping'] = true;
                $has_fire = ($state['fire_minutes'] ?? 0) > 0;
                time_passes(8 * 60, $state);
                add_energy($state, $has_fire ? 70 : 60);
                unset($state['sleeping']);

                return $has_fire ? 
                "The player slept for 8 hours in warmth." :
                "The player slept for 8 hours in the cold.";
            }
        ),
        'wait' => new Card(
            'Wait 1 hour',
            function(&$state) { return in_array(acting_player($state), human_keys()); },
            function(&$state) {
                time_passes(60, $state);
                return "Player does nothing for 1 hour.";
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
                return "End of the round.";
            }
        ),
    ];
}