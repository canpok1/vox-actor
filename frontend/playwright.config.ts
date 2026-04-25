import { defineConfig, devices } from "@playwright/test";

// 配信画面の E2E テスト設定。
// webServer で「Vite dev server (5173)」と「スタブバックエンド (8080)」を順次起動し、
// テストは Vite dev server を入口にして vite.config.ts の proxy 越しに stub と通信する。
// CI では Chromium のみで実行する。`npx playwright install --with-deps chromium` 済みを前提。
const FRONTEND_BASE_URL =
  process.env.PLAYWRIGHT_BASE_URL ?? "http://127.0.0.1:5173";
const FRONTEND_PORT = new URL(FRONTEND_BASE_URL).port || "5173";
const STUB_PORT = process.env.VOX_STUB_PORT ?? "8080";

export default defineConfig({
  testDir: "./test/e2e",
  fullyParallel: false,
  workers: 1,
  retries: process.env.CI ? 1 : 0,
  reporter: process.env.CI ? [["github"], ["list"]] : "list",
  use: {
    baseURL: FRONTEND_BASE_URL,
    trace: "on-first-retry",
  },
  projects: [
    {
      name: "chromium",
      use: {
        ...devices["Desktop Chrome"],
        // audio.play() を user gesture 無しでも許可する。
        // 配信画面の usePlaybackQueue は SSE 受信トリガーで再生するため、
        // この設定が無いと CI 上で play() が reject されて無限ループに陥る。
        launchOptions: { args: ["--autoplay-policy=no-user-gesture-required"] },
      },
    },
  ],
  webServer: [
    {
      command: `node test/stub-server.mjs`,
      url: `http://127.0.0.1:${STUB_PORT}/api/status`,
      reuseExistingServer: !process.env.CI,
      stdout: "pipe",
      stderr: "pipe",
      env: { VOX_STUB_PORT: STUB_PORT },
    },
    {
      command: `npm run dev -- --host 127.0.0.1 --port ${FRONTEND_PORT} --strictPort`,
      url: FRONTEND_BASE_URL,
      reuseExistingServer: !process.env.CI,
      stdout: "pipe",
      stderr: "pipe",
      // VITE_BACKEND_URL を渡して vite.config.ts の proxy target を stub のポートに向ける。
      env: { VITE_BACKEND_URL: `http://127.0.0.1:${STUB_PORT}` },
    },
  ],
});
