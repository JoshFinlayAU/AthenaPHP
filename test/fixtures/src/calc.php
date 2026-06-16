<?php

namespace Athena\Demo;

class Calc
{
    public function add(int $a, int $b): int
    {
        return $a + $b;
    }

    public function fib(int $n): int
    {
        return $n < 2 ? $n : $this->fib($n - 1) + $this->fib($n - 2);
    }
}
