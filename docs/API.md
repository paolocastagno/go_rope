# API Reference

## Client Package (`pkg/client`)

### Types

#### Client

```go
type Client struct {
    IdDevice                 string                  // Unique device identifier
    Destinations             []string                // Target server addresses
    MaxConcurrentConnections uint                    // Connection limit
    Done                     chan bool               // Completion signal
    TestDuration             time.Duration           // Test execution duration
    Timeout                  time.Duration           // Request timeout
    CfgFile                  string                  // Configuration file path
    LoggerEnabled            bool                    // Logging flag
    Connections              []quic.EarlyConnection // QUIC connections
    Counter                  chan int64              // Request counter
    Application              *interface{}            // Application state
}
```

### Methods

#### InitClient

```go
func (client *Client) InitClient(
    quicConf *quic.Config,
    CfgFile string,
    initLogic func(*toml.Tree) *interface{},
) error
```

Initializes the client with configuration and establishes QUIC connections.

**Parameters:**
- `quicConf`: QUIC protocol configuration
- `CfgFile`: Path to TOML configuration file
- `initLogic`: Application initialization callback

**Returns:** Error if initialization fails

**Example:**
```go
cli := &client.Client{}
quicConf := &quic.Config{
    MaxIdleTimeout:     10 * time.Second,
    MaxIncomingStreams: 10000000,
    KeepAlivePeriod:    10 * time.Second,
}
if err := cli.InitClient(quicConf, "config.toml", initFunc); err != nil {
    log.Fatal(err)
}
```

---

#### NewReq

```go
func (client *Client) NewReq(
    session []quic.EarlyConnection,
    id int64,
) error
```

Creates and sends a new request to a destination.

**Parameters:**
- `session`: Array of QUIC connections
- `id`: Unique request identifier

**Returns:** Error if request fails

---

#### PrintParams

```go
func (client *Client) PrintParams()
```

Prints current client configuration parameters.

---

### Functions

#### GetString

```go
func GetString(
    cfg map[string]interface{},
    key string,
    defaultValue string,
) string
```

Retrieves a string value from configuration map with default fallback.

---

#### GetFloat64

```go
func GetFloat64(
    cfg map[string]interface{},
    key string,
    defaultValue float64,
) float64
```

Retrieves a float64 value from configuration map.

---

#### GetDuration

```go
func GetDuration(
    cfg map[string]interface{},
    key string,
    defaultValue string,
) time.Duration
```

Parses and retrieves a duration value from configuration.

---

## Server Package (`pkg/server`)

### Types

#### Server

```go
type Server struct {
    IdDevice       string            // Unique device identifier
    ListenAddr     string            // Bind address and port
    BufSize        int64             // Job queue buffer size
    Workers        int64             // Number of worker goroutines
    ProcessingTime time.Duration     // Processing delay per request
    Timeout        time.Duration     // Connection timeout
    Application    *interface{}      // Application state
    LoggerEnabled  bool              // Logging flag
}
```

#### JobRequest

```go
type JobRequest struct {
    Request    util.RoPEMessage     // Request message
    QuicStream quic.Stream          // Associated stream
}
```

### Functions

#### InitServer

```go
func InitServer(
    server *Server,
    quicConf *quic.Config,
    initLogic func(*toml.Tree) *interface{},
    CfgFile string,
) error
```

Initializes the server with configuration.

**Parameters:**
- `server`: Server instance to configure
- `quicConf`: QUIC protocol configuration
- `initLogic`: Application initialization callback
- `CfgFile`: Path to TOML configuration file

---

#### Run

```go
func Run(
    server *Server,
    quicConf *quic.Config,
    ForwardDecision func(*util.RoPEMessage, *map[string]quic.EarlyConnection, int64) bool,
    ForwardSetLastResponse func(*util.RoPEMessage),
    ForwardBlock func(*util.RoPEMessage, quic.Stream),
) error
```

Starts the server listener and worker pool.

**Parameters:**
- `server`: Configured server instance
- `quicConf`: QUIC protocol configuration
- `ForwardDecision`: Callback for routing decisions
- `ForwardSetLastResponse`: Callback for response handling
- `ForwardBlock`: Callback when queue is full

---

## Utility Package (`pkg/util`)

### Message Types

#### RoPEMessage

