import { describe, expect, it } from "@jest/globals";

function capitalize(s: string): string {
  return s.charAt(0).toUpperCase() + s.slice(1);
}

function reverse(s: string): string {
  return s.split("").reverse().join("");
}

describe("string utilities", () => {
  describe("capitalize", () => {
    it("capitalizes first letter", () => {
      expect(capitalize("hello")).toBe("Hello");
    });

    it("handles empty string", () => {
      expect(capitalize("")).toBe("");
    });

    it("handles already capitalized", () => {
      expect(capitalize("Hello")).toBe("Hello");
    });
  });

  describe("reverse", () => {
    it("reverses a string", () => {
      expect(reverse("hello")).toBe("olleh");
    });

    it("handles palindrome", () => {
      expect(reverse("racecar")).toBe("racecar");
    });
  });
});
