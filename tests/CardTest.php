<?php
require_once './server/card.php';
require_once './server/state.php';
require_once './server/network.php';
require_once './tests/TestCase.php';

class CardTest extends TestCase {

    private array $deck;
    private array $state;

    protected function setUp(): void {
        $this->deck = init_deck();
        $this->state = init_state();
    }

    public function testCardClassAndPlayableFilter() {
        $card = new Card('Test', fn() => true, fn() => "Action Done");
        $this->assertTrue($card->is_playable($this->state));
        $this->assertEquals("Action Done", $card->play($this->state));

        // Test filtering
        $this->state['food'] = 0;
        $playable = playable_cards($this->deck, $this->state);
        $this->assertArrayNotHasKey('eat', $playable);
        $this->assertArrayHasKey('wait', $playable);

        // Test fallback (if no cards playable, returns skip)
        $this->state['acting_player'] = 'nobody';
        $fallback = playable_cards($this->deck, $this->state);
        $this->assertArrayHasKey('skip', $fallback);
    }

    public function testSkipCard() {
        $this->deck['skip']->play($this->state);
        $this->assertEquals('elshad', $this->state['acting_player']);
    }

    public function testEatCard() {
        $this->state['food'] = 20;
        $this->state['anar.energy'] = 50;
        
        // Raw food
        $msg = $this->deck['eat']->play($this->state);
        $this->assertEquals(65, $this->state['anar.energy']);
        $this->assertEquals(5, $this->state['food']);
        $this->assertStringContainsString("raw food", $msg);

        // Cooked food
        $this->state['food'] = 20;
        $this->state['fire_minutes'] = 10;
        $msg = $this->deck['eat']->play($this->state);
        $this->assertEquals(100, $this->state['anar.energy']); // Maxed
        $this->assertStringContainsString("cooked meal", $msg);
    }

    public function testHuntCard() {
        $this->state['date_time'] = '2023-06-21 12:00:00'; // Day
        $msg = $this->deck['hunt']->play($this->state);
        $this->assertEquals(12, $this->state['food']);
        $this->assertStringContainsString("highly productive", $msg);
    }

    public function testWoodCard() {
        $this->state['date_time'] = '2023-06-21 12:00:00'; // Day (Light > 3)
        $msg = $this->deck['wood']->play($this->state);
        $this->assertEquals(5, $this->state['wood']);
        $this->assertEquals(85, $this->state['anar.energy']);
    }

    public function testShelterAndBoatCards() {
        $this->state['wood'] = 300;
        $this->state['date_time'] = '2023-06-21 12:00:00';

        $this->deck['shelter']->play($this->state);
        $this->assertTrue($this->state['shelter']);
        $this->assertEquals(250, $this->state['wood']);

        $this->deck['boat']->play($this->state);
        $this->assertTrue($this->state['boat']);
        $this->assertEquals(200, $this->state['wood']);
    }

    public function testFishAndMapCards() {
        $this->state['boat'] = true;
        $this->state['date_time'] = '2023-06-21 12:00:00';
        $this->state['bottle_map'] = true;

        $msg = $this->deck['fish']->play($this->state);
        // Map found logic check
        $this->assertTrue($this->state['bottle_map']);

        // Follow map transitions location
        $this->state['shelter'] = true;
        $this->deck['follow_the_map']->play($this->state);
        $this->assertEquals('caves', $this->state['location']);
        $this->assertTrue($this->state['forest.shelter']);
        $this->assertArrayNotHasKey('shelter', $this->state);
    }

    public function testFireCard() {
        $this->state['wood'] = 10;
        $msg = $this->deck['fire']->play($this->state);
        $this->assertGreaterThan(0, $this->state['fire_minutes']);
        $this->assertEquals(0, $this->state['wood']);
        $this->assertStringContainsString("rubs sticks", $msg);
    }

    public function testSleepCard() {
        $this->state['date_time'] = '2023-06-21 00:00:00'; // Night
        $this->state['anar.energy'] = 10;
        
        $msg = $this->deck['sleep']->play($this->state);
        $this->assertEquals(60, $this->state['anar.energy']); // +60
        $this->assertStringContainsString("in the cold", $msg);
    }

    public function testWaitAndEndOfRoundCards() {
        $this->deck['wait']->play($this->state);
        $this->assertEquals(90, $this->state['anar.energy']);

        $this->state['acting_player'] = 'round';
        $this->deck['end_of_round']->play($this->state);
        $this->assertEquals(2, $this->state['round']);
        $this->assertEquals('anar', $this->state['acting_player']);
    }
}