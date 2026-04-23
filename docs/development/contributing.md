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

# E2Eテスト
make test-e2e

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

個別に起動したい場合は、2 つのターミナルで以下を実行します。

```bash
# ターミナル1: Go バックエンド（ストリーム配信モード、127.0.0.1:8080 でリッスン）
make dev-backend

# ターミナル2: Vite dev server（既定で http://localhost:5173）
make dev-frontend
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

## アーキテクチャ

レイヤー構成と責務ルールは [layer-rules.md](./layer-rules.md) を参照してください。
