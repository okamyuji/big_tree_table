/// <reference types="vitest/config" />
import { defineConfig } from "vite-plus";
import react from "@vitejs/plugin-react";
import tailwindcss from "@tailwindcss/vite";

// https://vite.dev/config/
export default defineConfig({
  lint: { options: { typeAware: true, typeCheck: true } },
  plugins: [react(), tailwindcss()],
  server: {
    proxy: {
      // Rails dev server runs on 3000 by default.
      // Inside docker compose, Vite talks to the backend service via
      // VITE_PROXY_TARGET=http://backend:3000.
      "/api": process.env.VITE_PROXY_TARGET ?? "http://localhost:3000",
    },
  },
  test: {
    globals: true,
    environment: "jsdom",
    setupFiles: ["./tests/setup.ts"],
    exclude: ["e2e/**", "node_modules/**"],
  },
});
