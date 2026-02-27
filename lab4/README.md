# Lab4: UDP Multiservice Server with Thread Pool

## Overview

This project implements a **UDP multiservice server** with **dynamic thread pool** management using **Clean Architecture** principles and **Go 1.26** best practices. The server supports multiple services running on different ports concurrently, with automatic thread pool scaling based on load.

## Architecture

### Clean Architecture Implementation

The project follows Clean Architecture principles with clear separation of concerns:

```
lab4/
├── cmd/                    # Application entry points
│   ├── server/            # Server application
│   └── client/            # Client application
├── internal/              # Private application code
│   ├── domain/           # Business entities and interfaces
│   ├── infrastructure/   # External concerns (network, storage)
│   └── usecase/          # Business logic implementation
├── pkg/                  # Public library code
│   └── config/           # Configuration management
└── Makefile             # Build automation
```

### Key Components

- **Domain Layer**: Core business entities and interfaces
- **Infrastructure Layer**: UDP networking, thread pool implementation
- **Use Case Layer**: Service implementations (Echo, Time, Calculator, Statistics)
- **Application Layer**: Server and client applications

## Features

### Multiservice Architecture

The server implements **Variant 17** from the assignment:
- **UDP protocol** with thread spawning on request
- **Multiple services** each running on dedicated ports
- **Thread pool** with dynamic scaling
- **Concurrent request handling** with proper synchronization

### Services

1. **Echo Service** (Port 8081)
   - Echoes back received text
   - Simple text processing service

2. **Time Service** (Port 8082)
   - Returns current server time
   - Supports both RFC3339 and Unix timestamp formats

3. **Calculator Service** (Port 8084)
   - Performs mathematical calculations
   - Supports: +, -, *, / operations
   - Floating-point arithmetic

4. **Statistics Service** (Port 8085)
   - Provides server and service statistics
   - Thread pool monitoring
   - Request/response metrics

### Thread Pool Management

- **Dynamic scaling**: Automatically adjusts worker count based on load
- **Configurable bounds**: Min/max worker limits
- **Timeout management**: Idle workers automatically terminate
- **Load-based expansion**: Expands when queue usage exceeds threshold

## Configuration

### Default Configuration

```yaml
Server:
  Host: "localhost"
  ReadBuffer: 4096
  WriteBuffer: 4096
  MaxPacketSize: 65536
  IdleTimeout: 60s

ThreadPool:
  MinWorkers: 5
  MaxWorkers: 50
  QueueSize: 1000
  WorkerTimeout: 30s
  ExpandThreshold: 0.8

Services:
  echo:
    Port: 8081
    Enabled: true
    MaxRequests: 1000
    Timeout: 5s
  time:
    Port: 8082
    Enabled: true
    MaxRequests: 1000
    Timeout: 5s
  calc:
    Port: 8084
    Enabled: true
    MaxRequests: 1000
    Timeout: 10s
  stats:
    Port: 8085
    Enabled: true
    MaxRequests: 100
    Timeout: 5s
```

## Installation and Setup

### Prerequisites

- **Go 1.26** or higher
- **Make** (for build automation)
- **Git** (for version management)

### Build from Source

```bash
# Clone the repository
git clone <repository-url>
cd lab4

# Install dependencies
make deps

# Build for current platform
make build

# Build for all platforms
make build-all

# Build for specific platform
make build-platform PLATFORM=linux/amd64
```

### Development Setup

```bash
# Install development tools
make install-tools

# Run in development mode
make dev-server    # In one terminal
make dev-client    # In another terminal
```

## Usage

### Starting the Server

```bash
# Using Makefile
make run-server

# Or directly
./bin/lab4-server

# With custom host
./bin/lab4-server -host=0.0.0.0
```

### Using the Client

```bash
# Using Makefile
make run-client

# Or directly
./bin/lab4-client

# With custom timeout
./bin/lab4-client -timeout=30s
```

### Client Commands

#### Quick Commands

```bash
# Echo service
client> echo Hello World!
Response: ECHO: Hello World!

# Time service
client> time
Response: Current time: 2024-02-27T15:30:45Z

client> time unix
Response: Unix timestamp: 1709046645

# Calculator service
client> calc 5 * 10
Response: 5.00 * 10.00 = 50.00

client> calc 100 / 4
Response: 100.00 / 4.00 = 25.00

# Statistics service
client> stats all
Response: {
  "echo": {
    "requests_received": 10,
    "requests_processed": 10,
    "errors": 0,
    "avg_response_time": "1.234ms",
    "last_request": "2024-02-27T15:30:45Z"
  },
  ...
}

client> stats service echo
Response: {
  "service": "echo",
  "requests_received": 10,
  "requests_processed": 10,
  "errors": 0,
  "avg_response_time": "1.234ms",
  "last_request": "2024-02-27T15:30:45Z"
}

client> stats help
Response: Stats Service Commands:
ALL - Show statistics for all services
SERVICE <service_name> - Show statistics for specific service
POOL - Show thread pool statistics
HELP - Show this help message
```

