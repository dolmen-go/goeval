#!/usr/bin/env bash

set -euo pipefail
cd "$(dirname "$0")" || exit 1

rm -rf .coverage.1 .coverage.2 .coverage
mkdir .coverage.1 .coverage.2 .coverage
go test -cover -coverpkg ./... -parallel=10 -race -args -test.gocoverdir="$(pwd)"/.coverage.1
GOCOVERDIR=$(pwd)/.coverage.2 go test ./...
go tool covdata merge -pcombine -i .coverage.1,.coverage.2 -o .coverage
go tool covdata percent -i .coverage
go tool covdata textfmt -i .coverage -o .coverage.out
rm -rf .coverage.1 .coverage.2 .coverage

test -t 0 && go tool cover -html=.coverage.out

