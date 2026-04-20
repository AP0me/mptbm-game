<?php

function acting_player(array $state): string {
    return $state['acting_player'];
}

function robot_input($string) {
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
        function(&$state) use(&$deck, &$players) {
            return trim(fgets(STDIN));
        }
    ),
    'round' => new Player (
        'Round',
        function(&$state) use(&$deck, &$players) {
            return robot_input('end_of_round');
        }
    ),
];
