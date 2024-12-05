# go_rope

`go_rope` is a Go package that provides functionalities for routing, client-server communication, and utility functions. It is designed to facilitate the development of network applications with support for the QUIC protocol, moving average calculations, and configuration management.

## Features

- **Client-Server Communication**: Implements client and server functionalities using the QUIC protocol.
- **Routing**: Provides routing logic with support for fixed and probabilistic routing decisions.
- **Configuration Management**: Loads and manages configuration using TOML files.
- **Utility Functions**: Includes utility functions for moving average calculations, logging, and more.

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
    "log"
    "time"

    "github.com/paolocastagno/go_rope/pkg/server"
    "github.com/quic-go/quic-go"
)

func main() {
    srv := &server.Server{
        IdDevice:   "Server",
        ListenAddr: "localhost:8080",
        AppCfg:     "../cfg/poa/server/app_reply.toml",
        QueueLen:   10,
        Workers:    5,
        Timeout:    30 * time.Second,
    }

    quicConf := &quic.Config{
        MaxIdleTimeout:     10 * time.Second,
        MaxIncomingStreams: 10000000,
        KeepAlivePeriod:    10 * time.Second,
    }

    if err := srv.InitServer(quicConf, server.InitReply); err != nil {
        log.Fatalf("Error running server: %v", err)
    }
}
```

#### Client

```go
package main

import (
    "log"
    "time"

    "github.com/paolocastagno/go_rope/pkg/client"
    "github.com/quic-go/quic-go"
)

func main() {
    cli := &client.Client{}

    configFile := "../cfg/poa/client/cfg.json"
    quicConf := &quic.Config{
        MaxIdleTimeout:     10 * time.Second,
        MaxIncomingStreams: 10000000,
        KeepAlivePeriod:    10 * time.Second,
    }

    if err := cli.InitClient(configFile, quicConf, client.InitFixed); err != nil {
        log.Fatalf("Error initializing client: %v", err)
    }
}
```

## Project Structure

- `pkg/client`: Contains client implementation.
- `pkg/server`: Contains server implementation.
- `pkg/routing`: Contains routing logic.
- `pkg/util`: Contains utility functions.
- `pkg/config`: Contains configuration management.

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.