```go
type RoPEMessage struct {
    ReqID       string      // Request identifier
    Type        RoPEMsgType // Message type
    Log         bool        // Enable logging
    ResSize     int32       // Response size
    Body        []byte      // Payload
    Source      string      // Source node
    Hop         string      // Current hop
    Destination string      // Target destination
    Timestamp   time.Time   // Message timestamp
}
```

#### RoPEMsgType

Message type constants:
- `Request`: Client request
- `Response`: Server response
- `QueueFull`: Server queue full error
- `ServerNotFound`: Destination not found
- `ServerTimeout`: Request timeout
- `NoRoute`: No route available
- `MessageLost`: Packet loss indicator

---

### Performance Monitoring

#### Histogram

Thread-safe histogram for binned data collection.

```go
func NewHistogram(binSize float64) *Histogram

func (h *Histogram) Add(value float64)

func (h *Histogram) Print(args ...string)  // Print to file or stdout
```

**Example:**
```go
hist := util.NewHistogram(0.01)  // 10ms bins
hist.Add(0.125)  // Add 125ms measurement
hist.Print("output.txt")  // Save to file
```

---

#### Histogram Matrix

2D histogram for multi-dimensional metrics.

```go
func NewMatrix(binSize float64) *Matrix

func (m *Matrix) Add(value float64, col int)

func (m *Matrix) AddColumn(col int, name string)

func (m *Matrix) Print(args ...string)
```

---

#### Moving Average (Mavg)

Efficient sliding window average.

```go
func NewMavg(size int64) Mavg

func Mavg_push(x *Mavg, y int64)

func Mavg_eval(x Mavg, window int64) float64
```

**Example:**
```go
mavg := util.NewMavg(60)  // 60-sample window
util.Mavg_push(&mavg, 1000)
avg := util.Mavg_eval(mavg, 1)
```

---

### Network Utilities

#### PingTest

```go
func PingTest(
    pingAddr string,
    direction string,
    duration time.Duration,
    wgPing *sync.WaitGroup,
    c chan float64,
    idDevice string,
)
```

Performs ping test and returns latency.

---

#### Delay

```go
func InitDelay(dist string, dist_params interface{}) interface{}

func Delay(distribution interface{}, distType string)
```

Supports delay injection with distributions:
- `uniform`: Random between min/max
- `exponential`: Exponential distribution
- `constant`: Fixed delay
- `no delay`: No injection

---

#### TLS Configuration

```go
func GenerateTLSConfig() *tls.Config
```

Generates self-signed TLS certificate for QUIC.

---

### ZMQ Integration

#### ZmqInfo

```go
func ZmqInfo(zmqAddr string, idDevice string)

type ZmqModem struct {
    InternalInterface string  // Network interface
    Operator          string  // Mobile operator
    IPAddress         string  // IP address
    Frequency         uint    // Signal frequency
    RSSI              int     // Signal strength
}
```

Retrieves modem information from ZMQ publisher.

---

## Routing Package (`pkg/routing`)

These are framework extension points called by your application:

### Routing Decision Functions

```go
type ForwardDecision func(
    msg *util.RoPEMessage,
    destinations []string,
    application *interface{},
) string
```

Returns destination address for routing.

---

```go
type ForwardSetLastResponse func(
    lastResp util.RoPEMessage,
    application *interface{},
)
```

Processes response and updates metrics.

---

## Configuration Package

### Functions

```go
func LoadConfig(filename string) (*toml.Tree, error)

func ParseClientConfig(tree *toml.Tree) (*ClientConfig, error)

func ParseServerConfig(tree *toml.Tree) (*ServerConfig, error)
```

---

## Common Patterns

### Custom Application Logic

```go
type MyApp struct {
    Config map[string]interface{}
}

func InitMyApp(conf *toml.Tree) *interface{} {
    app := &MyApp{
        Config: conf.ToMap(),
    }

    client.ForwardDecision = func(msg *util.RoPEMessage, dests []string, app *interface{}) string {
        return myApp.DecideDestination(msg, dests)
    }

    client.ForwardSetLastResponse = func(resp util.RoPEMessage, app *interface{}) {
        myApp.HandleResponse(resp)
    }

    appInterface := new(interface{})
    *appInterface = app
    return appInterface
}
```

### Error Handling

```go
if err := cli.InitClient(quicConf, timeout, initFunc); err != nil {
    log.Fatalf("Client init failed: %v", err)
}

if err := cli.NewReq(cli.Connections, reqID); err != nil {
    log.Printf("Request error: %v", err)
    // Handle gracefully or retry
}
```
