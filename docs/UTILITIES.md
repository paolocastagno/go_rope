# Utilities Guide

## Performance Monitoring

### Histogram

A thread-safe histogram for collecting and analyzing distribution of values (e.g., RTT measurements).

#### Creating a Histogram

```go
import "github.com/paolocastagno/go_rope/pkg/util"

// Create histogram with 10ms bins
hist := util.NewHistogram(0.01)  // binSize in seconds
```

#### Adding Data

```go
// Add a measurement (in seconds)
hist.Add(0.125)  // 125ms measurement
hist.Add(0.050)  // 50ms measurement
hist.Add(0.150)  // 150ms measurement
```

#### Output

```go
// Print to stdout
hist.Print()

// Print to file
hist.Print("output.txt")
```

**Output Format:**
```
bin, count
0, 5
1, 12
2, 8
3, 1
```

Where bin 0 contains values [0.0, 0.01), bin 1 contains [0.01, 0.02), etc.

---

### 2D Histogram (Matrix)

Collect metrics across multiple dimensions (e.g., RTT per destination).

#### Creating a Matrix

```go
// Create matrix with 10ms bins
matrix := util.NewMatrix(0.01)

// Add column names
matrix.AddColumn(0, "Destination-1")
matrix.AddColumn(1, "Destination-2")
matrix.AddColumn(2, "Destination-3")
```

#### Adding Data

```go
// Add measurement to row (value/binSize) and column (destination_index)
matrix.Add(0.125, 0)  // 125ms to destination 1
matrix.Add(0.050, 1)  // 50ms to destination 2
matrix.Add(0.075, 2)  // 75ms to destination 3
```

#### Output

```go
// Print to stdout
matrix.Print()

// Print to file
matrix.Print("matrix_output.txt")
```

**Output Format:**
```
Bin\Col  Destination-1  Destination-2  Destination-3
0        2              1              0
1        5              8              3
2        3              2              4
```

---

### Moving Average (Mavg)

Efficient sliding window average for bandwidth tracking.

#### Creating a Moving Average

```go
// Create with window size of 60 samples
mavg := util.NewMavg(60)
```

#### Adding Values

```go
// Add value to the moving average
util.Mavg_push(&mavg, 1024)  // Add 1024 bytes
util.Mavg_push(&mavg, 2048)  // Add 2048 bytes
```

#### Evaluating Average

```go
// Get average with window normalization
avg := util.Mavg_eval(mavg, 1) // Returns avg bytes per sample

// For bandwidth: if each sample represents 1 second
bandwidth := util.Mavg_eval(mavg, int64(10*time.Second/time.Second)) // bytes/sec
```

#### Example: Bandwidth Tracking

```go
const observationWindow = 10 * time.Second
const timeWindow int64 = 60  // samples

var bUp, bDown util.Mavg = util.NewMavg(timeWindow), util.NewMavg(timeWindow)
var startTime = time.Now()

// Every observationWindow interval
if time.Since(startTime) > observationWindow {
    startTime = time.Now()
    util.Mavg_push(&bUp, uploadBytes)
    util.Mavg_push(&bDown, downloadBytes)

    upBW := util.Mavg_eval(bUp, 10)      // bytes/second
    downBW := util.Mavg_eval(bDown, 10)  // bytes/second

    fmt.Printf("Up: %.2f Mbps, Down: %.2f Mbps\n",
        upBW*8/1e6, downBW*8/1e6)
}
```

---

## Network Utilities

### Ping Testing

#### Running Ping Test

```go
import (
    "sync"
    "github.com/paolocastagno/go_rope/pkg/util"
)

var wg sync.WaitGroup
latencyChan := make(chan float64)

// Run ping test
util.PingTest(
    "example.com:4433",  // target address
    "upstream",          // direction label
    30*time.Second,      // test duration
    &wg,                 // wait group
    latencyChan,         // results channel
    "device-1",          // device ID for logging
)

// Wait for results
<-latencyChan  // Blocking receive of latency (ms)
```

---

### Delay Injection

#### Available Distributions

**1. Uniform Distribution**
```go
dist := util.InitDelay("uniform", []interface{}{"10ms", "100ms"})

// Apply delay
for i := 0; i < 1000; i++ {
    util.Delay(dist, "uniform")
    // Random delay between 10-100ms
}
```

**2. Exponential Distribution**
```go
dist := util.InitDelay("exponential", "50ms")

// Apply delay
util.Delay(dist, "exponential")
// Average delay: 50ms
```

