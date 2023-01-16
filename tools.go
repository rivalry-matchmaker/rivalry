//go:build tools
// +build tools

package frontend

// As per recommendation on https://github.com/golang/go/wiki/Modules#how-can-i-track-tool-dependencies-for-a-module,
// this file tracks our tool dependencies.

import (
	_ "github.com/golang/mock/gomock"
	_ "github.com/golang/mock/mockgen"
	_ "github.com/golang/protobuf/protoc-gen-go"
	_ "github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway"
	_ "github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2"
	_ "golang.org/x/lint/golint"
)
