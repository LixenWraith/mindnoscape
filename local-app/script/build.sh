#!/usr/bin/env bash
echo "Starting build..."
go build -o bin/logviewer ./src/cmd/logviewer/
go build -o bin/mindnoscape ./src/cmd/mindnoscape/
echo "Starting db and config clear..."
rm data/*
echo "Done"

