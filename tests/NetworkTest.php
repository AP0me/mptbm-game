<?php
require_once './server/network.php';
require_once './server/state.php';
require_once './server/card.php';
require_once './tests/TestCase.php';

class NetworkTest extends TestCase {

    private $serverSide;
    private $clientSide;

    protected function setUp(): void {
        // Create an invisible, connected socket pair to simulate a network
        socket_create_pair(AF_UNIX, SOCK_STREAM, 0, $sockets);
        $this->serverSide = $sockets[0];
        $this->clientSide = $sockets[1];
    }

    protected function tearDown(): void {
        socket_close($this->serverSide);
        socket_close($this->clientSide);
    }

    public function testActingPlayerAndRobotInput() {
        $this->assertEquals('anar', acting_player(['acting_player' => 'anar']));
        
        $timeStart = microtime(true);
        $result = robot_input('test_action');
        $timeEnd = microtime(true);
        
        $this->assertEquals('test_action', $result);
        $this->assertGreaterThanOrEqual(1.0, $timeEnd - $timeStart); // Confirms sleep(1) ran
    }

    public function testHumanInput() {
        // Simulate a human typing "hunt" and hitting enter
        socket_write($this->clientSide, "hunt\n");
        $result = human_input($this->serverSide);
        
        $this->assertEquals('hunt', $result);
        
        // Verify the prompt was sent back to the client
        $prompt = socket_read($this->clientSide, 1024);
        $this->assertStringContainsString('CHOICE', $prompt);
    }

    public function testSafeSendAndPacketMultiSend() {
        $data = ['message' => 'hello'];
        
        $this->assertTrue(safe_send($this->serverSide, $data));
        $received = trim(socket_read($this->clientSide, 1024));
        $this->assertEquals(json_encode($data), $received);

        // Test multi send
        packet_multi_send([$this->serverSide], $data);
        $receivedMulti = trim(socket_read($this->clientSide, 1024));
        $this->assertEquals(json_encode($data), $receivedMulti);
    }

    public function testSendStateCardsAndMessage() {
        send_state(['wood' => 10], [$this->serverSide]);
        $statePacket = json_decode(trim(socket_read($this->clientSide, 1024)), true);
        $this->assertEquals('STATE', $statePacket['type']);
        $this->assertEquals(10, $statePacket['data']['wood']);

        send_messages(["Testing"], [$this->serverSide]);
        $msgPacket = json_decode(trim(socket_read($this->clientSide, 1024)), true);
        $this->assertEquals('MESSAGE', $msgPacket['type']);
        $this->assertEquals("Testing", $msgPacket['data']);

        send_cards(['skip' => new Card('Skip', fn()=>true, fn()=>null)], [$this->serverSide]);
        $cardPacket = json_decode(trim(socket_read($this->clientSide, 1024)), true);
        $this->assertEquals('CARDS', $cardPacket['type']);
        $this->assertEquals('Skip', $cardPacket['data']['skip']['name']);
    }

    public function testPlayerClassAndSocketMapping() {
        $p1 = new Player('anar', $this->serverSide, fn() => 'wait');
        
        $this->assertEquals('wait', $p1->decide([]));
        
        $sockets = client_socket_list([$p1]);
        $this->assertSame($this->serverSide, $sockets[0]);

        $found = sockets_to_players([$this->serverSide], [$p1]);
        $this->assertSame($p1, $found[0]);
    }

    public function testChooseCard() {
        $p1 = new Player('anar', $this->serverSide, fn() => 'hunt');
        $deck = init_deck();
        $state = init_state();
        
        $playable = ['hunt' => $deck['hunt'], 'skip' => $deck['skip']];
        $card = choose_card($p1, $playable, $state);
        $this->assertEquals('Hunt game', $card->name);

        // Test fallback to skip
        $pBad = new Player('anar', $this->serverSide, fn() => 'fake_card');
        $cardBad = choose_card($pBad, $playable, $state);
        $this->assertEquals('skip', $cardBad->name);
    }

    public function testServerSocket() {
        // Port 0 tells OS to assign a random available port
        $socket = server_socket('127.0.0.1', 0);
        $this->assertNotFalse($socket);
        socket_close($socket);

        // Test invalid IP
        $this->assertFalse(server_socket('999.999.999.999', 8080));
    }
}