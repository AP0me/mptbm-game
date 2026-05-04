<?php
require_once './server/state.php';
require_once './server/card.php';
require_once './server/network.php';
require_once './client/display.php';
require_once './tests/StateTest.php';
require_once './tests/CardTest.php';
require_once './tests/NetworkTest.php';
require_once './tests/DisplayTest.php';

echo "=== STARTING GAME ENGINE TEST SUITE ===\n";

// 4. Instantiate and Run each suite
(new StateTest())->run();
(new CardTest())->run();
(new NetworkTest())->run();
(new DisplayTest())->run();

echo "\n=== TESTING COMPLETE ===\n";