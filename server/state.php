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

function add_energy(array &$state, int $add): bool {
    $player_key = acting_player($state);
    $state["$player_key.energy"] = $state["$player_key.energy"] ?? 0;
    $state["$player_key.energy"] += $add;
    if (pl_dotkey($state, 'energy') <= 0) {
        $state["$player_key.energy"] = 0;
        remove_players($state, ['anar']);
        return true;
    }
    $state["$player_key.energy"] = 
    pl_dotkey($state, 'energy') > $state["$player_key.max_energy"] ?
        $state["$player_key.max_energy"] :
        pl_dotkey($state, 'energy');
    
    return false;
}

function add_food(array &$state, int $add) {
    $loc = pl_dotkey($state, 'location');
    $key = "$loc.food";
    $state[$key] = ($state[$key] ?? 0) + $add;
    if ($state[$key] < 0) {
        $state[$key] = 0;
    }
}

function add_wood(array &$state, int $add) {
    $loc = pl_dotkey($state, 'location');
    $key = "$loc.wood";
    $state[$key] = ($state[$key] ?? 0) + $add;
    if ($state[$key] < 0) {
        $state[$key] = 0;
    }
}

function add_fire(array &$state, int $add) {
    $loc = pl_dotkey($state, 'location');
    $key = "$loc.fire_minutes";
    $state[$key] = ($state[$key] ?? 0) + $add;
    if ($state[$key] < 0) {
        $state[$key] = 0;
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
    $loc = pl_dotkey($state, 'location');
    $sun_light_level = sun_light_level($state);
    $fire_light_level = (($state["$loc.fire_minutes"] ?? 0) > 0) ? 4 : 0;
    return $sun_light_level + $fire_light_level;
}

function time_passes(int $minutes, array &$state) {
    $state['date_time'] = date('Y-m-d H:i:s', strtotime("+$minutes minutes", date_time_stamp($state)));
    
    foreach (human_keys() as $player_key) {
        $player_loc = $state["$player_key.location"];
        if (isset($state["$player_loc.fire_minutes"])) {
            $state["$player_loc.fire_minutes"] = max(0, $state["$player_loc.fire_minutes"] - $minutes);
        }
    }

    $loc = pl_dotkey($state, 'location');
    $energy_spent = round(-10 * ($minutes / 60));
    if ($state['sleeping'] ?? false) {
        $energy_spent = $energy_spent / 5;
    }
    if ($state["$loc.shelter"] ?? false) {
        $energy_spent = $energy_spent / 2;
    }

    return add_energy($state, $energy_spent);
}

function pl_dotkey(array $state, string $bare_key) {
    $player_key = acting_player($state);
    return $state["$player_key.$bare_key"];
}

function human_keys() {
    return ['anar', 'elshad'];
}

function init_state(): array {
    $state = [
        'status' => 'RUNNING',
        'round' => 1,
        'player_order' => array_values(array_merge(human_keys(), ['round'])),
        'acting_player' => human_keys()[0],

        'date_time' => date('Y-m-d 00:00:00'),
    ];

    foreach (human_keys() as $human_key) {
        $state["$human_key.location"] = 'forest';
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
                $loc = pl_dotkey($state, 'location');
                return (
                    in_array(acting_player($state), human_keys()) &&
                    ($state["$loc.food"] ?? 0) > 0
                );
            },
            function(&$state) {
                $loc = pl_dotkey($state, 'location');
                $pdied = time_passes(30, $state);
                if($pdied) return "Player died from hunger.";
                $cooked = ($state["$loc.fire_minutes"] ?? 0) > 0 ? 50 : 20;
                $pdied = add_energy($state, $cooked);
                if($pdied) return "Player died from overeating.";
                add_food($state, -15);
                
                return $cooked > 20 ? "The player ate a cooked meal." : "The player ate raw food.";
            }
        ),
        'hunt' => new Card(
            'Hunt game',
            function(&$state) {
                return in_array(acting_player($state), human_keys());
            },
            function(&$state) {
                $light = sun_light_level($state);
                $is_day = ($light > 0);
                $yield = $is_day ? 12 : 3;
                add_food($state, $yield);
                $pdied = add_energy($state, -10);
                if($pdied) return "Player died from exhaustion.";
                $pdied = time_passes(60, $state);
                if($pdied) return "Player died from hunger.";
                return $is_day ?
                "The daytime hunt was highly productive." :
                "Hunting at night was difficult.";
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
                $pdied = add_energy($state, -5);
                if($pdied) return "Player died from exhaustion.";
                $has_light = light_level($state) > 3;
                add_wood($state, ($has_light ? 5 : 3));
                $pdied = time_passes(60, $state);
                if($pdied) return "Player died from hunger.";

                return $has_light ?
                "The player foraged for wood." :
                "The player foraged for wood. Lack of visibility made foraging challenging.";
            }
        ),
        'shelter' => new Card(
            'Build a shelter (50 wood)',
            function(&$state) {
                $loc = pl_dotkey($state, 'location');
                return (
                    in_array(acting_player($state), human_keys()) &&
                    ($state["$loc.wood"] ?? 0) >= 50
                );
            },
            function(&$state) {
                $loc = pl_dotkey($state, 'location');

                $pdied = add_energy($state, -45);
                if($pdied) return "Player died from exhaustion.";
                $pdied = time_passes(60, $state);
                if($pdied) return "Player died from hunger.";

                $state["$loc.shelter"] = true;
                add_wood($state, -50);

                return "The player built a shelter.";
            }
        ),
        'boat' => new Card(
            'Build a boat (250 wood)',
            function(&$state) {
                $loc = pl_dotkey($state, 'location');
                return (
                    in_array(acting_player($state), human_keys()) &&
                    ($state["$loc.wood"] ?? 0) >= 250
                );
            },
            function(&$state) {
                $loc = pl_dotkey($state, 'location');

                $pdied = add_energy($state, -45);
                if($pdied) return "Player died from exhaustion.";
                $pdied = time_passes(60, $state);
                if($pdied) return "Player died from hunger.";

                $state["$loc.boat"] = true;
                add_wood($state, -250);

                return "The player built a boat.";
            }
        ),
        'fish' => new Card(
            'Go fishing',
            function(&$state) {
                $loc = pl_dotkey($state, 'location');
                return (
                    in_array(acting_player($state), human_keys()) &&
                    ($state["$loc.boat"] ?? false)
                );
            },
            function(&$state) {
                $lucky = rand(0, 1);
                $yield = $lucky ? 12 : 3;
                add_food($state, $yield);
                $pdied = time_passes(60, $state);
                if($pdied) return "Player died from hunger.";
                
                if (sun_light_level($state) > 6 && !isset($state['bottle_map'])) {
                    $state['bottle_map'] = true;
                    return "The player fishes out a map in a bottle!";
                }

                return $lucky ?
                "The player caught a big fish." :
                "The fishing trip was unlucky.";
            }
        ),
        'fire' => new Card(
            'Make fire (up to 10 wood)',
            function(&$state) {
                $loc = pl_dotkey($state, 'location');
                return (
                    in_array(acting_player($state), human_keys()) &&
                    ($state["$loc.wood"] ?? 0) > 0
                );
            },
            function(&$state) {
                $loc = pl_dotkey($state, 'location');
                $from_scratch = false;

                if (($state["$loc.fire_minutes"] ?? 0) <= 0) {
                    $from_scratch = true;
                    $pdied = add_energy($state, -45);
                    if($pdied) return "Player died from exhaustion.";
                    $pdied = time_passes(60, $state);
                    if($pdied) return "Player died from hunger.";
                }

                $current_wood = $state["$loc.wood"] ?? 0;
                $wood_to_burn = min($current_wood, 10);

                add_fire($state, round($wood_to_burn * 60 * 1.5));
                add_wood($state, -$wood_to_burn);

                return $from_scratch ? 
                "The player rubs sticks together to make fire. It was exhausting and time consuming." : 
                "The player stokes the fire with more wood.";
            }
        ),
        'follow_the_map' => new Card(
            'Follow the map the player fished out.',
            function (&$state) {
                return (
                    in_array(acting_player($state), human_keys()) &&
                    ($state['bottle_map'] ?? false)
                );
            },
            function (&$state) {
                $player_key = acting_player($state);
                $loc = pl_dotkey($state, 'location');
                $food = $state["$loc.food"];
                $wood = $state["$loc.wood"];
                $fire = $state["$loc.fire_minutes"];
                add_food($state, -10);
                add_wood($state, -10);
                add_fire($state, -60);

                $state["$player_key.location"] = 'caves';
                add_food($state, $food);
                add_wood($state, $wood);
                add_fire($state, $fire);

                return "The player enters the caves with all the supplies they could carry.";
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
                $loc = pl_dotkey($state, 'location');
                $state['sleeping'] = true;
                $has_fire = ($state["$loc.fire_minutes"] ?? 0) > 0;
                
                $pdied = time_passes(8 * 60, $state);
                if($pdied) return "Player died from hunger.";
                $pdied = add_energy($state, $has_fire ? 70 : 60);
                if($pdied) return "Player died from exhaustion.";
                unset($state['sleeping']);

                return $has_fire ?
                "The player slept in warmth." :
                "The player slept in the cold.";
            }
        ),
        'wait' => new Card(
            'Wait 1 hour',
            function(&$state) { return in_array(acting_player($state), human_keys()); },
            function(&$state) {
                $pdied = time_passes(60, $state);
                if($pdied) return "Player died from hunger.";
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