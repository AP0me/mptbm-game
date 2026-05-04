<?php
require_once './client/display.php';
require_once './tests/TestCase.php';

class DisplayTest extends TestCase {

    public function testPrintError() {
        ob_start();
        print_error("Test Error");
        $output = ob_get_clean();
        
        $this->assertStringContainsString("[ERROR]: Test Error", $output);
        $this->assertStringContainsString("\033[1;31m", $output); // Red color check
    }

    public function testCenterText() {
        $this->assertEquals("  foo   ", center_text("foo", 8));
        $this->assertEquals("toolong", center_text("toolong", 5));
    }

    public function testMbStrwidthNoAnsi() {
        $text = "\033[1;31mHello\033[0m";
        // With ANSI, length is 14. Stripped, it should be 5.
        $this->assertEquals(5, mb_strwidth_no_ansi($text));
    }

    public function testPrintState() {
        ob_start();
        print_state(['wood' => 150, 'status' => 'RUNNING']);
        $output = ob_get_clean();

        $this->assertStringContainsString('~ GAME STATE ~', $output);
        $this->assertStringContainsString('wood', $output);
        $this->assertStringContainsString('150', $output);
        $this->assertStringContainsString('RUNNING', $output);
        $this->assertStringContainsString('╔', $output); // Borders rendered
    }

    public function testPrintCards() {
        ob_start();
        print_cards(['hunt' => ['name' => 'Hunt for food']]);
        $output = ob_get_clean();

        $this->assertStringContainsString('~ PLAYABLE CARDS ~', $output);
        $this->assertStringContainsString('hunt', $output);
        $this->assertStringContainsString('Hunt for food', $output);
    }

    public function testTypeWrite() {
        ob_start();
        type_write("AB", 100); // 100 microseconds is practically instant for testing
        $output = ob_get_clean();

        $this->assertEquals("AB\n", $output);
    }

    public function testPrintCardResponse() {
        ob_start();
        print_card_response("Success!");
        $output = ob_get_clean();

        $this->assertStringContainsString("» Success!", $output);
        $this->assertStringContainsString("\033[37m", $output); // White color
    }
}