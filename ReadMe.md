# Laguna - In-Memory Key-Value Database

Laguna is a high-performance, in-memory key-value database with write-ahead logging (WAL) and master-slave replication capabilities. It provides a simple, Redis-like interface for storing and retrieving key-value pairs with built-in durability and replication support.

## Features

- **In-Memory Storage**: Fast, sharded in-memory key-value storage engine
- **Write-Ahead Logging (WAL)**: Ensures data durability by writing changes to disk before applying them
- **Master-Slave Replication**: Automatic data synchronization from master to slave nodes
- **TCP Protocol**: Network interface for remote client connections
- **Transaction Support**: Transaction IDs for tracking operations
- **Configurable Logging**: Structured logging with configurable levels
- **Performance Profiling**: Built-in pprof support for performance analysis

## Supported Operations

The database supports three basic operations:

- **GET** `<key>` - Retrieve a value by key
- **SET** `<key>` `<value>` - Store a key-value pair
- **DEL** `<key>` - Delete a key-value pair

## Configuration

Laguna is configured via YAML configuration files. The main configuration file is `config.yaml` in the root directory.

### Configuration Structure

```yaml
engine:
  type: inmemory          # Storage engine type (currently only 'inmemory')
  shards: 100            # Number of shards for data distribution

logger:
  level: debug           # Logging level: debug, info, warn, error
  out: stdout            # Output destination: stdout, file path
  disable_stacktrace: true  # Disable stack traces in logs

transport:
  type: tcp              # Transport type: tcp or cli
  server:
    address: localhost:8080      # Server listening address
    max_conn: 100                # Maximum concurrent connections
    max_message_size: 4KB        # Maximum message size (supports KB, MB, GB)
    idle_timeout: 5m             # Idle connection timeout
    sem_wait_timeout: 5s         # Semaphore wait timeout

wal:
  enable: true                   # Enable/disable WAL
  flush_batch_size: 100          # Number of operations before flushing to disk
  flush_interval: "10ms"         # Interval for periodic flushing
  max_segment_size: "1MB"        # Maximum size of WAL segment files
  directory: "./data/wal"        # Directory for WAL files

replication:
  enable: true                   # Enable/disable replication
  is_master: true                # true for master, false for slave
  sync_interval: "1s"            # Interval for replication synchronization
  client:
    retry_count: 3               # Number of retries for failed requests
    address: localhost:9090      # Master server address (for slaves)
    max_resp_size: 1MB           # Maximum response size
    deadline: 10m                # Request deadline
  server:
    address: localhost:9090      # Replication server listening address (for master)
    max_conn: 5                  # Maximum replication connections
    max_message_size: 400KB      # Maximum replication message size
    idle_timeout: 10m            # Idle connection timeout
    sem_wait_timeout: 10s        # Semaphore wait timeout

profiler:
  enable: true                   # Enable/disable pprof profiler
  address: localhost:6060        # Profiler HTTP server address
```

### Setting Up Master Node

1. Create a `config.yaml` file with replication configured as master:

```yaml
engine:
  type: inmemory
  shards: 100

logger:
  level: info
  out: stdout
  disable_stacktrace: true

transport:
  type: tcp
  server:
    address: localhost:8080
    max_conn: 100
    max_message_size: 4KB
    idle_timeout: 5m
    sem_wait_timeout: 5s

wal:
  enable: true
  flush_batch_size: 100
  flush_interval: "10ms"
  max_segment_size: "1MB"
  directory: "./data/wal"

replication:
  enable: true
  is_master: true
  sync_interval: "1s"
  client:
    retry_count: 3
    address: localhost:9090
    max_resp_size: 1MB
    deadline: 10m
  server:
    address: localhost:9090
    max_conn: 5
    max_message_size: 400KB
    idle_timeout: 10m
    sem_wait_timeout: 10s

profiler:
  enable: false
  address: localhost:6060
```

2. Ensure the WAL directory exists or will be created automatically
3. Start the master node:

```bash
go run main.go
```

