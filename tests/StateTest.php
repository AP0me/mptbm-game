<?php
require_once './server/state.php';
require_once './server/network.php';
require_once './tests/TestCase.php';


class StateTest extends TestCase {

    private array $baseState;

    protected function setUp(): void {
        $this->baseState = init_state();
    }

    public function testInitStateAndHumanKeys() {
        $keys = human_keys();
        $this->assertEquals(['anar', 'elshad'], $keys);
        
        $state = init_state();
        $this->assertEquals('RUNNING', $state['status']);
        $this->assertEquals(1, $state['round']);
        $this->assertEquals('anar', $state['acting_player']);
        $this->assertEquals(100, $state['anar.energy']);
        $this->assertEquals(100, $state['elshad.max_energy']);
    }

    public function testPlayerOrderManipulation() {
        $state = ['player_order' => ['anar', 'elshad', 'round']];
        
        // test next_player_key
        $state['acting_player'] = 'anar';
        $this->assertEquals('elshad', next_player_key($state));
        $state['acting_player'] = 'round';
        $this->assertEquals('anar', next_player_key($state));
        
        // test missing player fallback
        $state['acting_player'] = 'ghost';
        $this->assertEquals('anar', next_player_key($state));

        // test remove_players
        remove_players($state, ['elshad']);
        $this->assertEquals(['anar', 'round'], $state['player_order']);

        // test add_players
        add_players($state, ['bot']);
        $this->assertEquals(['bot', 'anar', 'round'], $state['player_order']);
    }

    public function testEndTurn() {
        $state = $this->baseState; // acts: anar, order: anar, elshad, round
        end_turn($state);
        $this->assertEquals('elshad', $state['acting_player']);
    }

    public function testPlDotKey() {
        $state = ['acting_player' => 'elshad', 'elshad.health' => 50];
        $this->assertEquals(50, pl_dotkey($state, 'health'));
    }

    public function testResourcesFloorAtZero() {
        $state = [];
        add_food($state, 5);
        $this->assertEquals(5, $state['food']);
        add_food($state, -10);
        $this->assertEquals(0, $state['food']);

        add_wood($state, 5);
        $this->assertEquals(5, $state['wood']);
        add_wood($state, -10);
        $this->assertEquals(0, $state['wood']);
    }

    public function testAddEnergyCapsAndDeaths() {
        $state = $this->baseState;
        
        // Cap test
        $state['anar.energy'] = 90;
        add_energy($state, 50);
        $this->assertEquals(100, $state['anar.energy']);

        // Death test
        add_energy($state, -150);
        $this->assertEquals(0, $state['anar.energy']);
        $this->assertEquals('LOST', $state['status']);
        $this->assertNotContains('anar', $state['player_order']);
    }

    public function testDateTimeAndLightLevels() {
        $state = ['date_time' => '2023-06-21 12:00:00']; // Midday summer
        $timestamp = date_time_stamp($state);
        $this->assertIsInt($timestamp);

        // Sun light test
        $light = sun_light_level($state);
        $this->assertGreaterThan(0, $light);

        // Night time test
        $nightState = ['date_time' => '2023-06-21 00:00:00'];
        $this->assertEquals(0, sun_light_level($nightState));

        // Total light test (Fire adds 4)
        $nightState['fire_minutes'] = 10;
        $this->assertEquals(4, light_level($nightState));
    }

    public function testTimePasses() {
        $state = $this->baseState;
        $state['fire_minutes'] = 120;
        $initialTime = date_time_stamp($state);

        // 60 minutes base (-10 energy)
        time_passes(60, $state);
        $this->assertEquals($initialTime + 3600, date_time_stamp($state));
        $this->assertEquals(60, $state['fire_minutes']);
        $this->assertEquals(90, $state['anar.energy']);

        // Shelter halves energy drain (-5 energy)
        $state['shelter'] = true;
        time_passes(60, $state);
        $this->assertEquals(85, $state['anar.energy']);

        // Sleep divides by 5 (-2 energy)
        $player_key = acting_player($state);
        $state["$player_key.sleeping"] = true;
        unset($state['shelter']);
        time_passes(60, $state);
        $this->assertEquals(83, $state['anar.energy']);
    }

    public function testRobotPlayersInit() {
        socket_create_pair(AF_UNIX, SOCK_STREAM, 0, $sockets);
        $robots = robot_players($sockets[0]);
        $this->assertArrayHasKey('round', $robots);
        $this->assertInstanceOf(Player::class, $robots['round']);
        $this->assertEquals('end_of_round', $robots['round']->decide([]));
        socket_close($sockets[0]);
        socket_close($sockets[1]);
    }
}