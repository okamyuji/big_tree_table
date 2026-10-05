import { defineConfig } from "@playwright/test";

export default defineConfig({
  testDir: "./e2e",
  timeout: 30000,
  retries: 1,
  use: {
    baseURL: "http://localhost:5173",
    headless: true,
    screenshot: "only-on-failure",
    // Must match the backend's API_BASIC_AUTH_* when they are set.
    httpCredentials: process.env.API_BASIC_AUTH_USER
      ? {
          username: process.env.API_BASIC_AUTH_USER,
          password: process.env.API_BASIC_AUTH_PASSWORD ?? "",
        }
      : undefined,
  },
  projects: [
    {
      name: "chromium",
      use: { browserName: "chromium" },
    },
  ],
});
