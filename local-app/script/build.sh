#!bin/bash
echo "Starting build..."
go build -o bin/mindnoscape src/cmd/mindnoscape/main.go
echo "Starting db and config clear..."
rm data/*
echo "Done"

