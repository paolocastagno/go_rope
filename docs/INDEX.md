# Documentation Index

Welcome to the go_rope documentation! This directory contains comprehensive guides for using and extending the go_rope framework.

## Quick Navigation

### 📚 Core Documentation

| Document | Best For | Time |
|----------|----------|------|
| **[README](../README.md)** | Feature overview and quick start | 5 min |
| **[ARCHITECTURE.md](ARCHITECTURE.md)** | Understanding system design | 10 min |
| **[CONFIGURATION.md](CONFIGURATION.md)** | Setting up config files | 15 min |
| **[API.md](API.md)** | API reference and function signatures | Reference |
| **[EXAMPLES.md](EXAMPLES.md)** | Working, runnable examples | 20 min |

### 🔍 Specialized Guides

| Document | Topic | Use Case |
|----------|-------|----------|
| **[ROUTING.md](ROUTING.md)** | Routing strategies | Implementing custom routing logic |
| **[UTILITIES.md](UTILITIES.md)** | Utility functions | Performance monitoring and metrics |

---

## Getting Started

### For First-Time Users

1. Read **[README](../README.md)** for the overview
2. Review **[ARCHITECTURE.md](ARCHITECTURE.md)** to understand the system
3. Try the **[EXAMPLES.md](EXAMPLES.md)** quick start
4. Reference **[CONFIGURATION.md](CONFIGURATION.md)** when setting up

### For Integration

1. Check **[ARCHITECTURE.md](ARCHITECTURE.md)** for design
2. Study **[API.md](API.md)** for available functions
3. Review **[EXAMPLES.md](EXAMPLES.md)** for patterns
4. Implement using **[CONFIGURATION.md](CONFIGURATION.md)**

### For Performance Tuning

1. Review **[UTILITIES.md](UTILITIES.md)** for monitoring
2. Check **[CONFIGURATION.md](CONFIGURATION.md)** parameters
3. Study **[EXAMPLES.md](EXAMPLES.md)** - Performance Testing section
4. Reference **[ROUTING.md](ROUTING.md)** for optimization

### For Custom Development

1. Understand **[ARCHITECTURE.md](ARCHITECTURE.md)** extension points
2. Study **[ROUTING.md](ROUTING.md)** for custom routing
3. Review **[EXAMPLES.md](EXAMPLES.md)** - Advanced Patterns
4. Reference **[API.md](API.md)** for function signatures

---

## Document Overview

### ARCHITECTURE.md
**Overview of system design and components.**

Contains:
- System architecture diagram
- Component descriptions (Client, Server, Routing, Utilities)
- Communication flow and sequence diagrams
- Concurrency model
- Extension points
- Performance considerations

**Read when:** You need to understand how components interact

---

### CONFIGURATION.md
**Complete configuration reference.**

Contains:
- Configuration sections (configuration, application, logger, routing)
- Parameter descriptions and types
- Complete example configurations
- Duration format reference
- Configuration best practices
- Environment variable overrides
- Validation rules

**Read when:** Setting up or troubleshooting configuration

---

### ROUTING.md
**Guide to routing strategies.**

Contains:
- Overview of routing system
- Fixed routing
- Probability-based routing
- Latency-aware routing
- Custom routing implementation
- Routing decision factors
- Performance tips

**Read when:** Implementing custom routing or choosing a strategy

---

### API.md
**Complete API reference.**

Contains:
- Client package API
- Server package API
- Utility package API
- Message types and constants
- Performance monitoring API
- Network utilities API
- Common patterns and examples

**Read when:** You need specific function signatures or parameters

---

### UTILITIES.md
**Guide to utility functions and tools.**

Contains:
- Histogram usage (1D and 2D)
- Moving average (bandwidth tracking)
- Ping testing
- Delay injection (distributions)
- ZMQ integration
- TLS configuration
- Message types
- Graceful shutdown
- Best practices

**Read when:** Using monitoring or network utilities

---

### EXAMPLES.md
**Runnable examples and tutorials.**

Contains:
- Quick start examples (echo, load balancing, RTT)
- Performance testing examples
- Network condition simulation
- Monitoring and analysis
- Advanced patterns
- Setup and execution instructions

