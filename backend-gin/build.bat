@echo off
echo Building yuem-go-linux-amd64...
set GOOS=linux
set GOARCH=amd64
go build -o yuem-go-linux-amd64 ./cmd/api/
echo Done.
