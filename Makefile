all: build

protobuf:
	bash scripts/protobuf-check.sh
	bash proto/generate.sh

build:
	go build ./...

.PHONY: all build protobuf
