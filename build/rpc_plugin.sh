#!/bin/bash
protoc --proto_path=pkg/plugin/proto --go_out=pkg/plugin/proto --go_opt=Mplugin.proto=/proto --go-grpc_out=pkg/plugin/proto --go-grpc_opt=Mplugin.proto=/proto pkg/plugin/proto/plugin.proto