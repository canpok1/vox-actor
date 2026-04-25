BINARY_NAME=vox-actor

setup:
	go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.5.0
	go install github.com/v-standard/go-depcheck/cmd/depcheck@v0.0.2
	cd frontend && npm ci

# node_modules が未生成なら自動的に npm ci する。
# 依存マーカーとして frontend/node_modules ディレクトリを使う。
frontend/node_modules: frontend/package.json frontend/package-lock.json
	cd frontend && npm ci
	@touch frontend/node_modules

build-frontend: frontend/node_modules
	cd frontend && npm run build

build: build-frontend
	go build -o $(BINARY_NAME) .

clean:
	go clean
	rm -f $(BINARY_NAME)
	rm -rf frontend/dist

test: build-frontend
	go test ./...

fmt:
	go fmt ./...

typecheck: frontend/node_modules
	cd frontend && npm run typecheck

lint: build-frontend typecheck
	golangci-lint run ./...

depcheck: build-frontend
	go vet -vettool=$$(which depcheck) ./...

test-e2e: build-frontend
	go test -tags=e2e ./test/e2e/...

# Playwright による配信画面のE2Eテスト。
# `npx playwright install --with-deps chromium` 済みを前提とする（dev container では post-create.sh で導入済み）。
test-frontend-e2e: frontend/node_modules
	cd frontend && npm run test:e2e

run-stream: build-frontend
	go run . watch --engine-url http://voicevox:50021 --stream --stream-addr 0.0.0.0:8080 --queue

# 配信画面のローカル開発用ターゲット。
# dev-backend は frontend/dist を参照せず起動できる前提（assets.go の //go:embed all: と frontend/dist/.gitkeep でビルド可）。
# dev-frontend の Vite dev server (既定 :5173) は vite.config.ts の proxy で /events, /speakers.json, /test-clip, /clips/ を :8080 に中継する。
# 詳細な起動フローは docs/development/contributing.md の「配信画面のローカル開発フロー」を参照。
dev-frontend: frontend/node_modules
	cd frontend && npm run dev

# VOICEVOX エンジン URL は CLI 既定値（http://localhost:50021）または環境変数 VOX_ENGINE_URL に従う。
# dev container 等で voicevox:50021 を使う場合は `VOX_ENGINE_URL=http://voicevox:50021 make dev-backend` のように指定する。
dev-backend:
	go run . watch --stream --stream-addr 127.0.0.1:8080 --queue

dev:
	$(MAKE) -j2 dev-backend dev-frontend

install: build-frontend
	go install .

all: build

.PHONY: all setup build build-frontend clean test test-e2e test-frontend-e2e fmt lint typecheck depcheck run-stream dev dev-frontend dev-backend install
