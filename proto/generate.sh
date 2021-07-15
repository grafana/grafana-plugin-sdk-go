#!/bin/bash

# To compile all protobuf files in this repository, run
# "mage protobuf" at the top-level.

set -eu

SOURCE="${BASH_SOURCE[0]}"

while [ -h "$SOURCE" ] ; do SOURCE="$(readlink "$SOURCE")"; done
DIR="$( cd -P "$( dirname "$SOURCE" )" && pwd )"

cd "$DIR"

protoc -I ./ \
  --go_out=../genproto/pluginv2 \
  --go-grpc_out=../genproto/pluginv2 --go-grpc_opt=require_unimplemented_servers=false \
  backend.proto

protoc -I ./ \
  --go_out=../genproto/server \
  --go-grpc_out=../genproto/server --go-grpc_opt=require_unimplemented_servers=false \
  server.proto
