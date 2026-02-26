# UDP File Transfer Server (Lab 2)

A UDP-based file transfer implementation with reliability mechanisms, sliding window protocol, and performance optimization. This is Lab 2 modification of the TCP implementation from Lab 1.

## Features

### Core UDP Features
- **Reliable UDP Protocol**: Custom implementation with packet acknowledgment and retransmission
- **Sliding Window Protocol**: Prevents waiting for ACK on every packet
- **Packet Loss Handling**: Automatic retransmission with configurable timeouts
- **Performance Monitoring**: Real-time bitrate calculation and optimization
- **Cross-Platform Support**: Windows, macOS, Linux (AMD64/ARM64)

### Advanced Features
- **Dynamic Buffer Size Optimization**: Automatically finds optimal buffer size
- **Network Resilience**: Handles DROP/REJECT firewall rules
- **Performance Comparison**: UDP vs TCP performance analysis
- **Connection Recovery**: Handles network interruptions gracefully

## Architecture

```
cmd/
├── server/     # UDP server application
└── client/     # UDP client application

internal/
├── domain/     # UDP packets, sliding window, interfaces
├── usecase/    # Command handling logic
├── infrastructure/
│   ├── network/    # UDP reliability, connection management
│   └── repository/ # File management

pkg/
└── config/     # UDP-specific configuration
```

## UDP Protocol Implementation

### Packet Structure
```
+--------+--------+--------+--------+--------+--------+--------+
| Type   | SeqNum | AckNum | Window | Flags  | Timestamp |
+--------+--------+--------+--------+--------+--------+
| Checksum | DataLen | Data...                         |
+--------+--------+----------------------------------------+
```

### Packet Types
- **DATA (1)**: File data packets
- **ACK (2)**: Acknowledgment packets
- **NACK (3)**: Negative acknowledgment
- **SYN (4)**: Connection initiation
- **FIN (5)**: Connection termination
- **COMMAND (7)**: Command packets
- **RESPONSE (8)**: Command responses

### Sliding Window Protocol
- Configurable window size (default: 64 packets)
- Pipelining for improved throughput
- Automatic window management based on ACKs

## Building

### Prerequisites
- Go 1.26 or later
- Make (for cross-platform builds)

### Cross-Platform Build

```bash
# Build for all platforms
make build-all

# Build for specific platforms
make build-linux    # Linux AMD64
make build-windows  # Windows AMD64
make build-darwin   # macOS AMD64
make build-arm64    # ARM64 platforms
```

### Development Build

```bash
# Build for current platform
make build

# Run in development mode
make run-server
make run-client
```

## Running

### Start UDP Server

```bash
# Default (localhost:8080)
./bin/server-linux-amd64

# Custom host/port
./bin/server-linux-amd64 -host 0.0.0.0 -port 9000

# Run performance tests
./bin/server-linux-amd64 -test
```

### Connect with UDP Client

```bash
# Default connection
./bin/client-linux-amd64

# Custom server
./bin/client-linux-amd64 -host 192.168.1.100 -port 9000

# Run performance comparison
./bin/client-linux-amd64 -test
```

## Commands

### Basic Commands
- `ECHO <text>` - Echo the provided text
- `TIME` - Get current server time
- `CLOSE/EXIT/QUIT` - Close connection

### File Transfer Commands
- `UPLOAD <local_path> <remote_name>` - Upload file to server
- `DOWNLOAD <remote_name> <local_path>` - Download file from server

### Performance Commands
- `PERF` - Show performance report
- `TEST` - Run performance tests
- `HELP` - Show help

## Performance Testing

### Buffer Size Optimization

The client automatically tests different buffer sizes to find optimal performance:

```bash
# Test buffer sizes: 512, 1024, 2048, 4096, 8192, 16384, 32768 bytes
./bin/client-linux-amd64 -test
```

### UDP vs TCP Comparison

```bash
# Run comprehensive benchmark
make benchmark

# Or manual comparison
# 1. Run TCP server from Lab 1
# 2. Run UDP server from Lab 2
# 3. Compare performance metrics
```

## Network Resilience Testing

### Simulating Packet Loss (DROP)

```bash
# Add DROP rule (simulates packet loss)
sudo iptables -A OUTPUT -p udp --dport 8080 -j DROP

# Test with packet loss
./bin/client-linux-amd64 -test

# Remove rule
sudo iptables -D OUTPUT -p udp --dport 8080 -j DROP
```

### Simulating Connection Refusal (REJECT)

```bash
# Add REJECT rule (simulates connection refusal)
sudo iptables -A OUTPUT -p udp --dport 8080 -j REJECT

# Test with rejections
./bin/client-linux-amd64 -test

# Remove rule
sudo iptables -D OUTPUT -p udp --dport 8080 -j REJECT
```

### Physical Network Disconnection

```bash
# Test physical disconnection
# 1. Start server and client
# 2. Unplug network cable
# 3. Observe timeout handling
# 4. Reconnect and test recovery
```

## Configuration

### UDP Configuration

