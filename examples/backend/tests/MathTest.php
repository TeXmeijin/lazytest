<?php

declare(strict_types=1);

use PHPUnit\Framework\TestCase;

final class MathTest extends TestCase
{
    public function testAddition(): void
    {
        $this->assertSame(4, 2 + 2);
    }

    public function testSubtraction(): void
    {
        $this->assertSame(0, 2 - 2);
    }

    public function testMultiplication(): void
    {
        $this->assertSame(6, 2 * 3);
    }
}
