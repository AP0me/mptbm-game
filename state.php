<?php
class Card {
    public string $name;
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

    public function play(array &$state) {
        if (!$this->is_playable($state)) { return; }
        ($this->action)($state);
    }
}

function end_turn(array &$state) {
    $state['acting_player'] = ($state['acting_player'] + 1) % count($state['players']);
}

function current_player(array &$state) {
    return $state['players'][$state['acting_player']];
}

$state = [
    'status' => 'RUNNING',
    'round' => 1,
    'players' => ['Anar', 'The Sun', 'Round'],
    'acting_player' => 0,

    'light_level' => 2,
    'health' => 100,
    'coin' => 0,
];

$deck = [
    new Card(
        'skip',
        function() { return true; },
        function(&$state) { end_turn($state); }
    ),
    new Card(
        'sleep',
        function(&$state) {
            return (
                current_player($state) === 'Anar' &&
                $state['light_level'] <= 3
            );
        },
        function(&$state) {
            $state['health'] += 10;
            end_turn($state);
        }
    ),
    new Card(
        'day_cycle',
        function(&$state) {
            return current_player($state) === 'The Sun';
        },
        function(&$state) {
            $state['light_level'] = ($state['light_level'] + 3) % 11;
            end_turn($state);
        }
    ),
    new Card(
        'end_of_round',
        function(&$state) {
            return current_player($state) === 'Round';
        },
        function(&$state) {
            $state['round']++;
            end_turn($state);
        }
    ),
];