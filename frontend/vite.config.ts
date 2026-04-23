import tailwindcss from "@tailwindcss/vite";
import react from "@vitejs/plugin-react";
import { defineConfig } from "vite";

// 配信画面は Go バックエンド (`vox-actor watch --stream`) 提供の SSE / API と
// 同一オリジンで通信する前提なので、dev server からバックエンドへプロキシする。
// バックエンドの既定バインドは `--stream-addr 127.0.0.1:8080`。
const backendTarget = "http://localhost:8080";

export default defineConfig({
  root: ".",
  base: "/",
  plugins: [react(), tailwindcss()],
  build: {
    outDir: "dist",
    emptyOutDir: true,
    assetsDir: "assets",
  },
  server: {
    proxy: {
      // SSE エンドポイント。Vite (http-proxy) はデフォルトでストリーミングを維持するため
      // バッファリング無効化の明示設定は不要。changeOrigin でバックエンド側 Host を合わせる。
      "/events": {
        target: backendTarget,
        changeOrigin: true,
      },
      "/speakers.json": {
        target: backendTarget,
        changeOrigin: true,
      },
      "/test-clip": {
        target: backendTarget,
        changeOrigin: true,
      },
      "/clips": {
        target: backendTarget,
        changeOrigin: true,
      },
    },
  },
});
