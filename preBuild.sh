#!/usr/bin/env bash

# Set the needed URL scheme
git config --global url."ssh://git@github.com/".insteadOf "https://github.com/"
# Export Go private repository
export GOPRIVATE=github.com/wix-system
# Download dependencies
go mod -mod=mod download
# Build application
[ -d bin ] || mkdir bin
go build -mod=mod -o bin/tfChek .
ls -lah bin