**Read when:** Learning by example or need a starting template

---

## Common Tasks

### "I want to set up a basic client-server"
→ [README Quick Start](../README.md#quick-start) + [EXAMPLES.md Simple Echo](EXAMPLES.md#1-simple-echo-server-and-client)

### "I need to configure load balancing"
→ [ROUTING.md Probability](ROUTING.md#2-probability-based-routing) + [EXAMPLES.md Load Balancing](EXAMPLES.md#2-load-balancing-with-probability-routing)

### "I want to measure RTT/performance"
→ [UTILITIES.md Histogram](UTILITIES.md#histogram) + [EXAMPLES.md RTT Measurement](EXAMPLES.md#3-performance-testing-with-rtt-measurement)

### "I need to add custom routing logic"
→ [ROUTING.md Custom Implementation](ROUTING.md#custom-routing-implementation) + [EXAMPLES.md Advanced Patterns](EXAMPLES.md#advanced-patterns)

### "I want to simulate network conditions"
→ [UTILITIES.md Delay Injection](UTILITIES.md#delay-injection) + [EXAMPLES.md Network Simulation](EXAMPLES.md#4-network-condition-simulation)

### "I need to understand the architecture"
→ [ARCHITECTURE.md](ARCHITECTURE.md)

### "I'm looking for a specific function"
→ [API.md](API.md) (use browser search)

### "I want to monitor bandwidth"
→ [UTILITIES.md Moving Average](UTILITIES.md#moving-average-mavg) + [EXAMPLES.md Metrics Collection](EXAMPLES.md#monitoring-and-analysis)

---

## API Quick Reference

### Client
- `InitClient()` - Initialize client
- `NewReq()` - Send request
- `PrintParams()` - Print configuration

### Server
- `InitServer()` - Initialize server
- `Run()` - Start server

### Utilities
- `NewHistogram()` - Binned metrics
- `NewMatrix()` - 2D metrics
- `NewMavg()` - Bandwidth tracking
- `Delay()` - Network simulation

### Configuration
- TOML structure with [configuration], [application], [logger], [routing] sections

---

## Performance Benchmarks

For performance tuning reference:

| Operation | Typical Time | Notes |
|-----------|------------|-------|
| QUIC connection | 5-10ms | Measured RTT |
| Histogram Add | < 1µs | Lock-free in practice |
| Moving average update | < 1µs | O(1) operation |
| Request processing | Configurable | `ProcessingTime` in config |

---

## Troubleshooting

### "Configuration file not found"
→ Check [CONFIGURATION.md](CONFIGURATION.md) file format and location

### "Queue full messages"
→ Increase `BufSize` in [CONFIGURATION.md](CONFIGURATION.md#server-configuration)

### "Connection timeouts"
→ Review timeout settings in [CONFIGURATION.md](CONFIGURATION.md#client-configuration)

### "Low throughput"
→ Check [CONFIGURATION.md](CONFIGURATION.md) worker count and buffer size

### "High latency"
→ Review [ROUTING.md](ROUTING.md) routing strategy or [UTILITIES.md](UTILITIES.md#delay-injection) for simulation

---

## Key Concepts

### RoPEMessage
The core message type for all communication. See [API.md](API.md#ropemessage) for structure.

### Routing Decision
Function that determines destination for each request. See [ROUTING.md](ROUTING.md) for strategies.

### Metrics Collection
Use Histogram, Matrix, or Mavg for performance monitoring. See [UTILITIES.md](UTILITIES.md).

### Configuration
TOML-based configuration with sections for different aspects. See [CONFIGURATION.md](CONFIGURATION.md).

---

## Additional Resources

- **Examples Directory**: `examples/` - Working implementations
- **Source Code**: `pkg/` - Implementation details
- **Go Module**: `go.mod` - Dependencies

---

## Contributing to Documentation

When updating documentation:
1. Keep examples runnable and tested
2. Update related documents
3. Maintain consistent formatting
4. Review API changes against code
5. Test configuration examples

---

**Last Updated**: 2026-03-20

For the latest updates and issues, visit the [GitHub repository](https://github.com/paolocastagno/go_rope).
