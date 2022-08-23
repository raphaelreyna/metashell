#!/bin/bash
SRC_DIR=internal/rpc/proto
OUT_DIR=internal/rpc/go
FILES=$(find internal/rpc/proto -iname "*.proto")

Go() {
    rm -rf $OUT_DIR 2>&1 > /dev/null
	mkdir $OUT_DIR
	protoc \
		-I=$SRC_DIR \
		--proto_path=$SRC_DIR \
		--go_opt=paths=source_relative \
		--go_out=$OUT_DIR \
		--go-grpc_opt=paths=source_relative \
		--go-grpc_out=$OUT_DIR \
		$FILES 2>&1 > /dev/null
	go mod tidy
}

Go