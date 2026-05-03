# 開発者向け情報

本リポジトリへのコントリビューター向けの情報です。

## 前提条件

- Go 1.26.1 以上
- Node.js 20 以上（配信画面のビルドに使用）

## よく使うコマンド

```bash
# セットアップ（linter等のインストール）
make setup

# ビルド
make build

# テスト
make test

# E2Eテスト（Go: CLI レイヤー）
make test-e2e

# 配信画面の単体テスト（Vitest + jsdom + Testing Library）
make test-frontend-unit

# 配信画面の E2E テスト（Playwright + Chromium）
make test-frontend-e2e

# フォーマット
make fmt

# Lint
make lint

# 依存チェック
make depcheck

# ビルド成果物の削除
make clean
```

## 配信画面のローカル開発フロー

`frontend/` 配下の配信画面（Vite + TypeScript）を HMR 付きで開発するためのフローです。`make build-frontend` → Go 再起動のサイクルなしで、TS/CSS の編集が即時にブラウザへ反映されます。

### 前提

- Go のビルドが通る状態であること（`make setup` 済み、`frontend/node_modules` 導入済み）
- VOICEVOX エンジンが起動していること（既定: `http://localhost:50021`、環境変数 `VOX_ENGINE_URL` で上書き可）

### 起動

`make dev` でバックエンドと Vite dev server を並列起動できます。

```bash
make dev
```

個別に起動したい場合は、3 つのターミナルで以下を実行します。

```bash
# ターミナル1: viewer（HTTP サーバー、127.0.0.1:8080 でリッスン）
make dev-viewer

# ターミナル2: Vite dev server（既定で http://localhost:5173）
make dev-frontend

# ターミナル3: watch（キュー監視）
make dev-watch
```

ブラウザで **Vite の URL（例: `http://localhost:5173`）** を開きます。`/events`（SSE）, `/speakers.json`, `/test-clip`, `/clips/*` は `frontend/vite.config.ts` の `server.proxy` 経由で Go バックエンド（`localhost:8080`）に中継されます。

### 動作確認ポイント

- `● 接続中` バッジが表示され続ける（SSE が proxy 経由で届いている）
- 「音声テスト」タブで話者一覧が読み込まれ、テスト再生できる
- `queue` ディレクトリに台本を置くと、配信タブに clip が追加される（再生される）
- `frontend/src/*.ts` / `frontend/src/*.css` を編集すると、リロード無しで反映される

### 仕組みメモ

- `assets.go` の `//go:embed all:frontend/dist` と `frontend/dist/.gitkeep` の組み合わせにより、`frontend/dist` が未ビルドでも Go のビルドが通ります。本番配信では `make build-frontend` が生成する `index.html` / `assets/` が埋め込まれます。
- Vite dev server 自体は SSE のストリーミングを中断しません（Node 組み込みの http-proxy がデフォルトでレスポンスを逐次転送するため、追加設定は不要）。

### フロントエンドの構成

配信画面は React 18 + Tailwind CSS v4（CSS-first 設定）で実装されています。主要ファイルと責務は次のとおりです。

| パス | 役割 |
|------|------|
| `frontend/index.html` | `div#root` をマウントポイントとする最小 HTML。React ルートをマウントする `src/main.tsx` を読み込む |
| `frontend/src/main.tsx` | エントリポイント。`createRoot` で `App` を `StrictMode` 配下にマウント |
| `frontend/src/App.tsx` | 全体の状態管理とレイアウト。各種 `localStorage` 永続化、SSE 接続、話者一覧の取得、音量・タブ・トグル・テスト再生などを集約 |
| `frontend/src/app.css` | `@import "tailwindcss"` と `@theme` による Catppuccin 配色（`--color-ctp-*`）の定義 |
| `frontend/src/components/StatusBadge.tsx` | SSE 接続状態バッジ |
| `frontend/src/components/VolumeControls.tsx` | 音量スライダーと消音トグル |
| `frontend/src/components/Tabs.tsx` | 「配信」「音声テスト」のタブ切替 |
| `frontend/src/components/StreamPanel.tsx` | 配信タブ全体。タイムラインコントロールとタイムラインをまとめる |
| `frontend/src/components/TimelineControls.tsx` | 履歴上限、話者名／スタイル／時刻の表示トグル |
| `frontend/src/components/Timeline.tsx` | 履歴リスト（`<ol>`）とスクロール領域 |
| `frontend/src/components/TimelineItem.tsx` | 個々の clip 行。再生中は背景強調＋`▶` 表示、`scrollIntoView` で可視化 |
| `frontend/src/components/TestPanel.tsx` | 音声テストタブ |
| `frontend/src/components/TestControls.tsx` | 話者セレクタ、テスト再生ボタン、エラー表示 |
| `frontend/src/hooks/useEventSource.ts` | `/events` への `EventSource` 接続と自動再接続、`clip` イベント配信 |
| `frontend/src/hooks/usePlaybackQueue.ts` | `<audio>` 要素を介した再生キュー管理（有効時のみ shift→再生、`ended` で次を再生） |
| `frontend/src/hooks/usePersistedState.ts` | `localStorage` と連動する `useState`。`parse`／`serialize` で型変換 |
| `frontend/src/types/api.ts` | サーバー（`internal/infra/http_stream_player.go`）と対応する JSON 型と型ガード |

