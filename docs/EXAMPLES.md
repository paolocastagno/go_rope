# Examples and Tutorials

## Quick Start

### 1. Simple Echo Server and Client

#### Server Setup

Create `server_config.toml`:
```toml
[configuration]
IdDevice = "echo-server"
ListenAddress = "0.0.0.0:4433"
BufSize = 10000
Workers = 8
ProcessingTime = "1ms"
Timeout = "60s"

[application]
name = "echo"
init = "InitEcho"
```

Create `server.go`:
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

func InitEcho(conf *toml.Tree) *interface{} {
    fmt.Println("Echo server initialized")

    server.ForwardDecision = func(msg *util.RoPEMessage, sessions *map[string]quic.EarlyConnection, attempt int64) bool {
        // Echo: send back to source
        msg.Destination = msg.Source
        msg.Source = "echo-server"
        return attempt == 0  // Single forward
    }

    server.ForwardSetLastResponse = func(msg *util.RoPEMessage) {}

    return nil
}

func main() {
    srv := &server.Server{}
    cfg := &quic.Config{
        MaxIdleTimeout:     10 * time.Second,
        MaxIncomingStreams: 10000000,
        KeepAlivePeriod:    10 * time.Second,
    }

    if err := server.InitServer(srv, cfg, InitEcho, "server_config.toml"); err != nil {
        log.Fatal(err)
    }

    if err := server.Run(srv, cfg,
        server.ForwardDecision,
        server.ForwardSetLastResponse,
        nil); err != nil {
        log.Fatal(err)
    }
}
```

---

#### Client Setup

Create `client_config.toml`:
```toml
[configuration]
id_device = "echo-client"
destinations = ["localhost:4433"]
MaxConcurrentConnections = 10
TestDuration = "5s"
timeout = "10s"

[application]
name = "echo"
init = "InitEchoClient"
requestSize = 100
responseSize = 100
```

Create `client.go`:
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

var responses = 0
var responseMutex sync.Mutex

func InitEchoClient(conf *toml.Tree) *interface{} {
    fmt.Println("Echo client initialized")

    client.ForwardDecision = func(msg *util.RoPEMessage, destinations []string, app *interface{}) string {
        msg.Source = "echo-client"
        return destinations[0]
    }

    client.ForwardSetLastResponse = func(msg util.RoPEMessage, app *interface{}) {
        if msg.Type == util.Response {
            responseMutex.Lock()
            responses++
            responseMutex.Unlock()
        }
    }

    return nil
}

func main() {
    configFile := flag.String("config", "client_config.toml", "Config file")
    flag.Parse()

    cli := &client.Client{}
    cfg := &quic.Config{
        MaxIdleTimeout:     10 * time.Second,
        MaxIncomingStreams: 10000000,
        KeepAlivePeriod:    10 * time.Second,
    }

    if err := cli.InitClient(cfg, *configFile, InitEchoClient); err != nil {
        log.Fatal(err)
    }

    ticker := time.NewTicker(1 * time.Second)
    defer ticker.Stop()

    var wg sync.WaitGroup
    for {
        select {
        case <-cli.Done:
            wg.Wait()
            fmt.Printf("Received %d responses\n", responses)
            return
        case <-ticker.C:
            wg.Add(1)
            go func() {
                defer wg.Done()
                if err := cli.NewReq(cli.Connections, time.Now().UnixNano()); err != nil {
                    log.Printf("Request error: %v", err)
                }
            }()
        }
    }
}
```

---

### 2. Load Balancing with Probability Routing

Create `client_lb_config.toml`:
```toml
[configuration]
id_device = "lb-client"
destinations = ["server1:4433", "server2:4433", "server3:4433"]
MaxConcurrentConnections = 100
TestDuration = "60s"
timeout = "10s"

[application]
name = "loadbalanced"
init = "InitLB"
requestSize = 1024
responseSize = 1024

[routing]
strategy = "probability"
"server1:4433" = 0.5
"server2:4433" = 0.3
"server3:4433" = 0.2
```

Create `client_lb.go`:
```go
package main

import (
    "fmt"
    "math/rand"
    "time"

    "github.com/paolocastagno/go_rope/pkg/client"
    "github.com/paolocastagno/go_rope/pkg/util"
    "github.com/pelletier/go-toml"
)

type LBApp struct {
    Probabilities map[string]float64
    Destinations  []string
}

func InitLB(conf *toml.Tree) *interface{} {
    app := &LBApp{
        Probabilities: make(map[string]float64),
        Destinations:  []string{"server1:4433", "server2:4433", "server3:4433"},
    }

    // Set probabilities
    app.Probabilities["server1:4433"] = 0.5
    app.Probabilities["server2:4433"] = 0.3
    app.Probabilities["server3:4433"] = 0.2

    client.ForwardDecision = func(msg *util.RoPEMessage, destinations []string, appIface *interface{}) string {
        a := (*appIface).(*LBApp)

        // Weighted random selection
        r := rand.Float64()
        sum := 0.0
        for _, dest := range a.Destinations {
            sum += a.Probabilities[dest]
            if r <= sum {
                return dest
            }
        }
        return a.Destinations[len(a.Destinations)-1]
    }

    client.ForwardSetLastResponse = func(msg util.RoPEMessage, app *interface{}) {
        if msg.Type == util.Response {
            fmt.Printf("Response from: %s\n", msg.Source)
        }
    }

    appIface := new(interface{})
    *appIface = app
    return appIface
}
```

