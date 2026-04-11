import { defineConfig } from "vitest/config";

export default defineConfig({
  test: {
    include: ["src/**/*.test.ts"],
    exclude: [
      "**/*.e2e.test.ts",
      "**/*.live.test.ts",
      "**/*.browser.test.ts",
      "node_modules",
    ],
    environment: "node",
  },
});
