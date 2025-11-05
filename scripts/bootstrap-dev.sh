#!/usr/bin/env bash
set -euo pipefail

# Установка системных инструментов (Ubuntu/WSL)
if command -v apt-get >/dev/null 2>&1; then
  sudo apt-get update -y
  sudo apt-get install -y git make protobuf-compiler
fi

# Установка Go-инструментов
GO_BIN="$(go env GOPATH)/bin"
go install github.com/bufbuild/buf/cmd/buf@latest
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
go install github.com/envoyproxy/protoc-gen-validate@latest
go install github.com/vektra/mockery/v2@latest

# golangci-lint
curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b "${GO_BIN}"

echo "Installed tools to ${GO_BIN}. Ensure it's on your PATH."