---

### 3. Performance Testing with RTT Measurement

Create `rtt_client_config.toml`:
```toml
[configuration]
id_device = "rtt-test"
destinations = ["server:4433"]
MaxConcurrentConnections = 50
TestDuration = "30s"
timeout = "5s"

[application]
name = "rtt-tester"
init = "InitRTT"
requestSize = 256
responseSize = 256
```

Create `rtt_client.go`:
```go
package main

import (
    "encoding/binary"
    "fmt"
    "time"

    "github.com/paolocastagno/go_rope/pkg/client"
    "github.com/paolocastagno/go_rope/pkg/util"
    "github.com/pelletier/go-toml"
)

var histogram *util.Histogram

func InitRTT(conf *toml.Tree) *interface{} {
    histogram = util.NewHistogram(0.001)  // 1ms bins

    client.ForwardDecision = func(msg *util.RoPEMessage, destinations []string, app *interface{}) string {
        // Embed timestamp in request
        ts := time.Now().UnixNano()
        msg.Body = make([]byte, 8)
        binary.BigEndian.PutUint64(msg.Body, uint64(ts))
        return destinations[0]
    }

    client.ForwardSetLastResponse = func(msg util.RoPEMessage, app *interface{}) {
        if msg.Type == util.Response && len(msg.Body) >= 8 {
            ts := int64(binary.BigEndian.Uint64(msg.Body))
            rtt := time.Since(time.Unix(0, ts))
            histogram.Add(rtt.Seconds())
        }
    }

    return nil
}

func main() {
    // ... initialization code ...

    // At shutdown
    histogram.Print("rtt_results.txt")
    fmt.Println("RTT histogram saved to rtt_results.txt")
}
```

---

### 4. Network Condition Simulation

Monitor and simulate network latency:

```go
package main

import (
    "fmt"
    "time"

    "github.com/paolocastagno/go_rope/pkg/client"
    "github.com/paolocastagno/go_rope/pkg/util"
    "github.com/pelletier/go-toml"
)

func InitNetworkSimulator(conf *toml.Tree) *interface{} {
    // Initialize delay distribution
    delayDist := util.InitDelay("exponential", "50ms")

    client.ForwardDecision = func(msg *util.RoPEMessage, destinations []string, app *interface{}) string {
        // Apply simulated delay before sending
        util.Delay(delayDist, "exponential")
        return destinations[0]
    }

    return nil
}
```

---

## Running Examples

### Terminal 1 - Start Server
```bash
go run examples/server/main.go
```

### Terminal 2 - Start Client
```bash
go run examples/client/main.go -config examples/client/config.toml
```

---

## Monitoring and Analysis

### Collecting Metrics

```go
// Create matrix for per-destination RTT
rttnMatrix := util.NewMatrix(0.01)  // 10ms bins
rttnMatrix.AddColumn(0, "Server1")
rttnMatrix.AddColumn(1, "Server2")
rttnMatrix.AddColumn(2, "Server3")

// Add measurements
for i := 0; i < 1000; i++ {
    rtt1 := measureRTT("server1:4433")
    rtt2 := measureRTT("server2:4433")
    rtt3 := measureRTT("server3:4433")

    rttnMatrix.Add(rtt1, 0)
    rttnMatrix.Add(rtt2, 1)
    rttnMatrix.Add(rtt3, 2)
}

// Output results
rttnMatrix.Print("rtt_by_destination.txt")
```

---

## Advanced Patterns

### Custom Routing with State

```go
type AdvancedRouter struct {
    FailoverCount map[string]int
    LastError     map[string]error
}

func InitAdvancedRouter(conf *toml.Tree) *interface{} {
    router := &AdvancedRouter{
        FailoverCount: make(map[string]int),
        LastError:     make(map[string]error),
    }

    client.ForwardDecision = func(msg *util.RoPEMessage, destinations []string, app *interface{}) string {
        r := (*app).(*AdvancedRouter)

        // Select destination with fewest failures
        best := destinations[0]
        for _, dest := range destinations {
            if r.FailoverCount[dest] < r.FailoverCount[best] {
                best = dest
            }
        }
        return best
    }

    client.ForwardSetLastResponse = func(msg util.RoPEMessage, app *interface{}) {
        r := (*app).(*AdvancedRouter)
        if msg.Type != util.Response {
            r.FailoverCount[msg.Source]++
        }
    }

    appIface := new(interface{})
    *appIface = router
    return appIface
}
```