### Setting Up Slave Node

1. Create a configuration file (e.g., `slave_instance/config.yaml`) with replication configured as slave:

```yaml
engine:
  type: inmemory
  shards: 100

logger:
  level: info
  out: stdout
  disable_stacktrace: true

transport:
  type: tcp
  server:
    address: localhost:8081
    max_conn: 100
    max_message_size: 4KB
    idle_timeout: 5m
    sem_wait_timeout: 5s

wal:
  enable: true                    # WAL must be enabled for replication
  flush_batch_size: 100
  flush_interval: "10ms"
  max_segment_size: "1MB"
  directory: "./slave_instance/data/wal"

replication:
  enable: true
  is_master: false                # Set to false for slave
  sync_interval: "1s"
  client:
    retry_count: 3
    address: localhost:9090       # Master replication server address
    max_resp_size: 1MB
    deadline: 10m
  server:
    address: localhost:9091       # Different port for slave replication server
    max_conn: 5
    max_message_size: 400KB
    idle_timeout: 10m
    sem_wait_timeout: 10s
```

**Important Notes:**
- WAL must be enabled for replication to work
- Slave nodes can only perform read operations (GET). Write operations (SET, DEL) are rejected on slaves
- The slave's `client.address` should point to the master's replication server address
- Use different ports for transport and replication servers to avoid conflicts

2. Update the slave's main.go to use the correct config path
3. Start the slave node:

```bash
go run slave_instance/main.go
```

## Usage

### Connecting to the Database

Connect to the database using TCP:

```bash
# Connect to master on port 8080
nc localhost 8080

# Or connect to slave on port 8081
nc localhost 8081
```

### Running Commands

Once connected, you can send commands:

```
SET user:1 "John Doe"
OK

GET user:1
John Doe

SET user:2 "Jane Smith"
OK

DEL user:1
OK

GET user:1
(empty line - key not found)
```

Commands must be terminated with a newline character (`\n`).

### Using the Client

A sample client implementation is available in `client/client.go`. You can build and run it:

```bash
go build -o client ./client
./client
```

## Build and Run

### Prerequisites

- Go 1.25.1 or later

### Build

```bash
go build -o laguna main.go
```

### Run

```bash
# Run master node
./laguna

# Or using go run
go run main.go
```

### Run Tests

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -v -race -cover ./...

# Run tests for specific package
go test ./internal/database/...
```

### Makefile Commands

```bash
make fmt      # Format code
make test     # Run tests with race detection and coverage
make lint     # Run linter
make tidy     # Tidy dependencies
make run      # Run the application
```

## Architecture

### Components

- **Storage Engine**: In-memory key-value store with sharding support
- **WAL (Write-Ahead Log)**: Persists changes to disk in segments for durability
- **Replication**: Master-slave replication for high availability
- **Transport Layer**: TCP server for client connections
- **Query Handler**: Parses and executes client commands
- **Segment Manager**: Manages WAL segment files and indexing

### Data Flow

1. Client sends a command via TCP
2. Query handler parses the command
3. If WAL is enabled, the operation is written to WAL first
4. Operation is executed in the storage engine
5. On master nodes, changes are replicated to slaves
6. Response is sent back to the client

### Replication Flow

1. Master receives write operations (SET/DEL)
2. Operations are written to master's WAL
3. Slave periodically requests new data from master using LSN (Log Sequence Number)
4. Master sends log entries since the last LSN received by slave
5. Slave writes received entries to its WAL and applies them to storage

## Performance Profiling

When the profiler is enabled, you can access pprof endpoints:

```bash
# View CPU profile
go tool pprof http://localhost:6060/debug/pprof/profile

# View heap profile
go tool pprof http://localhost:6060/debug/pprof/heap

# View goroutine profile
go tool pprof http://localhost:6060/debug/pprof/goroutine
```

## Limitations

- Currently supports only in-memory storage (no persistent on-disk storage engine)
- Replication requires WAL to be enabled
- Slaves can only perform read operations
- Single master replication (no multi-master support)
