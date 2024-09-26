#!/bin/bash

# Set the GitHub repository URL
REPO_URL="https://github.com/LixenWraith/mindnoscape.git"

# Set the installation directory to the current directory
INSTALL_DIR="."

# Check if the current directory is empty
if [ "$(ls -A $INSTALL_DIR)" ]; then
    echo "Error: Current directory is not empty. Please run this script in an empty directory."
    exit 1
fi

echo "Cloning Mindnoscape repository..."
git clone "$REPO_URL" "$INSTALL_DIR"
if [ $? -ne 0 ]; then
    echo "Failed to clone the repository."
    exit 1
fi

# Change to the local-app directory
cd "$INSTALL_DIR/local-app" || exit 1

# Ensure all dependencies are downloaded
echo "Ensuring all dependencies are up to date..."
go mod tidy
if [ $? -ne 0 ]; then
    echo "Failed to update dependencies."
    exit 1
fi

# Build the Mindnoscape application
echo "Building Mindnoscape..."
go build -o mindnoscape src/cmd/main.go src/cmd/bootstrap.go
if [ $? -ne 0 ]; then
    echo "Failed to build the application."
    exit 1
fi

echo "Mindnoscape has been successfully installed."
echo "You can run it by executing: ./local-app/mindnoscape"