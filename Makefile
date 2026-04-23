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

run-stream: build-frontend
	go run . watch --engine-url http://voicevox:50021 --stream --stream-addr 0.0.0.0:8080 --queue

install: build-frontend
	go install .

all: build

.PHONY: all setup build build-frontend clean test test-e2e fmt lint typecheck depcheck run-stream install
