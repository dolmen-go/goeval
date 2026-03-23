#!/usr/bin/env bash

rm -rf .coverage.1 .coverage.2 .coverage
mkdir .coverage.1 .coverage.2 .coverage
go test -cover -coverpkg ./... -parallel=10 -race ./echo -args -test.gocoverdir="$(pwd)"/.coverage.1
GOCOVERDIR=$(pwd)/.coverage.2 go test ./echo -args
go tool covdata merge -pcombine -i .coverage.1,.coverage.2 -o .coverage
go tool covdata textfmt -i .coverage -o .coverage.out

test -t 0 && go tool cover -html=.coverage.out

