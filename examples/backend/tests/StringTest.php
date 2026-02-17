<?php

declare(strict_types=1);

use PHPUnit\Framework\TestCase;

final class StringTest extends TestCase
{
    public function testUpperCase(): void
    {
        $this->assertSame('HELLO', strtoupper('hello'));
    }

    public function testLowerCase(): void
    {
        $this->assertSame('hello', strtolower('HELLO'));
    }

    public function testContains(): void
    {
        $this->assertTrue(str_contains('hello world', 'world'));
    }

    public function testLength(): void
    {
        $this->assertSame(5, strlen('hello'));
    }
}
