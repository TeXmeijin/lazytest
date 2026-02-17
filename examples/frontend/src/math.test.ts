import { describe, expect, it } from "vitest";

function add(a: number, b: number): number {
  return a + b;
}

function multiply(a: number, b: number): number {
  return a * b;
}

describe("math", () => {
  describe("add", () => {
    it("adds two positive numbers", () => {
      expect(add(2, 3)).toBe(5);
    });

    it("adds negative numbers", () => {
      expect(add(-1, -2)).toBe(-3);
    });

    it("adds zero", () => {
      expect(add(5, 0)).toBe(5);
    });
  });

  describe("multiply", () => {
    it("multiplies two numbers", () => {
      expect(multiply(3, 4)).toBe(12);
    });

    it("multiplies by zero", () => {
      expect(multiply(5, 0)).toBe(0);
    });
  });
});
