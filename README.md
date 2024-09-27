# mindnoscape

## 1. Overview
Mindnoscape is a mind-mapping application that enables users to create, manage, and manipulate mind maps through various operations. The program is designed with a modular architecture, separating concerns into different packages for storage, data management, configuration, user interaction, event handling, and session management. The application initializes various managers, creates a CLI interface, and uses interface adapters to handle user commands and interact with the underlying data structures. The program includes session management, an adapter management system, and a command-line interface for user interaction. It also features a logging system for debugging and auditing capabilities.

## 2. Packages

### a. model
This package defines the core data structures used throughout the application.

### b. storage
This package handles data persistence and retrieval.

### c. data
This package manages data operations and business logic.

### d. config
This package handles configuration management.

### e. log
This package handles logging functionality.

### f. event
This package manages event handling.

### g. session
This package manages user sessions and command execution.

### h. adapter
This package provides adapters for different interfaces.

### i. cli
This package implements the command-line interface.

## 3. External dependencies

### go-sqlite3 (https://github.com/mattn/go-sqlite3)

## 4. Installation

This program requires Go to compile (for version refer to mindnoscape/local-app/go.mod ).

Download the installation script:

	curl -O https://raw.githubusercontent.com/LixenWraith/mindnoscape/main/install_mindnoscape.sh
	chmod +x install_mindnoscape.sh

Install in an empty directory:

	./install_mindnoscape.sh /path/to/empty/directory
