# Routing Strategies

## Overview

The routing package provides multiple strategies for distributing requests across destinations. Each strategy can be customized for different use cases.

## Routing Strategies

### 1. Fixed Routing

Routes all requests to a single, predetermined destination.

**Use Cases:**
- Simple client-server communication
- Single destination scenarios
- Baseline performance testing

**Configuration:**
```toml
[routing]
strategy = "fixed"

[routing.variables]
dest = "server.example.com:4433"
```

**Implementation:**
```go
func FixedDecision(req *util.RoPEMessage, dest string) {
    req.Destination = dest
    // Update bandwidth metrics
}
```

**Metrics Collected:**
- Uplink bandwidth (requests sent)
- Downlink bandwidth (responses received)

---

### 2. Probability-Based Routing

Distributes requests across multiple destinations based on probability weights.

**Use Cases:**
- Load balancing across multiple servers
- A/B testing
- Multi-destination deployment

**Configuration:**
```toml
[routing]
strategy = "probability"

[routing.destinations]
server1 = "srv1.example.com:4433"
server2 = "srv2.example.com:4433"
server3 = "srv3.example.com:4433"

[routing.probabilities]
server1 = 0.5
server2 = 0.3
server3 = 0.2
```

**Implementation Details:**
- Cumulative probability selection
- Per-destination bandwidth tracking
- Weighted load distribution

---

### 3. Latency-Aware Routing

Makes routing decisions based on measured latency (RTT) to each destination.

**Use Cases:**
- Performance optimization
- Network condition adaptation
- Geographic load balancing
- Multi-cloud deployments

**Configuration:**
```toml
[routing]
strategy = "probabilityLatency"

[routing.destinations]
server1 = "srv1.example.com:4433"
server2 = "srv2.example.com:4433"

[routing.latency]
updateInterval = "10s"
weights = [1.0, 0.8]  # Relative weights for latency calculation
```

**Algorithm:**
1. Measure RTT to each destination
2. Calculate latency-based preference score
3. Select destination with best score
4. Periodically update latency measurements

**Metrics:**
- Per-destination latency averages
- Request distribution per destination
- Latency history for trending

---

## Custom Routing Implementation

To implement a custom routing strategy:

1. **Create routing logic function:**
```go
func CustomDecision(req *util.RoPEMessage, destinations []string, app *interface{}) string {
    // Your routing logic here
    return selectedDestination
}
```

2. **Register with client:**
```go
client.ForwardDecision = CustomDecision
```

3. **Handle response processing:**
```go
func CustomSetLastResponse(resp util.RoPEMessage, app *interface{}) {
    // Update metrics or state based on response
}

client.ForwardSetLastResponse = CustomSetLastResponse
```

## Routing Decision Factors

The framework supports routing decisions based on:

- **Static factors:** Fixed destination, probability weights
- **Dynamic factors:** Measured latency, bandwidth usage, queue depth
- **Application factors:** Custom logic from application layer
- **Network factors:** Availability, error rates

## Message Routing Fields

The `RoPEMessage` struct provides routing information:

```go
type RoPEMessage struct {
    Source      string  // Originating node
    Destination string  // Target destination
    Hop         string  // Current hop/relay
    ReqID       string  // Unique request identifier
    // ... other fields
}
```

## Performance Tips

1. **Fixed routing** is fastest - no decision overhead
2. **Probability routing** uses simple weight-based selection
3. **Latency-aware routing** includes measurement overhead but provides best performance
4. Cache routing decisions when possible
5. Use separate metrics channels per destination for non-blocking updates
