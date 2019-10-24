SRC_DIR=./proto
DST_DIR=./genproto

all: build

# ${DST_DIR}/datasource/datasource.pb.go: ${SRC_DIR}/datasource.proto
# 	protoc -I=${SRC_DIR} --go_out=plugins=grpc:${DST_DIR}/datasource/ ${SRC_DIR}/datasource.proto

#${DST_DIR}/transform/transform.pb.go: ${SRC_DIR}/transform.proto
#	protoc -I=${SRC_DIR} --go_out=plugins=grpc:${DST_DIR}/transform/ ${SRC_DIR}/transform.proto

build-datasource-proto: ${DST_DIR}/datasource/datasource.pb.go
	protoc -I=${SRC_DIR} --go_out=plugins=grpc,paths=source_relative:${DST_DIR}/datasource/ ${SRC_DIR}/datasource.proto 

build-transform-proto: ${DST_DIR}/transform/transform.pb.go
	protoc -I=${SRC_DIR} --go_out=plugins=grpc,paths=source_relative:${DST_DIR}/transform/ ${SRC_DIR}/transform.proto

# https://github.com/golang/protobuf/issues/39#issuecomment-452831378 (Currently must be two commands)
build-proto: build-datasource-proto build-transform-proto

build: build-proto
	go build ./...

.PHONY: all build build-proto
