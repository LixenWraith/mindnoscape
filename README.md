# mindnoscape

Program Description for Mindnoscape (0.5.4):

## 1. Overview
Mindnoscape is a mind-mapping application that enables users to create, manage, and manipulate mind maps through various operations. The program is designed with a modular architecture, separating concerns into different packages for data management, storage, configuration, user interaction, event handling, session management, and CLI interaction. The application initializes various managers, creates a CLI interface, and uses adapters to handle user commands and interact with the underlying data structures. The program includes robust session management, a comprehensive adapter management system, and a command-line interface for user interaction. It also features an enhanced logging system for better debugging and auditing capabilities.

## 2. Explanation of Packages

### a. model
This package defines the core data structures used throughout the application.

Files:
- command_models.go: Defines the Command struct representing user commands.
- config_models.go: Defines the Config struct for application configuration.
- mindmap_models.go: Defines Mindmap, MindmapInfo, and MindmapFilter structs.
- node_models.go: Defines Node, NodeInfo, and NodeFilter structs.
- user_models.go: Defines User, UserInfo, and UserFilter structs.

### b. storage
This package handles data persistence and retrieval.

Files:
- databases.go: Defines Database interface and BaseDatabase struct for database operations.
- file_io.go: Provides functions for exporting and importing mindmaps to/from files.
- mindmap_store.go: Implements MindmapStore interface and MindmapStorage struct for mindmap-related storage operations.
- node_store.go: Implements NodeStore interface and NodeStorage struct for node-related storage operations.
- sqlite_database.go: Implements SQLiteDatabase for SQLite-specific database operations.
- storage_manager.go: Defines Storage struct and NewStorage function to initialize storage.
- user_store.go: Implements UserStore interface and UserStorage struct for user-related storage operations.

### c. data
This package manages data operations and business logic.

Files:
- data_manager.go: Defines DataManager struct and NewDataManager function to initialize data management.
- mindmap_manager.go: Implements MindmapManager for mindmap-related operations.
- node_manager.go: Implements NodeManager for node-related operations.
- user_manager.go: Implements UserManager for user-related operations.

### d. config
This package handles configuration management.

Files:
- config.go: Provides functions for loading, saving, and retrieving application configuration.

### e. log
This package handles logging functionality.

Files:
- log.go: Implements Logger struct and methods for logging commands, errors, and info messages.

### f. event
This package manages event handling.

Files:
- event.go: Implements EventManager for publishing and subscribing to events.

### g. session
This package manages user sessions and command execution.

Files:
- session.go: Implements Session struct and methods for managing individual user sessions.
- session_manager.go: Implements SessionManager for managing multiple concurrent sessions.
- session_command.go: Implements SessionCommand for validating and executing commands.
- mindmap_handlers.go, node_handlers.go, user_handlers.go, system_handlers.go: Implement command handlers for respective operations.

### h. adapter
This package provides adapters for different interfaces.

Files:
- adapter_manager.go: Implements AdapterManager for managing different adapter instances.
- cli_adapter.go: Implements CLIAdapter for handling CLI-specific interactions.

### i. cli
This package implements the command-line interface.

Files:
- cli.go: Implements CLI struct and methods for running the command-line interface.

## 3. Program Flows

### a. Initialization
1. The program starts in main.go, setting up signal handling and initializing components.
2. Configuration is loaded using the config package.
3. A logger is initialized using the log package, with enhanced info logging capabilities.
4. Storage is initialized using the storage package, based on the loaded configuration.
5. A data manager is created using the data package, which initializes user, mindmap, and node managers.
6. A session manager is created using the session package.
7. An adapter manager is created using the adapter package.
8. A CLI adapter is created and added to the adapter manager.
9. A CLI instance is created using the cli package.

### b. User Interaction
1. The CLI runs in a loop, accepting user input and parsing it into commands.
2. Commands are passed to the CLIAdapter's CommandProcess method.
3. The CLIAdapter expands short commands to full commands and validates them using the SessionCommand struct.
4. Valid commands are sent to the session manager via a command channel.
5. The session manager executes the command within the appropriate session context.
6. Results or errors from command execution are returned to the CLIAdapter via result and error channels.
7. The CLIAdapter returns the result or error to the CLI for display to the user.

### c. Command Execution
1. The session manager receives commands via the command channel.
2. The appropriate session is retrieved based on the session ID associated with the command.
3. The command is executed within the session context using the Session's CommandRun method.
4. Command handlers specific to each operation (user, mindmap, node, system) are invoked based on the command scope and operation.
5. Command handlers interact with the respective managers (UserManager, MindmapManager, NodeManager) to perform the requested operation.
6. Results or errors from the command execution are returned to the session manager.

### d. Data Operations
1. User, mindmap, and node managers in the data package handle respective data operations.
2. These managers interact with the storage package to persist or retrieve data.
3. The storage package provides an abstraction layer for data persistence, with implementations for SQLite database operations.
4. Events are published using the EventManager in the event package for certain operations, allowing other components to react to changes.

### e. Event Handling
1. The EventManager in the event package allows components to publish and subscribe to events.
2. Various managers (e.g., MindmapManager, NodeManager) use events to notify other components of important changes.
3. Event-driven communication helps maintain loose coupling between different parts of the application.

### f. Session Management
1. The SessionManager creates and manages multiple concurrent user sessions.
2. Each user interaction is associated with a specific session identified by a unique session ID.
3. Sessions maintain state, such as the current user and selected mindmap.
4. Command execution and data operations are performed within the context of a session.
5. Sessions are automatically cleaned up after a specified timeout of inactivity.

### g. Adapter Management
1. The AdapterManager maintains different adapter instances, including the CLIAdapter.
2. Adapters provide a layer of abstraction between the user interface (CLI) and the core application logic.
3. The AdapterManager handles the lifecycle of adapter instances, starting and stopping them as needed.

### h. Configuration Management
1. The config package handles loading, saving, and retrieving application configuration.
2. Configuration includes database settings, log file paths, and default user options.
3. The loaded configuration is used to initialize various components, such as storage and loggers.

### i. Logging
1. The Logger in the log package provides functionality for logging commands, errors, and info messages.
2. Different components of the application use the logger to record important events, errors, and debug information for improved troubleshooting and auditing.
3. Log files are created and written to based on the configured log file paths.
4. The logging system now includes more detailed information logging, which can be enabled or disabled as needed.

### j. Node Operations
1. The NodeManager now includes more advanced operations such as sorting nodes, finding nodes based on queries, and handling node movement within the mindmap structure.
2. Node operations are more tightly integrated with the event system, allowing for better synchronization between different parts of the application.

### k. Mindmap Operations
1. The MindmapManager has been enhanced to handle more complex operations, including importing and exporting mindmaps in different formats.
2. Mindmap permissions and visibility (public/private) are now managed more explicitly.


