SRC_DIR=./proto
DST_DIR=./genproto

all: build

scripts/go/bin/golangci-lint: scripts/go/go.mod
	@cd scripts/go; \
	go build -o ./bin/golangci-lint github.com/golangci/golangci-lint/cmd/golangci-lint

${DST_DIR}/datasource/datasource.pb.go: ${SRC_DIR}/datasource.proto
	protoc -I=${SRC_DIR} --go_out=plugins=grpc:${DST_DIR}/datasource/ ${SRC_DIR}/datasource.proto

build-proto: ${DST_DIR}/datasource/datasource.pb.go

build: build-proto
	go build ./...

test: build-proto
	go test ./...

lint: build-proto scripts/go/bin/golangci-lint
	go vet ./...
	./scripts/bin/golangci-lint run ./...

.PHONY: all build build-proto
