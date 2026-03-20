# go_rope

**A high-performance QUIC-based network framework for distributed systems and performance testing.**

`go_rope` is a production-ready Go package designed for building scalable network applications with support for multiple routing strategies, comprehensive performance monitoring, and flexible configuration management.

## Features

- **🚀 QUIC Protocol**: Low-latency, multiplexed network communication
- **🔀 Advanced Routing**: Fixed, probability-based, and latency-aware routing strategies
- **📊 Performance Monitoring**: Built-in histograms, moving averages, and matrix metrics
- **⚙️ Flexible Configuration**: TOML-based configuration for easy deployment
- **🧵 Concurrent Processing**: Efficient goroutine-based concurrency model
- **🔬 Network Simulation**: Support for latency injection and delay distributions
- **📡 Extensible Architecture**: Custom routing and application logic hooks
- **🛠️ Comprehensive Utilities**: Ping testing, ZMQ integration, TLS certificate generation

## Quick Start

### Installation

```sh
go get github.com/paolocastagno/go_rope
```

### Minimal Example

**Server (`server.go`):**
```go
package main

import (
    "log"
    "time"
    "github.com/paolocastagno/go_rope/pkg/server"
    "github.com/pelletier/go-toml"
    "github.com/quic-go/quic-go"
)

func InitServer(conf *toml.Tree) *interface{} {
    return nil
}

func main() {
    srv := &server.Server{}
    quicConf := &quic.Config{
        MaxIdleTimeout: 10 * time.Second,
        MaxIncomingStreams: 10000000,
    }

    if err := server.InitServer(srv, quicConf, InitServer, "config.toml"); err != nil {
        log.Fatal(err)
    }

    if err := server.Run(srv, quicConf, nil, nil, nil); err != nil {
        log.Fatal(err)
    }
}
```

**Client (`client.go`):**
```go
package main

import (
    "log"
    "time"
    "github.com/paolocastagno/go_rope/pkg/client"
    "github.com/pelletier/go-toml"
    "github.com/quic-go/quic-go"
)

func InitClient(conf *toml.Tree) *interface{} {
    return nil
}

func main() {
    cli := &client.Client{}
    quicConf := &quic.Config{
        MaxIdleTimeout: 10 * time.Second,
        MaxIncomingStreams: 10000000,
    }

    if err := cli.InitClient(quicConf, "config.toml", InitClient); err != nil {
        log.Fatal(err)
    }

    for range time.NewTicker(1 * time.Second).C {
        if err := cli.NewReq(cli.Connections, 1); err != nil {
            log.Printf("Request error: %v", err)
        }
    }
}
```

## Documentation

Complete documentation is available in the `docs/` directory:

| Document | Description |
|----------|-------------|
| **[ARCHITECTURE.md](docs/ARCHITECTURE.md)** | System design, components, and communication flow |
| **[ROUTING.md](docs/ROUTING.md)** | Routing strategies and custom implementation |
| **[CONFIGURATION.md](docs/CONFIGURATION.md)** | Configuration guide and TOML reference |
| **[API.md](docs/API.md)** | Complete API reference for all packages |
| **[UTILITIES.md](docs/UTILITIES.md)** | Utility functions guide (histograms, monitoring, etc.) |
| **[EXAMPLES.md](docs/EXAMPLES.md)** | Runnable examples and advanced patterns |

## Project Structure

```
go_rope/
├── pkg/
│   ├── client/          # QUIC client implementation
│   ├── server/          # QUIC server implementation
│   ├── routing/         # Routing strategies (fixed, probability, latency-aware)
│   ├── util/            # Utilities (metrics, networking, TLS)
│   └── config/          # Configuration management
├── examples/
│   ├── client/          # Example client implementations
│   └── server/          # Example server implementations
├── docs/                # Complete documentation
└── go.mod              # Dependencies
```

## Configuration

Configuration uses TOML format. Example `config.toml`:

```toml
[configuration]
id_device = "client-1"
destinations = ["server.example.com:4433"]
MaxConcurrentConnections = 100
TestDuration = "60s"
timeout = "30s"

[application]
name = "fixed"
init = "InitFixed"
requestSize = 1024
responseSize = 1024
```

See [CONFIGURATION.md](docs/CONFIGURATION.md) for complete reference.

## Use Cases

- **Performance Testing**: Benchmark network applications with realistic load
- **Load Balancing**: Distribute requests across multiple servers
- **Network Condition Simulation**: Test application behavior under various network conditions
- **Distributed Systems**: Build scalable, multi-node applications
- **Real-time Analytics**: Collect and monitor network metrics
- **Research**: Network protocol research and experimentation

## Key Components

### 1. Client (`pkg/client`)
- Connects to multiple destinations
- Sends requests and collects responses
- Supports test duration limits
- Manages concurrent connections

### 2. Server (`pkg/server`)
- Accepts QUIC connections
- Processes requests with worker pool
- Implements job queue for backpressure
- Customizable request handling

### 3. Routing (`pkg/routing`)
- **Fixed**: Single destination
- **Probability**: Weighted random selection
- **Latency-Aware**: Dynamic selection based on RTT measurements

### 4. Utilities (`pkg/util`)
- **Histogram**: Binned metric collection
- **Matrix**: 2D histograms for multi-dimensional metrics
- **Moving Average**: Efficient bandwidth tracking
- **Network**: Ping testing, ZMQ integration, delay injection

## Performance Characteristics

- **QUIC**: Minimized connection establishment time
- **Goroutines**: Lightweight concurrency without thread overhead
- **Lock-Free**: Where possible, for reduced contention
- **Thread-Safe**: All monitoring utilities are goroutine-safe
- **Scalable**: Tested with thousands of concurrent connections

## Dependencies

- `quic-go`: QUIC protocol implementation
- `go-toml`: Configuration file parsing
- `go-ping`: Network latency measurement
- `zmq4`: ZeroMQ integration
- `gonum`: Statistics and distributions
- `influxdb`: Optional metrics export

See [go.mod](go.mod) for complete dependency list.

## Examples

Working examples are provided in the `examples/` directory:

```bash
# Start server
cd examples/server
go run main.go

# Start client (in another terminal)
cd examples/client
go run main.go -config config.toml
```

See [EXAMPLES.md](docs/EXAMPLES.md) for detailed examples including:
- Echo server/client
- Load balancing
- RTT measurement
- Network simulation

## Contributing

Contributions are welcome! Please ensure:
- Code follows Go conventions
- Tests pass: `go test ./...`
- Documentation is updated

## License

See LICENSE file for details.

## Support

For issues, questions, or suggestions:
1. Check the [documentation](docs/) for answers
2. Review [examples](examples/) for usage patterns
3. Open an issue on GitHub with reproduction steps
