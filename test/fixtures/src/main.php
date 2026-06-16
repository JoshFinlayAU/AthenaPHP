<?php

require __DIR__ . '/calc.php';

use Athena\Demo\Calc;

$c = new Calc();
echo "add: ", $c->add(20, 22), "\n";
echo "fib: ", $c->fib(10), "\n";
echo "php: ", PHP_MAJOR_VERSION, ".", PHP_MINOR_VERSION, "\n";
echo "athena: ", extension_loaded('athena') ? "loaded" : "missing", "\n";
