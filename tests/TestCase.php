<?php

class TestCase {
    protected int $assertions = 0;
    protected int $failures = 0;
    protected array $errors = [];

    /**
     * This is the "runner" that mimics how PHPUnit executes a class.
     */
    public function run() {
        $className = get_class($this);
        echo "\nRunning tests for $className:\n";

        // Get all methods starting with 'test'
        $methods = get_class_methods($this);
        foreach ($methods as $method) {
            if (str_starts_with($method, 'test')) {
                // PHPUnit's lifecycle: setUp -> test -> tearDown
                $this->setUp();
                
                try {
                    $this->$method();
                    echo "\033[32m.\033[0m"; // Green dot for success
                } catch (Exception $e) {
                    $this->failures++;
                    $this->errors[] = "Failure in $method: " . $e->getMessage();
                    echo "\033[31mF\033[0m"; // Red F for failure
                }

                $this->tearDown();
            }
        }

        echo "\n\nTotal Assertions: {$this->assertions}";
        if ($this->failures > 0) {
            echo "\n\033[31mFAILURES ({$this->failures}):\033[0m\n";
            foreach ($this->errors as $error) {
                echo "- $error\n";
            }
        } else {
            echo "\n\033[32mOK! All tests passed.\033[0m\n";
        }
    }

    // Lifecycle hooks to be overridden by child classes
    protected function setUp(): void {}
    protected function tearDown(): void {}

    // --- Assertions ---

    public function assertEquals($expected, $actual, string $message = '') {
        $this->assertions++;
        if ($expected !== $actual) {
            $msg = $message ?: "Expected " . var_export($expected, true) . ", but got " . var_export($actual, true);
            throw new Exception($msg);
        }
    }

    public function assertTrue($condition, string $message = '') {
        $this->assertions++;
        if ($condition !== true) {
            throw new Exception($message ?: "Failed asserting that condition is true");
        }
    }

    public function assertFalse($condition, string $message = '') {
        $this->assertions++;
        if ($condition !== false) {
            throw new Exception($message ?: "Failed asserting that condition is false");
        }
    }

    public function assertIsInt($actual, string $message = '') {
        $this->assertions++;
        if (!is_int($actual)) {
            throw new Exception($message ?: "Expected integer, got " . gettype($actual));
        }
    }

    public function assertGreaterThan($expected, $actual, string $message = '') {
        $this->assertions++;
        if (!($actual > $expected)) {
            throw new Exception($message ?: "Expected $actual to be greater than $expected");
        }
    }

    public function assertGreaterThanOrEqual($expected, $actual, string $message = '') {
        $this->assertions++;
        if (!($actual >= $expected)) {
            throw new Exception($message ?: "Expected $actual to be >= $expected");
        }
    }

    public function assertArrayHasKey($key, $array, string $message = '') {
        $this->assertions++;
        if (!array_key_exists($key, $array)) {
            throw new Exception($message ?: "Array does not contain key: $key");
        }
    }

    public function assertArrayNotHasKey($key, $array, string $message = '') {
        $this->assertions++;
        if (array_key_exists($key, $array)) {
            throw new Exception($message ?: "Array should NOT contain key: $key");
        }
    }

    public function assertNotContains($needle, $haystack, string $message = '') {
        $this->assertions++;
        if (in_array($needle, $haystack)) {
            throw new Exception($message ?: "Haystack should not contain " . var_export($needle, true));
        }
    }

    public function assertStringContainsString($needle, $haystack, string $message = '') {
        $this->assertions++;
        if (strpos($haystack, $needle) === false) {
            throw new Exception($message ?: "String '$haystack' does not contain '$needle'");
        }
    }

    public function assertInstanceOf($expectedClass, $actual, string $message = '') {
        $this->assertions++;
        if (!($actual instanceof $expectedClass)) {
            throw new Exception($message ?: "Expected instance of $expectedClass, got " . get_class($actual));
        }
    }

    public function assertNotFalse($actual, string $message = '') {
        $this->assertions++;
        if ($actual === false) {
            throw new Exception($message ?: "Expected value not to be false");
        }
    }

    public function assertSame($expected, $actual, string $message = '') {
        // For objects/resources, assertSame checks if they are the exact same instance
        $this->assertions++;
        if ($expected !== $actual) {
            throw new Exception($message ?: "Values are not strictly identical");
        }
    }
}