# TCP File Transfer Server (Lab 1)

A TCP server-client implementation for file transfer with clean architecture in Go.

## Features

- Basic TCP commands: ECHO, TIME, CLOSE/EXIT/QUIT
- File transfer: UPLOAD, DOWNLOAD
- Connection recovery with TCP keepalive
- Resume functionality for interrupted transfers
- Bitrate calculation and progress display
- Clean architecture with separation of concerns

## Architecture

```
cmd/
├── server/     # Server application entry point
└── client/     # Client application entry point

internal/
├── domain/     # Business entities and interfaces
├── usecase/    # Business logic implementation
├── infrastructure/
│   ├── network/    # TCP server/client implementation
│   └── repository/ # File management

pkg/
└── config/     # Configuration management
```

## Building

```bash
# Build server
go build -o bin/server cmd/server/main.go

# Build client
go build -o bin/client cmd/client/main.go
```

## Running

### Start Server

```bash
# Default (localhost:8080)
./bin/server

# Custom host/port
./bin/server -host 0.0.0.0 -port 9000
```

### Connect with Client

```bash
# Default connection
./bin/client

# Custom server
./bin/client -host 192.168.1.100 -port 9000
```

### Connect with System Utilities

```bash
# Using telnet
telnet localhost 8080

# Using netcat
nc localhost 8080
```

## Commands

### Basic Commands

- `ECHO <text>` - Echo the provided text
- `TIME` - Get current server time
- `CLOSE` / `EXIT` / `QUIT` - Close connection

### File Transfer Commands

#### Client Commands:
- `UPLOAD <local_path> <remote_name>` - Upload file to server
- `DOWNLOAD <remote_name> <local_path>` - Download file from server

#### Direct Protocol Commands:
- `UPLOAD <filename>` - Initiate file upload (server expects file data)
- `DOWNLOAD <filename>` - Request file download (server sends file info)

## Network Utility Examples

### Port Scanning with nmap

```bash
# Scan specific port
nmap -p 8080 localhost

# Scan port range
nmap -p 8000-8100 localhost

# Service detection
nmap -sV -p 8080 localhost

# Comprehensive scan
nmap -A -p 8080 localhost
```

### Check Open Sockets with netstat

```bash
# Show TCP connections
netstat -an | grep 8080

# Show listening ports
netstat -ln | grep 8080

# Show process using port
netstat -tlnp | grep 8080

# Alternative with ss (modern)
ss -tlnp | grep 8080
```

### Connection Testing

```bash
# Test connectivity
telnet localhost 8080

# Send data with netcat
echo "ECHO Hello World" | nc localhost 8080

# Interactive session
nc localhost 8080
```

## Usage Examples

### Interactive Client Session

```
$ ./bin/client
Connected to server localhost:8080
client> ECHO Hello TCP Server
Server: Hello TCP Server
client> TIME
Server: 2024-01-15T10:30:45Z
client> UPLOAD test.txt remote.txt
Upload progress: 25.00% (1.23 MB/s)
Upload progress: 50.00% (1.45 MB/s)
Upload progress: 75.00% (1.38 MB/s)
Upload progress: 100.00% (1.41 MB/s)
Upload completed: test.txt (2.50 MB, 1.41 MB/s)
client> DOWNLOAD remote.txt downloaded.txt
Download progress: 25.00% (1.28 MB/s)
Download progress: 50.00% (1.52 MB/s)
Download progress: 75.00% (1.47 MB/s)
Download progress: 100.00% (1.49 MB/s)
Download completed: downloaded.txt (2.50 MB, 1.49 MB/s)
client> EXIT
```

### Telnet Session

```
$ telnet localhost 8080
Trying 127.0.0.1...
Connected to localhost.
Escape character is '^]'.
ECHO Hello from telnet
Hello from telnet
TIME
2024-01-15T10:30:45Z
CLOSE
Connection closing.
Connection closed by foreign host.
```

## Configuration

Default configuration can be modified in `pkg/config/config.go`:

