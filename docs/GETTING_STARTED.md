# Getting Started Guide

Welcome to go_rope! This guide will get you up and running in minutes.

## Prerequisites

- Go 1.23 or later
- Basic knowledge of Go
- A text editor or IDE

## 5-Minute Setup

### 1. Install go_rope

```bash
go get github.com/paolocastagno/go_rope
```

### 2. Create a Simple Server

Save as `server.go`:

```go
package main

import (
    "fmt"
    "log"
    "time"

    "github.com/paolocastagno/go_rope/pkg/server"
    "github.com/paolocastagno/go_rope/pkg/util"
    "github.com/pelletier/go-toml"
    "github.com/quic-go/quic-go"
)

func InitServer(conf *toml.Tree) *interface{} {
    fmt.Println("✓ Server initialized")

    // Echo responses back
    server.ForwardDecision = func(msg *util.RoPEMessage, _ *map[string]quic.EarlyConnection, _ int64) bool {
        msg.Destination = msg.Source
        msg.Source = "server"
        return false
    }

    server.ForwardSetLastResponse = func(_ *util.RoPEMessage) {}
    return nil
}

func main() {
    srv := &server.Server{}
    quicConf := &quic.Config{MaxIdleTimeout: 10 * time.Second}

    if err := server.InitServer(srv, quicConf, InitServer, "server.toml"); err != nil {
        log.Fatal(err)
    }

    if err := server.Run(srv, quicConf, server.ForwardDecision, nil, nil); err != nil {
        log.Fatal(err)
    }
}
```

### 3. Create a Simple Client

Save as `client.go`:

```go
package main

import (
    "flag"
    "fmt"
    "log"
    "sync"
    "time"

    "github.com/paolocastagno/go_rope/pkg/client"
    "github.com/paolocastagno/go_rope/pkg/util"
    "github.com/pelletier/go-toml"
    "github.com/quic-go/quic-go"
)

var count = 0
var mu sync.Mutex

func InitClient(conf *toml.Tree) *interface{} {
    fmt.Println("✓ Client initialized")

    client.ForwardDecision = func(msg *util.RoPEMessage, dests []string, _ *interface{}) string {
        msg.Source = "client"
        return dests[0]
    }

    client.ForwardSetLastResponse = func(_ util.RoPEMessage, _ *interface{}) {
        mu.Lock()
        count++
        mu.Unlock()
    }

    return nil
}

func main() {
    configFile := flag.String("config", "client.toml", "Config file")
    flag.Parse()

    cli := &client.Client{}
    quicConf := &quic.Config{MaxIdleTimeout: 10 * time.Second}

    if err := cli.InitClient(quicConf, *configFile, InitClient); err != nil {
        log.Fatal(err)
    }

    ticker := time.NewTicker(100 * time.Millisecond)
    defer ticker.Stop()

    var wg sync.WaitGroup
    for {
        select {
        case <-cli.Done:
            wg.Wait()
            fmt.Printf("✓ Received %d responses\n", count)
            return
        case <-ticker.C:
            wg.Add(1)
            go func() {
                defer wg.Done()
                cli.NewReq(cli.Connections, time.Now().UnixNano())
            }()
        }
    }
}
```

### 4. Create Configuration Files

**server.toml:**
```toml
[configuration]
IdDevice = "server-1"
ListenAddress = "0.0.0.0:4433"
BufSize = 10000
Workers = 8
ProcessingTime = "1ms"
Timeout = "60s"

[application]
name = "echo"
init = "InitServer"
```

**client.toml:**
```toml
[configuration]
id_device = "client-1"
destinations = ["localhost:4433"]
MaxConcurrentConnections = 10
TestDuration = "10s"
timeout = "5s"

[application]
name = "test"
init = "InitClient"
```

### 5. Run It!

**Terminal 1 - Start Server:**
```bash
go run server.go
# Output: ✓ Server initialized
#         Server ready 0.0.0.0:4433, workers 8, queue length=10000
```

**Terminal 2 - Start Client:**
```bash
go run client.go -config client.toml
# Output: ✓ Client initialized
#         Test duration set to 10s
#         Connected to localhost:4433
#         ... (requests running) ...
#         ✓ Received 100 responses
```

Congratulations! You have a working go_rope application! 🎉

---

## Next Steps

### 1. Explore Examples
Check out the `examples/` directory for more sophisticated patterns:
- Multi-destination load balancing
- Performance metrics collection
- Network simulation
- Custom routing logic

