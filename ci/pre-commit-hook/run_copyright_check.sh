#!/usr/bin/env bash

python3 -m pip install pre-commit
pre-commit install
go install golang.org/x/tools/cmd/goimports@latest
go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.57.2
go install honnef.co/go/tools/cmd/staticcheck@2022.1
go run ./ci/pre-commit-hook/copyrights.go vmray-cluster-operator
