import react from "@vitejs/plugin-react";
import { defineConfig } from "vitest/config";

// Vitest 専用の設定。配信画面の実行時設定 (vite.config.ts) と分離し、
// HMR や proxy 等の dev server 設定を持ち込まない。
export default defineConfig({
  plugins: [react()],
  test: {
    environment: "jsdom",
    globals: true,
    setupFiles: ["./test/setup.ts"],
    include: ["src/**/*.{test,spec}.{ts,tsx}"],
  },
});