### 2. Add Performance Monitoring

Collect RTT measurements:

```go
import "github.com/paolocastagno/go_rope/pkg/util"

histogram := util.NewHistogram(0.001)  // 1ms bins

// In your response handler:
histogram.Add(rttInSeconds)

// At exit:
histogram.Print("results.txt")
```

### 3. Implement Custom Routing

Try probability-based routing across multiple servers:

```go
destinations := []string{"srv1:4433", "srv2:4433", "srv3:4433"}

client.ForwardDecision = func(msg *util.RoPEMessage, dests []string, _ *interface{}) string {
    // Route 50% to srv1, 30% to srv2, 20% to srv3
    if rand.Float64() < 0.5 {
        return dests[0]
    } else if rand.Float64() < 0.6 {
        return dests[1]
    }
    return dests[2]
}
```

### 4. Read the Full Documentation

Dive deeper:
- **[ARCHITECTURE.md](ARCHITECTURE.md)** - System design
- **[CONFIGURATION.md](CONFIGURATION.md)** - All config options
- **[API.md](API.md)** - Complete API reference
- **[EXAMPLES.md](EXAMPLES.md)** - Advanced examples

---

## Common Patterns

### Pattern 1: Simple Request-Response

See the 5-minute setup above.

---

### Pattern 2: Collect Bandwidth Metrics

```go
const obsWindow = 10 * time.Second
const timeWindow int64 = 60

var bUp, bDown = util.NewMavg(timeWindow), util.NewMavg(timeWindow)
var startTime = time.Now()

// In your metrics update code:
if time.Since(startTime) > obsWindow {
    startTime = time.Now()
    util.Mavg_push(&bUp, bytesSent)
    util.Mavg_push(&bDown, bytesReceived)

    fmt.Printf("Up: %.2f MB/s, Down: %.2f MB/s\n",
        util.Mavg_eval(bUp, 10)/1e6,
        util.Mavg_eval(bDown, 10)/1e6)
}
```

---

### Pattern 3: Load Testing

```go
// client.toml
[configuration]
id_device = "load-test"
destinations = ["server:4433"]
MaxConcurrentConnections = 1000
TestDuration = "300s"
timeout = "10s"

# Run: go run loadtest_client.go -config client.toml
```

---

## Troubleshooting

### "connection refused"
- Verify server is running
- Check port number in config (default: 4433)

### "configuration file not found"
- Ensure `.toml` files exist in current directory
- Use absolute paths if needed

### "queue full messages"
- Increase `BufSize` in server config
- Reduce `MaxConcurrentConnections` in client

### "slow performance"
- Check `Workers` count (should match CPU cores)
- Monitor `ProcessingTime` setting
- Review `TestDuration` (results after warm-up)

---

## Quick Configuration Cheat Sheet

### Server Parameters

```toml
[configuration]
IdDevice = "server-1"            # Unique identifier
ListenAddress = "0.0.0.0:4433"   # Bind address
BufSize = 10000                  # Job queue depth
Workers = 8                      # Worker goroutines (use CPU count)
ProcessingTime = "1ms"           # Processing delay
Timeout = "60s"                  # Connection timeout
```

### Client Parameters

```toml
[configuration]
id_device = "client-1"                  # Unique identifier
destinations = ["server:4433"]          # Server addresses
MaxConcurrentConnections = 100          # Connection limit
TestDuration = "60s"                    # Test duration (0 = unlimited)
timeout = "5s"                          # Request timeout
```

---

## Where to Go From Here

| Goal | Resource |
|------|----------|
| Understand architecture | [ARCHITECTURE.md](ARCHITECTURE.md) |
| Configure application | [CONFIGURATION.md](CONFIGURATION.md) |
| Use API functions | [API.md](API.md) |
| Add monitoring | [UTILITIES.md](UTILITIES.md) |
| See examples | [EXAMPLES.md](EXAMPLES.md) |
| Custom routing | [ROUTING.md](ROUTING.md) |

---

## Getting Help

1. **Documentation**: Check [INDEX.md](INDEX.md) for quick navigation
2. **Examples**: Review `examples/` directory
3. **API Docs**: See [API.md](API.md) for function signatures
4. **Configuration**: Reference [CONFIGURATION.md](CONFIGURATION.md)

---

Happy coding! 🚀

For more information, see [INDEX.md](INDEX.md) for complete documentation navigation.
