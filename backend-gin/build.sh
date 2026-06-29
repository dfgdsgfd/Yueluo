#!/usr/bin/env bash
set -euo pipefail
echo "Building yuem-go-linux-amd64..."
GOOS=linux GOARCH=amd64 go build -o yuem-go-linux-amd64 ./cmd/api/
echo "Done."
