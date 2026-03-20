# go_rope Architecture

## Overview

`go_rope` is a high-performance, QUIC-based network communication framework designed for distributed systems and network testing. It provides a flexible architecture for client-server communication with support for multiple routing strategies, performance monitoring, and metrics collection.

## System Components

### 1. **Client Package** (`pkg/client`)

The client component handles outbound QUIC connections and request management.

**Key Features:**
- Establishes connections to multiple destinations
- Sends requests with configurable payloads
- Receives and processes responses
- Supports test duration limits
- Concurrent request management
- Request/response payloads can include timestamps for RTT measurement

**Main Functions:**
- `InitClient()`: Initialize client with configuration
- `NewReq()`: Create and send a new request
- `setupTestDuration()`: Configure automatic test termination

### 2. **Server Package** (`pkg/server`)

The server component accepts QUIC connections and processes incoming requests.

**Key Features:**
- Multi-worker request processing
- Job queue with configurable buffer size
- Graceful connection handling
- Application-specific logic hooks
- Response forwarding with decision logic

**Main Functions:**
- `InitServer()`: Initialize server with configuration
- `Run()`: Start the server listener
- Worker goroutines process requests from a queue

### 3. **Routing Package** (`pkg/routing`)

Provides multiple routing strategies for request forwarding.

**Supported Strategies:**
- **Fixed**: Routes all requests to a fixed destination
- **Probability**: Routes based on probabilistic rules
- **Probability with Latency**: Routes based on latency measurements

**Components:**
- `proxy.go`: Proxy abstraction
- `fixed.go`: Fixed routing implementation
- `probability.go`: Probability-based routing
- `probabilityLatency.go`: Latency-aware routing

### 4. **Utility Package** (`pkg/util`)

Provides essential utilities for performance monitoring and network operations.

**Key Components:**

#### Message Format (RoPEMessage)
```go
type RoPEMessage struct {
    ReqID       string          // Request identifier
    Type        RoPEMsgType     // Request/Response/Error type
    Log         bool            // Enable logging
    ResSize     int32           // Response size
    Body        []byte          // Message payload
    Source      string          // Source node ID
    Hop         string          // Current hop
    Destination string          // Target destination
    Timestamp   time.Time       // Message timestamp
}
```

#### Performance Monitoring
- **Moving Average (Mavg)**: Sliding window average for bandwidth calculations
- **Histogram**: Binned data collection for RTT/latency analysis
- **Matrix**: 2D histogram for multi-dimensional metrics

#### Network Utilities
- **Ping**: Network reachability and latency testing
- **ZMQ**: Integration with ZeroMQ for modem information
- **Delay**: Support for delay injection (uniform, exponential, constant)

#### TLS Configuration
- Auto-generates self-signed certificates for QUIC
- Used for both client and server

## Communication Flow

```
┌─────────────┐                                    ┌─────────────┐
│   Client    │  ────── QUIC Stream ─────────>    │   Server    │
│             │  <──── Response Stream ───────    │             │
└─────────────┘                                    └─────────────┘
       │                                                  │
       │ (1) InitClient()                                │
       │     - Load config                               │
       │     - Create QUIC connections                   │
       │     - Setup test duration                       │
       │                                                  │
       │ (2) NewReq()                                    │
       │     - Create RoPEMessage                        │
       │     - Apply routing decision                    │ (3) Accept connection
       │     - Send via stream                           │
       │     - Encode data                               │ (4) newRequest()
       │     - Receive response                          │     - Accept stream
       │     └─────────────────────────────────────>    │     - Decode message
       │                                                 │
       │                                                 │ (5) worker()
       │                                                 │     - Process job
       │                                                 │     - Apply decision logic
       │     <──────────────────────────────────────    │     - Encode response
       │                                                 │     - Send response
       │ (6) forwardResponse()
       │     - Log response
       │     - Update metrics
       └──────────────────────────────────────────────────┘
```

## Configuration Architecture

Configuration uses TOML format with sections for:
- **configuration**: Client/server parameters
- **application**: Application-specific settings
- **logger**: Logging configuration
- **routing**: Routing strategy parameters

## Concurrency Model

### Client Side
- Main event loop with ticker-based request generation
- Worker goroutines for each request
- Sync primitives to limit concurrent connections

### Server Side
- Single listener goroutine
- Per-connection goroutines handle stream acceptance
- Worker pool processes jobs from a queue
- Queue depth indicates system load

## Extension Points

The framework is designed for extensibility:

1. **Application Logic**: Custom initialization and request/response handling
2. **Routing Strategies**: Implement custom `ForwardDecision` functions
3. **Response Handling**: Custom `ForwardSetLastResponse` logic
4. **Message Types**: Extensible message type system
5. **Monitoring**: Add custom metrics collection

## Performance Considerations

- **QUIC Protocol**: Low-latency, multiplexed connections
- **Goroutines**: Efficient concurrency without threads
- **Moving Averages**: Efficient bandwidth tracking
- **Histograms**: O(1) insertion, sorted output
- **Graceful Shutdown**: Clean connection termination