#### Advanced Commands

```bash
# Connect to specific service
client> connect localhost:8081
Connected to localhost:8081

# Send custom requests
client> send echo ECHO "Custom message"
Response: ECHO: Custom message

client> send time UNIX ""
Response: Unix timestamp: 1709046645
```

## API Reference

### Request Format

```json
{
  "id": "unique-request-id",
  "command": "COMMAND_NAME",
  "data": "request data"
}
```

### Response Format

```json
{
  "id": "unique-request-id",
  "service": "service-type",
  "data": "response data",
  "timestamp": 1709046645,
  "error": "error message (if any)"
}
```

### Service Endpoints

| Service | Port | Commands | Description |
|---------|------|----------|-------------|
| Echo | 8081 | Any text | Echoes received text |
| Time | 8082 | GET, UNIX | Returns current time |
| Calculator | 8084 | Arithmetic expressions | Performs calculations |
| Statistics | 8085 | ALL, SERVICE, POOL, HELP | Server statistics |

## Performance Characteristics

### Thread Pool Behavior

- **Initial Workers**: 5 threads
- **Maximum Workers**: 50 threads
- **Queue Size**: 1000 pending requests
- **Expansion Trigger**: 80% queue utilization
- **Worker Timeout**: 30 seconds idle

### Performance Metrics

- **Concurrent Requests**: Up to 50 simultaneous
- **Request Processing**: Sub-millisecond average
- **Memory Usage**: ~10MB base + per-request overhead
- **Network I/O**: Non-blocking UDP with buffers

## Testing

### Running Tests

```bash
# Run all tests
make test

# Run with coverage
make test-coverage

# Run benchmarks
make benchmark

# Run security scan
make security
```

### Test Coverage

The project includes comprehensive tests for:
- Thread pool management
- Service implementations
- Network communication
- Configuration handling
- Error scenarios

## Build Targets

### Available Make Targets

```bash
make help              # Show all available targets
make all              # Clean, deps, lint, test, build
make deps             # Install dependencies
make build            # Build for current platform
make build-all        # Build for all platforms
make run-server       # Build and run server
make run-client       # Build and run client
make clean            # Clean build artifacts
make package          # Create release packages
```

### Cross-Platform Builds

Supported platforms:
- Linux (amd64, arm64)
- macOS (amd64, arm64)
- Windows (amd64, arm64)

```bash
# Build for all platforms
make build-all

# Build for specific platform
make build-platform PLATFORM=linux/amd64

# Create release packages
make package
```

## Monitoring and Debugging

### Statistics Service

The statistics service provides real-time monitoring:
- Request counts per service
- Error rates
- Average response times
- Thread pool utilization

### Logging

The server provides structured logging:
- Service startup/shutdown events
- Connection handling
- Error conditions
- Performance metrics

### Debug Mode

Enable verbose logging:
```bash
./bin/lab4-server -v
```

## Security Considerations

### Network Security

- **UDP Protocol**: Connectionless, requires careful input validation
- **Packet Size Limits**: Configurable maximum packet size
- **Timeout Protection**: Request timeouts prevent resource exhaustion
- **Input Validation**: All inputs are validated before processing

### Thread Pool Security

- **Resource Limits**: Maximum worker count prevents DoS
- **Queue Limits**: Bounded queue prevents memory exhaustion
- **Timeout Management**: Idle workers automatically terminate

## Troubleshooting

### Common Issues

1. **Port Already in Use**
   ```bash
   # Check port usage
   lsof -i :8081
   
   # Kill process using port
   kill -9 <PID>
   ```

2. **Permission Denied**
   ```bash
   # Use non-privileged ports (>1024)
   # Or run with sudo (not recommended)
   ```

3. **High Memory Usage**
   ```bash
   # Check thread pool stats
   client> stats pool
   
   # Reduce max workers in config
   ```

4. **Connection Timeouts**
   ```bash
   # Increase timeout
   ./bin/lab4-client -timeout=30s
   ```

### Performance Tuning

1. **Thread Pool Optimization**
   - Adjust `MinWorkers` based on typical load
   - Set `MaxWorkers` to prevent resource exhaustion
   - Tune `ExpandThreshold` for responsive scaling

2. **Network Optimization**
   - Adjust `ReadBuffer` and `WriteBuffer` sizes
   - Set appropriate `IdleTimeout` values
   - Configure `MaxPacketSize` for your use case

## Contributing

### Development Workflow

1. Fork the repository
2. Create a feature branch
3. Make changes with tests
4. Run `make all` to verify
5. Submit a pull request

### Code Style

- Follow Go best practices
- Use `gofmt` and `goimports`
- Write comprehensive tests
- Document public APIs

## License

This project is part of the NSSaDS course work and follows academic integrity guidelines.

## Version History

- **v1.0.0**: Initial implementation with UDP multiservice server
- **v1.1.0**: Added thread pool dynamic scaling
- **v1.2.0**: Enhanced statistics and monitoring

## Contact

For questions or issues related to this project, please contact the course instructor or teaching assistant.
