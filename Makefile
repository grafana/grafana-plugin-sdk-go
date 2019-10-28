SRC_DIR=./proto
DST_DIR=./genproto

all: build

scripts/go/bin/golangci-lint: scripts/go/go.mod
	@cd scripts/go; \
	go build -o ./bin/golangci-lint github.com/golangci/golangci-lint/cmd/golangci-lint

scripts/go/bin/revive: scripts/go/go.mod
	@cd scripts/go; \
	go build -o ./bin/revive github.com/mgechev/revive

scripts/go/bin/gosec: scripts/go/go.mod
	@cd scripts/go; \
	go build -o ./bin/gosec github.com/securego/gosec/cmd/gosec

${DST_DIR}/datasource/datasource.pb.go: ${SRC_DIR}/datasource.proto
	protoc -I=${SRC_DIR} --go_out=plugins=grpc:${DST_DIR}/datasource/ ${SRC_DIR}/datasource.proto

build-proto: ${DST_DIR}/datasource/datasource.pb.go

build: build-proto
	go build ./...

test: build-proto
	go test ./...

lint: build-proto scripts/go/bin/golangci-lint scripts/go/bin/revive scripts/go/bin/gosec
	go vet ./...
	./scripts/go/bin/golangci-lint run ./...
	./scripts/go/bin/revive -exclude ./vendor/... -formatter stylish -config scripts/go/configs/revive.toml ./...
	./scripts/go/bin/gosec -quiet -exclude=G104,G107,G108,G201,G202,G204,G301,G304,G401,G402,G501 -conf=scripts/go/configs/gosec.json ./...

.PHONY: all build build-proto
