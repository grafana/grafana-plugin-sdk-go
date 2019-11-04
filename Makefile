GOPATH=$(shell go env GOPATH)

all: build

protobuf:
	bash scripts/protobuf-check.sh
	bash proto/generate.sh

$(GOPATH)/bin/golangci-lint:
	go get -u github.com/golangci/golangci-lint@v1.21.0

$(GOPATH)/bin/revive:
	go get -u github.com/mgechev/revive@88015ccf8e97dec79f401f2628aa199f8fe8cb10

$(GOPATH)/bin/gosec:
	go get -u github.com/securego/gosec/cmd/gosec@b4c76d4234afbdec09cfd5843f3e59f03ef586cf

build:
	go build ./...

test:
	go test ./...

lint: $(GOPATH)/bin/golangci-lint $(GOPATH)/bin/revive $(GOPATH)/bin/gosec
	go vet ./...
	golangci-lint run ./...
	revive -exclude ./vendor/... -formatter stylish -config scripts/go/configs/revive.toml ./...
	gosec -quiet -exclude=G104,G107,G108,G201,G202,G204,G301,G304,G401,G402,G501 -conf=scripts/go/configs/gosec.json ./...

.PHONY: all build protobuf
