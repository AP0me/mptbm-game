<?php

function current_player(array $state): Player {
    $acting_player = $state['acting_player'];
    return $state['players'][$acting_player];
}

function robot($string) {
    echo "$string\n";
    sleep(1);
    return $string;
}

class Player {
    public string $name;
    public Closure $brain;

    public function __construct(string $name, callable $brain) {
        $this->name = $name;
        $this->brain = $brain(...); 
    }

    public function decide(array $state): string {
        return ($this->brain)($state);
    }
}

$players = [
    'anar' => new Player (
        'Anar',
        function(&$state) {
            return trim(fgets(STDIN));
        }
    ),
    'sun' => new Player (
        'The Sun',
        function(&$state) {
            return robot('day_cycle');
        }
    ),
    'round' => new Player (
        'Round',
        function(&$state) {
            return robot('end_of_round');
        }
    ),
];