```go
UDPConfig{
    WindowSize:           64,                    // Sliding window size
    PacketTimeout:        100 * time.Millisecond,  // Packet timeout
    RetransmissionTimeout: 500 * time.Millisecond,  // Retransmission timeout
    MaxRetransmissions:   5,                     // Max retransmissions
    BufferSizes:         []int{512, 1024, 2048, 4096, 8192, 16384, 32768},
    TestDuration:        30 * time.Second,
}
```

### Performance Tuning

- **Window Size**: Larger windows improve throughput but increase memory usage
- **Packet Timeout**: Balance between responsiveness and false timeouts
- **Buffer Size**: Optimal size varies by network conditions

## Performance Analysis

### Expected Results

Based on UDP characteristics:

1. **Higher Throughput**: UDP should be 1.5x+ faster than TCP
2. **Lower Latency**: No handshake or congestion control overhead
3. **Packet Loss**: Some loss expected, handled by retransmission

### Buffer Size Optimization

The optimal buffer size balances:

- **Network MTU**: Reduces fragmentation (typically 1500 bytes)
- **System Overhead**: Fewer system calls for larger buffers
- **Memory Efficiency**: Larger buffers use more memory
- **UDP Characteristics**: No flow control, can send bursts

### Sample Performance Output

```
=== Performance Report ===
File: test_file.dat
Total Size: 10.00 MB
Transferred: 10.00 MB
Packets Sent: 1563
Packets Lost: 23
Retransmissions: 23
Packet Loss Rate: 1.47%
Average Bitrate: 15.23 MB/s

=== UDP Performance Statistics ===
Packets Sent: 1563
Packets Lost: 23
Retransmissions: 23
Packet Loss Rate: 1.47%
Average Bitrate: 15.23 MB/s
UDP vs TCP Performance Ratio: 1.52
✓ UDP is 1.52x faster than TCP (meets requirement)
Optimal Buffer Size: 8192 bytes (16.45 MB/s)
===============================
```

## UDP Protocol Advantages

### Why UDP is Faster Than TCP

1. **No Connection Overhead**: No three-way handshake
2. **No Congestion Control**: No rate limiting
3. **No Acknowledgment Delays**: Our implementation uses pipelining
4. **Smaller Headers**: 8 bytes vs 20+ bytes for TCP
5. **No Retransmission Delays**: Custom timeout handling

### Trade-offs

- **Reliability**: Implemented at application layer
- **Ordering**: Managed by sequence numbers
- **Flow Control**: Sliding window prevents overflow
- **Error Detection**: Checksums and acknowledgments

## Troubleshooting

### Common Issues

**High Packet Loss**
- Check network quality
- Adjust timeout values
- Reduce window size

**Poor Performance**
- Test different buffer sizes
- Check for network congestion
- Verify firewall rules

**Connection Timeouts**
- Increase timeout values
- Check network connectivity
- Verify server is running

### Debug Commands

```bash
# Monitor UDP traffic
sudo tcpdump -i any -n udp port 8080

# Check UDP socket statistics
netstat -su | grep udp

# Monitor system resources
top -p $(pgrep server)
```

## Testing Scenarios

### Basic Functionality
```bash
# Test commands
echo "ECHO Hello UDP" | nc -u localhost 8080

# Test file transfer
./bin/client-linux-amd64
client> UPLOAD test.txt remote.txt
client> DOWNLOAD remote.txt downloaded.txt
```

### Performance Validation
```bash
# Run comprehensive tests
make test-performance

# Compare with TCP
make benchmark
```

### Network Failure Simulation
```bash
# Test packet loss
./bin/client-linux-amd64 &
sudo iptables -A OUTPUT -p udp --dport 8080 -j DROP
sleep 10
sudo iptables -D OUTPUT -p udp --dport 8080 -j DROP

# Test recovery
./bin/client-linux-amd64
```

## Security Considerations

- **No Authentication**: Educational implementation only
- **Packet Injection**: Vulnerable to forged packets
- **DDoS Protection**: Basic rate limiting needed
- **Data Integrity**: Checksums provide basic protection

## Future Enhancements

- **Encryption**: Add AES encryption for data
- **Authentication**: Implement challenge-response
- **Compression**: Reduce bandwidth usage
- **Multicast Support**: One-to-many transfers
- **QoS Support**: Differentiated services

## Comparison with Lab 1 (TCP)

| Feature | Lab 1 (TCP) | Lab 2 (UDP) | Improvement |
|---------|----------------|----------------|-------------|
| Reliability | Built-in | Custom implementation | Application-level control |
| Performance | Baseline | 1.5x+ faster | Significant throughput gain |
| Latency | Higher | Lower | No handshake overhead |
| Scalability | Limited | Better | Connectionless nature |
| Complexity | Simpler | More complex | Custom protocol implementation |

## Conclusion

Lab 2 demonstrates that UDP can significantly outperform TCP for file transfer when:

1. **Reliability is implemented at application layer**
2. **Network conditions are favorable**
3. **Buffer sizes are optimally tuned**
4. **Sliding window prevents ACK delays**

The 1.5x performance improvement requirement is achievable through careful protocol design and optimization.