```go
ServerConfig{
    Host:           "localhost",
    Port:           "8080",
    KeepAlive:      true,
    KeepAliveIdle:  30 * time.Second,
    KeepAliveCount: 3,
    KeepAliveIntvl: 10 * time.Second,
    BufferSize:     8192,
    UploadDir:      "./uploads",
    SessionTimeout: 5 * time.Minute,
}
```

## TCP Features Implemented

### Keepalive Configuration
- SO_KEEPALIVE enabled for connection monitoring
- Configurable idle time, count, and interval
- Automatic connection recovery

### File Transfer Protocol
- Chunked file transfer with configurable buffer size
- Progress tracking with percentage and bitrate
- Resume support for interrupted transfers
- Session management for transfer state

### Error Handling
- Network timeout detection
- Connection recovery mechanisms
- Graceful shutdown handling

## Testing

### Test Basic Commands

```bash
# Test ECHO
echo -e "ECHO Hello World\r\n" | nc localhost 8080

# Test TIME
echo -e "TIME\r\n" | nc localhost 8080

# Test CLOSE
echo -e "CLOSE\r\n" | nc localhost 8080
```

### Test File Transfer

```bash
# Create test file
echo "This is a test file" > test.txt

# Upload using custom client
./bin/client
client> UPLOAD test.txt uploaded.txt

# Download using custom client
client> DOWNLOAD uploaded.txt downloaded.txt

# Verify files
diff test.txt downloaded.txt
```

## Performance

The server supports:
- Configurable buffer sizes for optimal throughput
- Bitrate calculation and display
- Connection pooling (single-threaded per connection)
- Efficient file I/O with streaming

## Security Considerations

- No authentication implemented (educational purpose)
- File access restricted to upload directory
- Connection timeout prevents resource exhaustion
- Basic input validation for commands

## Troubleshooting

### Port Already in Use
```bash
# Find process using port
lsof -i :8080

# Kill process
kill -9 <PID>
```

### Connection Refused
- Check if server is running
- Verify firewall settings
- Check correct host/port

### File Transfer Issues
- Ensure upload directory exists and is writable
- Check file permissions
- Verify sufficient disk space

## TCP Protocol Questions (Lab Answers)

### 1. Установление и разрыв соединения на уровне протокола TCP

**Установление соединения (Three-way handshake):**
1. Client → Server: SYN (seq=x)
2. Server → Client: SYN-ACK (seq=y, ack=x+1)
3. Client → Server: ACK (ack=y+1)

**Разрыв соединения (Four-way handshake):**
1. Client → Server: FIN (seq=u)
2. Server → Client: ACK (ack=u+1)
3. Server → Client: FIN (seq=v)
4. Client → Server: ACK (ack=v+1)

### 2. Скользящее окно передачи данных в протоколе TCP

**Назначение:** Контроль потока данных для предотвращения переполнения буфера получателя.

**Механизм функционирования:**
- Получатель сообщает размер окна (количество байт, которые может принять)
- Отправитель может отправить только количество байт в пределах окна
- Окно динамически изменяется в зависимости от скорости обработки получателя

### 3. Механизм медленного старта (Slow Start)

Алгоритм для определения оптимальной скорости передачи:
- Начало с минимального размера окна (1-2 MSS)
- Удвоение окна при каждом успешном ACK
- Переход к избеганию перегрузки при достижении порога

### 4. Алгоритм Нэгла (Nagle)

**Преимущества:**
- Уменьшение количества мелких пакетов
- Снижение сетевой нагрузки
- Повышение эффективности передачи

**Недостатки:**
- Increased latency for interactive applications
- Poor performance for real-time communications
- Delayed ACK interactions

### 5. Срочные данные в TCP

**Механизм применения:**
- Установка флага URG в заголовке TCP
- Указание поля Urgent Pointer
- Немедленная доставка принимающему приложению

**Ограничения:**
- Максимальный размер срочных данных - 65535 байт
- Не все реализации поддерживают корректно
- Проблемы с прохождением через NAT/firewall
