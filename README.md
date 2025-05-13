# go_rope

`go_rope` is a Go package that provides functionalities for routing, client-server communication, and utility functions. It is designed to facilitate the development of network applications with support for the QUIC protocol, moving average calculations, histograms, and configuration management.

## Features

- **Client-Server Communication**: Implements client and server functionalities using the QUIC protocol.
- **Routing**: Provides routing logic with support for fixed and probabilistic routing decisions.
- **Configuration Management**: Loads and manages configuration using TOML files.
- **Utility Functions**: Includes utility functions for moving average calculations, histograms, logging, and more.
- **Test Duration**: Supports automatic stopping of the client after a configurable test duration.

## Installation

To install the package, run:

```sh
go get github.com/paolocastagno/go_rope
```

## Usage

### Client-Server Example

#### Server

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

func InitReply(conf *toml.Tree) {
    fmt.Println("Initializing server logic...")
    // Server-specific initialization logic
}

func main() {
    srv := server.NewServer(
        "Server",
        "localhost:8080",
        "app_server.toml",
        10,
        5,
        30*time.Second,
    )

    quicConf := &quic.Config{
        MaxIdleTimeout:     10 * time.Second,
        MaxIncomingStreams: 10000000,
        KeepAlivePeriod:    10 * time.Second,
    }

    if err := srv.InitServer(quicConf, InitReply); err != nil {
        log.Fatalf("Error running server: %v", err)
    }
}
```

#### Client

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

func InitFixed(conf *toml.Tree) *interface{} {
    fmt.Println("Initializing client logic...")
    // Client-specific initialization logic
    return nil
}

func main() {
    configFile := flag.String("config", "cfg.toml", "Path to the configuration file")
    flag.Parse()

    cli := &client.Client{}

    quicConf := &quic.Config{
        MaxIdleTimeout:     10 * time.Second,
        MaxIncomingStreams: 10000000,
        KeepAlivePeriod:    10 * time.Second,
    }

    if err := cli.InitClient(quicConf, *configFile, InitFixed); err != nil {
        log.Fatalf("Error initializing client: %v", err)
    }

    ticker := time.NewTicker(1 * time.Second)
    defer ticker.Stop()

    var wg sync.WaitGroup
    for {
        select {
        case <-cli.Done:
            fmt.Println("Test duration elapsed. Stopping client...")
            wg.Wait()
            return
        case <-ticker.C:
            wg.Add(1)
            go func() {
                defer wg.Done()
                fmt.Println("Sending request...")
                // Add request logic here
            }()
        }
    }
}
```

### Configuration Example (`cfg.toml`)

```toml
[config]
IdDevice = "1"
RequestsPerSec = 1
MaxConcurrentConnections = 20000
TestDuration = "30s"
Timeout = "40s"
destinations = ["localhost:8080"]

[application]
name = "fixed"
init = "InitFixed"
destinations = ["localhost:8080"]
requestSize = 100
responseSize = 100
```

## Project Structure

- `pkg/client`: Contains client implementation, including test duration handling and QUIC-based communication.
- `pkg/server`: Contains server implementation with routing and response logic.
- `pkg/util`: Contains utility functions for moving averages, histograms, logging, and graceful shutdown.
- `pkg/config`: Contains configuration management using TOML files.
