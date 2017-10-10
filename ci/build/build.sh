#!/bin/bash -e

env
go version

echo "Install dep."
go get -u github.com/golang/dep/cmd/dep

echo "Install dependency package."
dep ensure

echo "Build leeroy."
go build .

