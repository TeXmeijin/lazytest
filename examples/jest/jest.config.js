/** @type {import('jest').Config} */
module.exports = {
  testMatch: ["<rootDir>/src/**/*.test.ts"],
  transform: {
    "^.+\\.ts$": "ts-jest",
  },
};
