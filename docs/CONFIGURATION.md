# Configuration Guide

## Configuration Structure

go_rope uses TOML format for configuration. Configuration is organized into sections:

## Configuration Sections

### 1. Configuration Section

Main settings for client or server behavior.

#### Client Configuration

```toml
[configuration]
# Unique identifier for this client instance
id_device = "client-1"

# Server destinations (load-balanced across these)
destinations = ["server1.example.com:4433", "server2.example.com:4433"]

# Maximum number of concurrent connections
MaxConcurrentConnections = 100

# Total duration for the test (0 = unlimited)
TestDuration = "60s"

# Timeout for individual requests
timeout = "30s"
```

**Parameters:**
- `id_device` (string, required): Unique client identifier
- `destinations` (array, required): List of server addresses
- `MaxConcurrentConnections` (integer, default: 1): Concurrent request limit
- `TestDuration` (duration, default: "0s"): Test runtime (0 = no limit)
- `timeout` (duration, default: "30s"): Request timeout

---

#### Server Configuration

```toml
[configuration]
# Unique identifier for this server instance
IdDevice = "server-1"

# Address to listen on
ListenAddress = "0.0.0.0:4433"

# Buffer size for job queue
BufSize = 10000

# Number of worker goroutines
Workers = 8

# Time spent processing each request
ProcessingTime = "10ms"

# Connection timeout
Timeout = "60s"
```

**Parameters:**
- `IdDevice` (string, required): Unique server identifier
- `ListenAddress` (string, required): Bind address and port
- `BufSize` (integer, required): Job queue capacity
- `Workers` (integer, required): Number of worker threads
- `ProcessingTime` (duration, required): Artificial processing delay
- `Timeout` (duration, required): Connection timeout

---

### 2. Application Section

Application-specific configuration passed to custom logic.

```toml
[application]
# Application name/type
name = "fixed"

# Initialization function name
init = "InitFixed"

# Destination servers
destinations = ["server1.example.com:4433"]

# Request payload size (bytes)
requestSize = 1024

# Expected response size (bytes)
responseSize = 2048
```

**Common Parameters:**
- `name` (string): Application identifier
- `init` (string): Initialization function name
- `destinations` (array): Application-level destination list
- `requestSize` (integer): Request body size
- `responseSize` (integer): Response body size

---

### 3. Logger Section

Logging configuration.

```toml
[logger]
# Enable logging
enabled = true

# Log level (debug, info, warn, error)
level = "info"

# Log file path (empty = stdout)
output = "logs/go_rope.log"

# Timestamp format
timestamp_format = "2006-01-02T15:04:05"
```

---

### 4. Routing Section

Router-specific configuration.

```toml
[routing]
# Routing strategy (fixed, probability, probabilityLatency)
strategy = "fixed"

# Variables for the selected strategy
[routing.variables]
dest = "server.example.com:4433"

# Destination definitions
[routing.destinations]
srv1 = "server1.example.com:4433"
srv2 = "server2.example.com:4433"

# Probability weights (for probability-based routing)
[routing.probabilities]
srv1 = 0.7
srv2 = 0.3
```

---

## Complete Example Configurations

### Minimal Client Configuration

```toml
[configuration]
id_device = "test-client"
destinations = ["localhost:4433"]
MaxConcurrentConnections = 10
TestDuration = "10s"
timeout = "5s"

[application]
name = "fixed"
init = "InitFixed"
destinations = ["localhost:4433"]
requestSize = 100
responseSize = 100
```

### Complete Server Configuration

```toml
[configuration]
IdDevice = "test-server"
ListenAddress = "0.0.0.0:4433"
BufSize = 50000
Workers = 16
ProcessingTime = "5ms"
Timeout = "120s"

[application]
name = "handler"
init = "InitHandler"

[logger]
enabled = true
level = "info"
output = ""
```

### Multi-Destination Client with Probability Routing

```toml
[configuration]
id_device = "load-test"
destinations = [
    "server1.example.com:4433",
    "server2.example.com:4433",
    "server3.example.com:4433",
]
MaxConcurrentConnections = 500
TestDuration = "300s"
timeout = "10s"

[application]
name = "distributed"
init = "InitDistributed"
requestSize = 2048
responseSize = 1024

[routing]
strategy = "probability"

[routing.probabilities]
"server1.example.com:4433" = 0.5
"server2.example.com:4433" = 0.3
"server3.example.com:4433" = 0.2
```

---

## Duration Format

Duration values use Go's time format:
- `100ms` - 100 milliseconds
- `1s` - 1 second
- `10s` - 10 seconds
- `1m` - 1 minute
- `1h` - 1 hour

---

## Configuration Best Practices

1. **Buffer Size**: Set `BufSize` to 10x the expected concurrent requests
2. **Workers**: Use number of CPU cores for best performance
3. **Timeouts**: Set slightly higher than expected RTT + processing time
4. **Processing Time**: Match actual server processing time for realistic testing
5. **Test Duration**: Allow time for warm-up (first 10% often anomalous)
6. **Queue Depth**: Monitor for "Queue full" messages; increase `BufSize` if frequent

---

## Environment Variables

You can override configuration values with environment variables:

```bash
ROPE_BUFSIZE=100000 ROPE_WORKERS=32 ./server
```

---

## Configuration Validation

The framework performs validation:
- Required fields must be present
- Duration values must be valid
- Numeric values must be in valid ranges
- Array values cannot be empty where required

Loading invalid configuration will result in an error during initialization.
