#!/bin/bash -e

env
go version

echo "Build leeroy."
go build .

