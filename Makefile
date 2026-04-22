BINARY_NAME=vox-actor

setup:
	go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.5.0
	go install github.com/v-standard/go-depcheck/cmd/depcheck@v0.0.2

build:
	go build -o $(BINARY_NAME) main.go

clean:
	go clean
	rm -f $(BINARY_NAME)

test:
	go test ./...

fmt:
	go fmt ./...

lint:
	golangci-lint run ./...

depcheck:
	go vet -vettool=$$(which depcheck) ./...

test-e2e:
	go test -tags=e2e ./test/e2e/...

run-stream:
	go run . watch --engine-url http://voicevox:50021 --stream --stream-addr 0.0.0.0:8080 .vox-actor/queue

install:
	go install .

all: build

.PHONY: all setup build clean test test-e2e fmt lint depcheck run-stream
