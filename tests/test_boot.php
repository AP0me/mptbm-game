<?php
// test_boot.php

$registry = ['passed' => 0, 'failed' => 0];

function assert_equals($expected, $actual, $message = "") {
    global $registry;
    if ($expected === $actual) {
        $registry['passed']++;
        echo "✅ PASS: $message\n";
    } else {
        $registry['failed']++;
        echo "❌ FAIL: $message\n";
        echo "   Expected: " . json_encode($expected) . "\n";
        echo "   Actual:   " . json_encode($actual) . "\n";
    }
}

function run_test_suite(string $name, callable $suite) {
    echo "\n--- Running Suite: $name ---\n";
    $suite();
}

function print_summary() {
    global $registry;
    echo "\n" . str_repeat("=", 30) . "\n";
    echo "TEST SUMMARY\n";
    echo "Passed: " . $registry['passed'] . "\n";
    echo "Failed: " . $registry['failed'] . "\n";
    echo str_repeat("=", 30) . "\n";
    exit($registry['failed'] > 0 ? 1 : 0);
}