状態は原則 `App` で集中管理し、子コンポーネントは props で受け取る。`localStorage` のキーは `usePersistedState` に集約され、既存キー（`vox-actor.stream.*`）と互換です。

## 配信画面のコンポーネント／フック単体テスト（Vitest）

`frontend/src/` 配下のコンポーネント・カスタムフックに対する単体テストです。Vitest + jsdom + Testing Library を使用し、`localStorage` 復元やレンダリング分岐など E2E では踏みにくいロジックを検証します。

### 実行

```bash
make test-frontend-unit
# 直接実行する場合:
cd frontend && npm run test:unit
# ローカル開発時のウォッチモード:
cd frontend && npm run test:unit:watch
```

### テストファイルの配置ルール

テスト対象のソースと同じディレクトリに横並びで配置します（`Foo.tsx` ↔ `Foo.test.tsx`、`useFoo.ts` ↔ `useFoo.test.ts`）。

セットアップは `frontend/test/setup.ts`（`@testing-library/jest-dom` の matcher を有効化）と `frontend/vitest.config.ts` にまとめています。

## 配信画面の E2E テスト（Playwright）

`frontend/test/e2e/` 以下に Playwright + Chromium による配信画面の E2E テストがあります。実バックエンド（`vox-actor viewer`）の代わりに `frontend/test/stub-server.mjs` のスタブを使い、SSE / `/api/status` / `/test-clip` / `/clips/*.wav` を決定的にエミュレートします。

### 実行

```bash
make test-frontend-e2e
# 直接実行する場合:
cd frontend && npm run test:e2e
```

`playwright.config.ts` の `webServer` がスタブ（既定 `127.0.0.1:8080`）と Vite dev server（既定 `127.0.0.1:5173`）を自動起動します。`8080` が既に使われている場合（例: `make dev-viewer` を起動中）は、`VOX_STUB_PORT` と `PLAYWRIGHT_BASE_URL` で別ポートに振り分けてください。

```bash
cd frontend
VOX_STUB_PORT=18080 PLAYWRIGHT_BASE_URL=http://127.0.0.1:15173 \
  npx playwright test --reporter=list
```

`vite.config.ts` の `server.proxy` の転送先は環境変数 `VITE_BACKEND_URL` で上書きできます（既定 `http://localhost:8080`）。`playwright.config.ts` はこれを利用してスタブのポートを Vite に伝えます。

### スタブの制御エンドポイント

テストは `APIRequestContext` でスタブの制御チャネルを叩き、SSE 配信内容や `/api/status` の応答を切り替えます（`frontend/test/e2e/helpers.ts`）。

| メソッド・パス | 用途 |
|---|---|
| `POST /__stub/clip` | SSE で `clip` イベントを配信 |
| `POST /__stub/error` | SSE で `error` イベントを配信 |
| `POST /__stub/disconnect` | 全 SSE 接続を一斉切断（再接続テスト用） |
| `POST /__stub/api-status` | `/api/status` の応答を上書き |
| `POST /__stub/reset` | 内部状態を初期化 |

## アーキテクチャ

レイヤー構成と責務ルールは [layer-rules.md](./layer-rules.md) を参照してください。