**3. Constant Delay**
```go
dist := util.InitDelay("constant", "25ms")

// Apply delay
util.Delay(dist, "constant")
// Fixed 25ms delay
```

**4. No Delay**
```go
dist := util.InitDelay("no delay", nil)

util.Delay(dist, "no delay")
// No artificial delay
```

#### Integration with Requests

```go
func SendRequestWithDelay(client *client.Client, delayDist interface{}) {
    // Apply delay
    util.Delay(delayDist, "exponential")

    // Send request
    if err := client.NewReq(client.Connections, requestID); err != nil {
        log.Printf("Request failed: %v", err)
    }
}
```

---

### ZMQ Integration (Modem Information)

#### Retrieving Modem Status

```go
import "github.com/paolocastagno/go_rope/pkg/util"

// Get modem information from ZMQ publisher
util.ZmqInfo("tcp://modem-publisher:5555", "device-1")
```

#### Modem Information Structure

```go
type ZmqModem struct {
    InternalInterface string  // e.g., "wwan0"
    Operator          string  // e.g., "Vodafone"
    IPAddress         string  // e.g., "10.0.0.1"
    Frequency         uint    // e.g., 2600 (MHz)
    RSSI              int     // Signal strength (dBm), typically -120 to -25
}
```

---

## TLS Configuration

### Generating Self-Signed Certificates

```go
import (
    "github.com/paolocastagno/go_rope/pkg/util"
    "github.com/quic-go/quic-go"
)

// Generate TLS config
tlsConfig := util.GenerateTLSConfig()

// Use with server
listener, err := quic.ListenAddrEarly(
    "0.0.0.0:4433",
    tlsConfig,
    quicConf,
)

// Use with client
conn, err := quic.DialAddrEarly(
    ctx,
    "server.example.com:4433",
    &tls.Config{InsecureSkipVerify: true},  // For clients, bypass verification
    quicConf,
)
```

---

## Message Types

### RoPEMessage Type Constants

```go
const (
    Request        = "Request"         // New request from client
    Response       = "Response"        // Response from server
    QueueFull      = "QueueFull"       // Server queue full
    ServerNotFound = "ServerNotFound"  // Destination unreachable
    ServerTimeout  = "ServerTimeout"   // Request timeout
    NoRoute        = "NoRoute"         // No route available
    MessageLost    = "MessageLost"     // Packet loss occurred
)
```

### Message Creation Examples

#### Client Request

```go
msg := util.RoPEMessage{
    ReqID:       "client1_12345",
    Type:        util.Request,
    Log:         true,
    Body:        []byte("request payload"),
    ResSize:     1024,
    Source:      "client-id",
    Destination: "server.example.com:4433",
    Timestamp:   time.Now(),
}
```

#### Server Response

```go
response := util.RoPEMessage{
    ReqID:       req.ReqID,
    Type:        util.Response,
    Body:        responseData,
    ResSize:     int32(len(responseData)),
    Source:      "server-id",
    Destination: req.Source,
    Timestamp:   time.Now(),
}
```

---

## Graceful Shutdown

### Setup Shutdown Handler

```go
import "github.com/paolocastagno/go_rope/pkg/util"

// Register shutdown handler
util.SetupGracefulShutdown(func() {
    fmt.Println("Shutting down...")
    util.CloseLogger()
    os.Exit(0)
})
```

---

## Logging Integration

### Configuration via TOML

```toml
[logger]
enabled = true
level = "info"
output = "logs/app.log"
```

### In Code

```go
if serverConfig.LoggerEnabled {
    util.SetLoggerParamFromConf(loggerConfig)
}

// Logger integration happens automatically with LoggerEnabled flag
```

---

## Best Practices

1. **Histogram Bin Size**: Choose bins appropriate for your measurement:
   - RTT: 1-10ms bins
   - Bandwidth: 1MB bins

2. **Moving Average Window**: Larger windows = smoother but slower updates
   - Real-time: 10-60 samples
   - Reporting: 100-300 samples

3. **Delay Injection**: Match real network observations
   - Use exponential for WAN links
   - Use uniform for known constraints

4. **Thread Safety**: Histograms and matrices are thread-safe
   - Safe to call `Add()` from multiple goroutines
   - `Print()` also thread-safe

5. **Memory**: Histograms use only active bins
   - Sparse storage for distributed values
