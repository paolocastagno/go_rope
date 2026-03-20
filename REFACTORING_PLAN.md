# Refactoring Plan: go_rope Bug Fixes

**Status**: In Planning
**Created**: 2026-03-20
**Target Completion**: Phase 3 completion

---

## Overview

This plan addresses all 31 identified bugs through **4 phased sprints**, each with clear deliverables, testing strategy, and success criteria.

**Key Principles**:
- ✅ Fix critical bugs first (crashes)
- ✅ Add synchronization before logic fixes
- ✅ Comprehensive testing at each phase
- ✅ Maintain backward compatibility where possible
- ✅ Document changes thoroughly

---

## Dependency Graph

```
Phase 1: Foundation (Synchronization)
    ↓ (unblocks)
Phase 2: Stability (Error Handling)
    ↓ (unblocks)
Phase 3: Quality (Logic & Resources)
    ↓ (unblocks)
Phase 4: Polish (Performance & Edge Cases)
```

---

# PHASE 1: FOUNDATION - SYNCHRONIZATION & CRASHES

**Duration**: 3-4 days
**Risk**: Medium
**Impact**: High (fixes most critical bugs)
**Blockers**: None

## Goal
Fix all race conditions and undefined variable crashes that prevent the application from running.

### 1.1 - Fix Undefined Variables (HIGH URGENCY)

**Files Affected**:
- `pkg/routing/fixed.go` (2 critical bugs)
- `pkg/routing/probabilityLatency.go` (1 critical bug)

**Changes Required**:

#### a) Fix fixed.go:59-60

**Current Code** (BROKEN):
```go
for i, s := range d {  // d undefined!
    fmt.Printf("\tUplink %s:  %f bytes/s\n", s, util.Mavg_eval(b_up_i[i], ...))
}
```

**Root Cause**: Code copies from probability.go but doesn't have `d` or `b_up_i`

**Fix Option 1** (Recommended - Remove dead code):
```go
// DELETE lines 59-61 entirely
// This code appears to be copy-paste that was never finished
// Fixed routing doesn't track per-destination metrics
```

**Fix Option 2** (If used elsewhere - Extract properly):
```go
// Implement proper bandwidth tracking
// This requires architectural change - defer to Phase 3
```

**Recommendation**: **DELETE** - It's incomplete/dead code

**Test**:
```bash
go build ./pkg/routing  # Should compile without errors
```

#### b) Fix probabilityLatency.go:126

**Current Code** (BROKEN):
```go
for i < len(d) && d[i] != lastResp.Destination {  // d undefined!
    i++
}
```

**Root Cause**: Variable named wrong - should be `sinks`

**Fix**:
```go
for i < len(sinks) && sinks[i] != lastResp.Destination {
    i++
}
```

**Verification**:
- Check variable `sinks` is available in scope
- Verify logic: finding index of destination in sinks array
- Confirm it's used correctly at line 136: `cd += int64(len(lastResp.Body))`

**Implementation**:
```go
// Before (line 118-136):
func PLSetLastResponse(lastResp *util.RoPEMessage) {
    if lastResp.Type == util.Response {
        i := 0
        for i < len(d) && d[i] != lastResp.Destination {  // WRONG
            i++
        }
        // ...
    }
}

// After:
func PLSetLastResponse(lastResp *util.RoPEMessage) {
    if lastResp.Type == util.Response {
        i := 0
        for i < len(sinks) && sinks[i] != lastResp.Destination {  // CORRECT
            i++
        }
        // ...
    }
}
```

**Test**:
```go
// Verify sinks array is initialized
// Verify latency routing compiles and doesn't panic
go test ./pkg/routing -v
```

---

### 1.2 - Add Mutex Synchronization (CRITICAL)

**Files Affected**:
- `pkg/server/server.go`
- `pkg/routing/probability.go`
- `pkg/routing/probabilityLatency.go`
- `pkg/routing/fixed.go`
- `pkg/util/util.go` (logger)
- `pkg/routing/proxy.go`

**Strategy**: Create wrapper types with embedded mutexes

#### a) Synchronize server.go global state

**Current** (UNSAFE):
```go
var sessions map[string]quic.EarlyConnection
var ForwardBlock func(...)
var ForwardDecision func(...)
var ForwardSetLastResponse func(...)
```

**Solution - Create struct with mutex**:

```go
type ServerState struct {
    mu                    sync.RWMutex
    sessions              map[string]quic.EarlyConnection
    forwardBlock          func(*util.RoPEMessage, quic.Stream)
    forwardDecision       func(*util.RoPEMessage, *map[string]quic.EarlyConnection, int64) bool
    forwardSetLastResponse func(*util.RoPEMessage)
}

var serverState = &ServerState{
    sessions: make(map[string]quic.EarlyConnection),
}

// Add getter/setter methods:
func (s *ServerState) GetSessions() map[string]quic.EarlyConnection {
    s.mu.RLock()
    defer s.mu.RUnlock()
    return s.sessions
}

func (s *ServerState) SetForwardDecision(f func(...) bool) {
    s.mu.Lock()
    defer s.mu.Unlock()
    s.forwardDecision = f
}

func (s *ServerState) CallForwardDecision(...) bool {
    s.mu.RLock()
    defer s.mu.RUnlock()
    if s.forwardDecision == nil {
        return false
    }
    return s.forwardDecision(...)
}
```

**Changes in worker() function** (line 232):
```go
// Before:
send := ForwardDecision(&packet, &sessions, attempt)

// After:
send := serverState.CallForwardDecision(&packet, serverState.GetSessions(), attempt)
```

**Test**:
```go
// Verify no race conditions with -race flag
go test -race ./pkg/server -v
```

#### b) Synchronize probability.go globals

**Current** (UNSAFE):
```go
var p, d []string
var cu, cd int64
var cup, cdw []int64
var stime = time.Time{}
var b_up, b_down util.Mavg
var b_up_i, b_down_i []util.Mavg
```

**Solution - Create wrapper**:

```go
type ProbabilityRouter struct {
    mu         sync.RWMutex

    // Configuration
    destinations []string
    weights      []float64

    // Metrics
    cu, cd       int64              // cumulative up/down
    cup, cdw     []int64            // per-destination
    stime        time.Time          // observation window start
    b_up, b_down util.Mavg          // overall bandwidth
    b_up_i, b_down_i []util.Mavg   // per-destination bandwidth
}

var probRouter *ProbabilityRouter

func (pr *ProbabilityRouter) Decision(req *util.RoPEMessage) {
    pr.mu.Lock()
    defer pr.mu.Unlock()

    // Original logic from WeightedRandomDecision()
    cu += int64(len(req.Body))

    if pr.stime == (time.Time{}) {
        pr.stime = time.Now()
    } else if time.Now().After(pr.stime.Add(10*time.Second)) {
        pr.stime = time.Now()
        util.Mavg_push(&pr.b_up, pr.cu)
        util.Mavg_push(&pr.b_down, pr.cd)
        // ... rest of logic
    }
}

func (pr *ProbabilityRouter) SetLastResponse(lastResp *util.RoPEMessage) {
    pr.mu.Lock()
    defer pr.mu.Unlock()

    // Original logic from WeightedRandomSetLastResponse()
    if lastResp.Type == util.Response {
        pr.cd += int64(len(lastResp.Body))
        // ... find destination index and update metrics
    }
}
```

**Update InitProbability**:
```go
func InitProbability(conf *toml.Tree, proxy *Proxy) {
    probRouter = &ProbabilityRouter{
        destinations: [...],
        weights: [...],
        b_up: util.NewMavg(100),
        b_down: util.NewMavg(100),
    }

    proxy.ForwardDecision = func(msg *util.RoPEMessage) {
        probRouter.Decision(msg)
        // set destination...
    }
}
```

**Test**:
```bash
go test -race ./pkg/routing -v -run Probability
```

#### c) Synchronize probabilityLatency.go globals

**Same approach as probability.go**:

```go
type LatencyAwareRouter struct {
    mu sync.RWMutex

    routingProb []string
    sinks       []string
    countup     []int64
    countdown   []int64

    distribution_up   interface{}
    distribution_down interface{}

    st        time.Time
    byte_up   int64
    byte_down int64
    // ... more fields
}

var latencyRouter *LatencyAwareRouter

func (lr *LatencyAwareRouter) Decision(req *util.RoPEMessage) {
    lr.mu.Lock()
    defer lr.mu.Unlock()
    // ... implementation
}

func (lr *LatencyAwareRouter) SetLastResponse(lastResp *util.RoPEMessage) {
    lr.mu.Lock()
    defer lr.mu.Unlock()
    // ... implementation
}
```

**Test**:
```bash
go test -race ./pkg/routing -v -run Latency
```

#### d) Synchronize fixed.go globals

**Current** (UNSAFE):
```go
var cu, cd int64
var stim = time.Time{}
var b_u, b_d = util.NewMavg(twind), util.NewMavg(twind)
```

**Solution**:

```go
type FixedRouter struct {
    mu   sync.RWMutex
    dest string

    cu, cd int64
    stim   time.Time
    b_u, b_d util.Mavg
}

var fixedRouter *FixedRouter

func (fr *FixedRouter) Decision(req *util.RoPEMessage) {
    fr.mu.Lock()
    defer fr.mu.Unlock()

    cu += int64(len(req.Body))
    if (fr.stim == time.Time{}) {
        fr.stim = time.Now()
    } else if time.Now().After(fr.stim.Add(10*time.Second)) {
        fr.stim = time.Now()
        util.Mavg_push(&fr.b_u, fr.cu)
        util.Mavg_push(&fr.b_d, fr.cd)
        fmt.Printf("\nUplink: %f\n", util.Mavg_eval(fr.b_u, 10))
        fmt.Printf("Downlink: %f\n", util.Mavg_eval(fr.b_d, 10))
    }
}
```

**Remove dead code**:
```go
// DELETE lines 59-61 (the broken loop)
```

**Test**:
```bash
go build ./pkg/routing/fixed.go  # Must compile
go test -race ./pkg/routing -v -run Fixed
```

#### e) Synchronize util.go logger

**Current** (UNSAFE):
```go
var client influxdb2.Client
var writeAPI api.WriteAPIBlocking
```

**Solution - Use sync.Once**:

```go
var (
    loggerOnce sync.Once
    logClient  influxdb2.Client
    logAPI     api.WriteAPIBlocking
    logErr     error
)

func initLogger() error {
    var err error
    loggerOnce.Do(func() {
        logClient = influxdb2.NewClient(...)
        logAPI = logClient.WriteAPIBlocking(...)
    })
    return err
}

func IsLoggerEnabled() bool {
    // Already thread-safe due to sync.Once
    return logClient != nil
}

func CloseLogger() {
    // Use RWMutex here
    loggerMu.Lock()
    defer loggerMu.Unlock()
    if logClient != nil {
        logClient.Close()
        logClient = nil
    }
}
```

**Test**:
```bash
go test -race ./pkg/util -v -run Logger
```

---

### 1.3 - Fix Nil Channel Panics (proxy.go)

**Current** (line 20-21, 132, 240, etc):
```go
var tokensUp, tokensDn chan struct{}  // Can be nil

// Later:
tokensUp <- struct{}{}  // PANIC if nil
```

**Solution**:

```go
type Proxy struct {
    mu               sync.RWMutex
    tokensUp, tokensDn chan struct{}  // Nil-safe by using methods
    parallelConn     int
}

func (p *Proxy) SendUpToken() bool {
    p.mu.RLock()
    ch := p.tokensUp
    p.mu.RUnlock()

    if ch == nil {
        return true  // No rate limiting
    }

    select {
    case ch <- struct{}{}:
        return true
    case <-time.After(5 * time.Second):
        return false
    }
}

func (p *Proxy) ReceiveUpToken() {
    p.mu.RLock()
    ch := p.tokensUp
    p.mu.RUnlock()

    if ch != nil {
        <-ch
    }
}
```

**Or simpler - Initialize properly**:

```go
func (p *Proxy) Init() {
    if p.parallelConn > 0 {
        p.tokensUp = make(chan struct{}, p.parallelConn)
        p.tokensDn = make(chan struct{}, p.parallelConn)
    } else {
        // Create dummy channels that never block
        p.tokensUp = make(chan struct{})
        p.tokensDn = make(chan struct{})
    }
}
```

**Test**:
```bash
# Test with ParallelConn = 0
go test -race ./pkg/routing -v -run Proxy
```

---

### 1.4 - Phase 1 Testing

**Test suite to run**:

```bash
# Run all sync tests
go test -race ./pkg/server ./pkg/routing ./pkg/util -v

# Stress test with concurrency
go test -race ./pkg/... -v -run TestConcurrent

# Check for goroutine leaks
go test ./pkg/... -v -count=10  # Run multiple times

# Compile check
go build ./...
```

**Success Criteria**:
- ✅ All packages compile without errors
- ✅ No race condition warnings: `go test -race ./...`
- ✅ Undefined variable compile errors fixed
- ✅ No nil channel panics in tests
- ✅ All changes backward compatible in API

---

**Estimated Effort**: 8-12 hours
**Complexity**: Medium
**Risk**: Medium (refactoring critical paths)

---

# PHASE 2: STABILITY - ERROR HANDLING & CRASHES

**Duration**: 2-3 days
**Risk**: Medium
**Impact**: High (prevents crash scenarios)
**Blockers**: Phase 1

## Goal
Fix type assertion panics and replace panics with proper error handling.

### 2.1 - Fix GenerateTLSConfig Panics

**File**: `pkg/util/util.go:52-73`

**Current** (BAD):
```go
func GenerateTLSConfig() *tls.Config {
    key, err := rsa.GenerateKey(rand.Reader, 1024)
    if err != nil {
        panic(err)  // LINE 55
    }
    // ... more panics
}
```

**Solution - Return error**:

```go
func GenerateTLSConfig() (*tls.Config, error) {
    key, err := rsa.GenerateKey(rand.Reader, 1024)
    if err != nil {
        return nil, fmt.Errorf("failed to generate RSA key: %w", err)
    }

    template := x509.Certificate{SerialNumber: big.NewInt(1)}
    certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
    if err != nil {
        return nil, fmt.Errorf("failed to create certificate: %w", err)
    }

    keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})
    certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})

    tlsCert, err := tls.X509KeyPair(certPEM, keyPEM)
    if err != nil {
        return nil, fmt.Errorf("failed to parse TLS certificate: %w", err)
    }

    return &tls.Config{
        Certificates: []tls.Certificate{tlsCert},
        NextProtos:   []string{"RoPEProtocol"},
    }, nil
}
```

**Update call sites**:

In `pkg/server/server.go:157`:
```go
// Before:
listener, err := quic.ListenAddrEarly(server.ListenAddr, util.GenerateTLSConfig(), quicConf)

// After:
tlsConfig, err := util.GenerateTLSConfig()
if err != nil {
    return fmt.Errorf("failed to generate TLS config: %w", err)
}
listener, err := quic.ListenAddrEarly(server.ListenAddr, tlsConfig, quicConf)
```

**Test**:
```bash
go test ./pkg/util -v -run TLS
# Verify error handling works
```

---

### 2.2 - Fix Type Assertion Panics in Delay Functions

**File**: `pkg/util/util.go:193-250`

**Current** (UNSAFE):
```go
func InitDelay(dist string, dist_params interface{}) interface{} {
    if dist_params == nil {
        panic("No delay or distribution specified")  // LINE 197
    }

    switch dist {
    case "uniform":
        params := dist_params.([]interface{})  // Unchecked assertion!
        // ...
    case "exponential":
        avg, _ := time.ParseDuration(dist_params.(string))  // Unchecked!
        // ...
    case "constant":
        distribution = dist_params  // No type check
        // ...
    }
}

func Delay(distribution interface{}, distType string) {
    switch distType {
    case "uniform":
        d := (distribution.(distuv.Uniform)).Rand()  // PANIC if wrong type
    case "exponential":
        d := (distribution.(distuv.Exponential)).Rand()  // PANIC if wrong type
    case "constant":
        delay, _ = time.ParseDuration(distribution.(string))  // PANIC if wrong type
    }
}
```

**Solution - Add validation**:

```go
func InitDelay(dist string, dist_params interface{}) (interface{}, error) {
    if dist_params == nil && dist != "no delay" {
        return nil, fmt.Errorf("distribution '%s' requires parameters", dist)
    }

    var distribution interface{}

    switch dist {
    case "uniform":
        params, ok := dist_params.([]interface{})
        if !ok || len(params) < 2 {
            return nil, fmt.Errorf("uniform distribution requires 2 parameters [min, max]")
        }

        minStr, ok := params[0].(string)
        if !ok {
            return nil, fmt.Errorf("uniform min parameter must be a duration string")
        }
        min, err := time.ParseDuration(minStr)
        if err != nil {
            return nil, fmt.Errorf("invalid uniform min duration: %w", err)
        }

        maxStr, ok := params[1].(string)
        if !ok {
            return nil, fmt.Errorf("uniform max parameter must be a duration string")
        }
        max, err := time.ParseDuration(maxStr)
        if err != nil {
            return nil, fmt.Errorf("invalid uniform max duration: %w", err)
        }

        if min >= max {
            return nil, fmt.Errorf("uniform min must be less than max")
        }

        fmt.Printf("Distribution (uniform)\n\t - interval (%d - %d)\n", min, max)
        distribution = &distuv.Uniform{
            Min: float64(int64(min) / int64(time.Second)),
            Max: float64(int64(max) / int64(time.Second)),
        }

    case "exponential":
        avgStr, ok := dist_params.(string)
        if !ok {
            return nil, fmt.Errorf("exponential distribution requires duration string parameter")
        }

        avg, err := time.ParseDuration(avgStr)
        if err != nil {
            return nil, fmt.Errorf("invalid exponential duration: %w", err)
        }

        rate := float64(time.Second) / float64(int64(avg))
        fmt.Printf("Distribution (exponential)\n\t - rate %f\n", rate)
        distribution = distuv.Exponential{Rate: rate}

    case "constant":
        constStr, ok := dist_params.(string)
        if !ok {
            return nil, fmt.Errorf("constant distribution requires duration string parameter")
        }

        _, err := time.ParseDuration(constStr)
        if err != nil {
            return nil, fmt.Errorf("invalid constant duration: %w", err)
        }

        fmt.Printf("Distribution (constant)\n\t - value %s\n", constStr)
        distribution = constStr

    case "no delay":
        return nil, nil

    default:
        return nil, fmt.Errorf("unsupported distribution: %s", dist)
    }

    return distribution, nil
}

func Delay(distribution interface{}, distType string) error {
    if distribution == nil {
        return nil  // No delay
    }

    var delay time.Duration

    switch distType {
    case "uniform":
        u, ok := distribution.(distuv.Uniform)
        if !ok {
            return fmt.Errorf("expected distuv.Uniform, got %T", distribution)
        }
        d := u.Rand()
        delay = time.Duration(d * float64(time.Second))

    case "exponential":
        e, ok := distribution.(distuv.Exponential)
        if !ok {
            return fmt.Errorf("expected distuv.Exponential, got %T", distribution)
        }
        d := e.Rand()
        delay = time.Duration(d * float64(time.Second))

    case "constant":
        s, ok := distribution.(string)
        if !ok {
            return fmt.Errorf("expected string, got %T", distribution)
        }
        var err error
        delay, err = time.ParseDuration(s)
        if err != nil {
            return fmt.Errorf("invalid constant delay: %w", err)
        }

    case "no delay":
        return nil

    default:
        return fmt.Errorf("unsupported delay type: %s", distType)
    }

    if delay > 0 {
        time.Sleep(delay)
    }
    return nil
}
```

**Update call sites**:

Anywhere `InitDelay` is called:
```go
// Before:
dist := util.InitDelay("uniform", []interface{}{"10ms", "100ms"})

// After:
dist, err := util.InitDelay("uniform", []interface{}{"10ms", "100ms"})
if err != nil {
    return fmt.Errorf("failed to initialize delay: %w", err)
}
```

Anywhere `Delay` is called:
```go
// Before:
util.Delay(dist, "uniform")

// After:
if err := util.Delay(dist, "uniform"); err != nil {
    return err
}
```

**Test**:
```bash
go test ./pkg/util -v -run Delay
# Test invalid inputs don't panic
```

---

### 2.3 - Fix Configuration Type Assertions

**File**: `pkg/client/client.go:84, 102, 104`

**Current** (UNSAFE):
```go
client.IdDevice = GetString(cfgMap["configuration"].(map[string]interface{}), ...)  // LINE 84
// cfgMap["configuration"] could not exist or not be a map

client.MaxConcurrentConnections = uint(config.GetDefault("configuration.MaxConcurrentConnections", int64(1)).(int64))  // LINE 102
// Type assertion without check

client.TestDuration = GetDuration(cfgMap["configuration"].(map[string]interface{}), ...)  // LINE 104
// Repeated assertion problem
```

**Solution**:

```go
func loadConfig(client *Client, initLogic func(*toml.Tree) *interface{}) error {
    if client.CfgFile == "" {
        return errors.New("no configuration file specified")
    }

    config, err := toml.LoadFile(client.CfgFile)
    if err != nil {
        return fmt.Errorf("error loading configuration: %v", err)
    }

    cfgMap := config.ToMap()
    fmt.Printf("Config contents: %v\n", cfgMap)

    // Safely retrieve configuration section
    configSection, ok := cfgMap["configuration"]
    if !ok {
        return errors.New("missing 'configuration' section in config")
    }

    configMap, ok := configSection.(map[string]interface{})
    if !ok {
        return fmt.Errorf("'configuration' section should be a map, got %T", configSection)
    }

    // Get IdDevice
    client.IdDevice = GetString(configMap, "id_device", "default_id")
    if client.IdDevice == "" {
        return errors.New("'id_device' cannot be empty")
    }

    // Get Destinations
    configDestinations, ok := config.Get("configuration.destinations").([]interface{})
    if !ok || len(configDestinations) == 0 {
        return errors.New("missing or invalid 'destinations' in 'configuration' section")
    }

    client.Destinations = make([]string, len(configDestinations))
    for i, d := range configDestinations {
        str, ok := d.(string)
        if !ok {
            return fmt.Errorf("invalid destination at index %d: expected string, got %T", i, d)
        }
        client.Destinations[i] = str
    }

    // Get MaxConcurrentConnections
    maxConnValue := config.GetDefault("configuration.MaxConcurrentConnections", int64(1))
    maxConnInt, ok := maxConnValue.(int64)
    if !ok {
        return fmt.Errorf("MaxConcurrentConnections must be integer, got %T", maxConnValue)
    }
    if maxConnInt <= 0 {
        return fmt.Errorf("MaxConcurrentConnections must be positive")
    }
    client.MaxConcurrentConnections = uint(maxConnInt)

    // Get TestDuration
    client.TestDuration = GetDuration(configMap, "TestDuration", "0s")

    // Get Timeout
    client.Timeout = GetDuration(configMap, "timeout", "30s")

    client.Counter = make(chan int64, client.MaxConcurrentConnections)
    client.LoggerEnabled = config.GetDefault("logger_enabled", false).(bool)
    client.Connections = make([]quic.EarlyConnection, 0, client.MaxConcurrentConnections)

    // Retrieve the application section
    applicationConfig, ok := config.Get("application").(*toml.Tree)
    if !ok {
        return errors.New("missing or invalid 'application' section in configuration")
    }

    client.Application = initLogic(applicationConfig)

    if client.LoggerEnabled {
        loggercfg, ok := config.Get("logger").(*toml.Tree)
        if !ok {
            return errors.New("no logger configuration specified")
        }
        util.SetLoggerParamFromConf(loggercfg)
    }

    return nil
}
```

**Test**:
```bash
go test ./pkg/client -v
# Test with malformed config files
```

---

### 2.4 - Fix server.go Config Error Message

**File**: `pkg/server/server.go:111-114`

**Current** (BUG):
```go
if bufSize, ok := cfg.Get("configuration.BufSize").(int64); ok {
    server.BufSize = bufSize
} else {
    return errors.New("missing or invalid 'BufSize' in configuration[" +
        fmt.Sprintf("%d", bufSize) + "]")  // bufSize undefined!
}
```

**Fix**:
```go
if bufSize, ok := cfg.Get("configuration.BufSize").(int64); ok {
    if bufSize <= 0 {
        return fmt.Errorf("BufSize must be positive, got %d", bufSize)
    }
    server.BufSize = bufSize
} else {
    return errors.New("missing or invalid 'BufSize' in configuration; expected positive integer")
}
```

**Test**:
```bash
# Test with invalid config
go build ./pkg/server
```

---

### 2.5 - Phase 2 Testing

```bash
# Run all error handling tests
go test ./pkg/util ./pkg/client ./pkg/server -v

# Test panic recovery
go test -v ./pkg/... -run TestErrorHandling

# Verify no panics with bad input
go test ./pkg/... -v -count=5
```

**Success Criteria**:
- ✅ No panics on invalid input
- ✅ All errors returned with context
- ✅ Type assertions have proper validation
- ✅ Configuration errors are clear
- ✅ All error paths tested

---

**Estimated Effort**: 6-8 hours
**Complexity**: Medium
**Risk**: Medium (API changes for error returns)

---

# PHASE 3: QUALITY - LOGIC & RESOURCES

**Duration**: 3-5 days
**Risk**: Medium
**Impact**: Medium (prevents data corruption, improves reliability)
**Blockers**: Phase 1 & 2

## Goal
Fix resource leaks, bounds checking, and logic errors.

### 3.1 - Fix Index Bounds Checking

**Files**:
- `pkg/routing/probability.go:56`
- `pkg/routing/probabilityLatency.go:79`
- `pkg/client/client.go:223-228`

#### a) Probability routing - Fix empty array access

**Current** (UNSAFE):
```go
var pdest float64 = p[0]  // PANIC if p is empty
```

**Fix**:
```go
if len(p) == 0 {
    return fmt.Errorf("no destinations available for probability routing")
}
var pdest float64 = p[0]
```

#### b) Latency routing - Fix empty array access

**Current** (UNSAFE):
```go
var pdest float64 = routingProb[0]  // PANIC if empty
```

**Fix**:
```go
if len(routingProb) == 0 {
    return fmt.Errorf("no destinations available for latency routing")
}
var pdest float64 = routingProb[0]
```

#### c) Client - Fix destination lookup

**Current** (SILENT FAIL):
```go
idxdest := 0  // Default value
for i, d := range client.Destinations {
    if d == dest {
        idxdest = i
        break
    }
}
// If dest not found, uses index 0 silently!
stream, err := session[idxdest].OpenStream()
```

**Fix**:
```go
idxdest := -1
for i, d := range client.Destinations {
    if d == dest {
        idxdest = i
        break
    }
}

if idxdest == -1 {
    return fmt.Errorf("destination %s not found in connections list", dest)
}

if idxdest >= len(session) {
    return fmt.Errorf("index %d out of range for %d connections", idxdest, len(session))
}

stream, err := session[idxdest].OpenStream()
```

**Test**:
```bash
go test -v ./pkg/routing ./pkg/client -run TestBounds
```

---

### 3.2 - Fix Resource Leaks

#### a) Client connection cleanup

**File**: `pkg/client/client.go:193-199`

**Current** (LEAK):
```go
for _, d := range destinations {
    c, err := quic.DialAddrEarly(ctx, d, tlsConf, quicConf)
    if err != nil {
        return err  // Previous connections not closed!
    }
    client.Connections = append(client.Connections, c)
}
```

**Fix**:
```go
for _, d := range destinations {
    c, err := quic.DialAddrEarly(ctx, d, tlsConf, quicConf)
    if err != nil {
        // Clean up previously opened connections
        for _, conn := range client.Connections {
            conn.CloseWithError(0, "cleanup after connection error")
        }
        return fmt.Errorf("failed to connect to %s: %w", d, err)
    }
    client.Connections = append(client.Connections, c)
}
```

#### b) Proxy connection resource leak

**File**: `pkg/routing/proxy.go:94-101`

**Current** (LEAK):
```go
s, err := quic.DialAddrEarly(ctx, addr, tlsConf, quicConf)
sessions[addr] = s  // s might be nil
if err != nil {
    return err
}
defer s.CloseWithError(...)  // Panics if s is nil
```

**Fix**:
```go
s, err := quic.DialAddrEarly(ctx, addr, tlsConf, quicConf)
if err != nil {
    return fmt.Errorf("failed to dial %s: %w", addr, err)
}

sessions[addr] = s

defer func() {
    if s != nil {
        s.CloseWithError(0, "closing connection")
    }
}()
```

#### c) Stream close error handling

**File**: `pkg/server/server.go:264-266`

**Current** (SILENT):
```go
err := stream.Close()
if err != nil {
    return  // Error ignored!
}
```

**Fix**:
```go
err := stream.Close()
if err != nil {
    fmt.Printf("Error closing stream: %v\n", err)  // Log the error
    // Continue - stream will eventually timeout
}
```

**Test**:
```bash
# Run with -race flag and check for resource leaks
go test -race -timeout 10s ./pkg/client ./pkg/routing -v
```

---

### 3.3 - Fix Goroutine Management

**File**: `pkg/server/server.go:164-166, 170-177`

**Current** (ISSUES):
```go
// Workers spawned but never cleaned up
for i := range make([]int, server.Workers) {
    go worker(i, requestQueue, ForwardDecision, ForwardSetLastResponse)
}

// No graceful shutdown
for {
    sess, err := listener.Accept(context.Background())
    if err != nil {
        return err
    }
    go newRequest(sess, requestQueue, ForwardBlock)
}
```

**Solution - Add context and graceful shutdown**:

```go
func Run(server *Server, quicConf *quic.Config,
    ForwardDecision func(*util.RoPEMessage, *map[string]quic.EarlyConnection, int64) bool,
    ForwardSetLastResponse func(*util.RoPEMessage),
    ForwardBlock func(*util.RoPEMessage, quic.Stream)) error {

    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    requestQueue := make(chan JobRequest, server.BufSize)
    defer close(requestQueue)

    listener, err := quic.ListenAddrEarly(server.ListenAddr, util.GenerateTLSConfig(), quicConf)
    if err != nil {
        return fmt.Errorf("error creating listener: %v", err)
    }
    defer listener.Close()

    // Start worker pool with WaitGroup
    var wg sync.WaitGroup

    for i := range make([]int, server.Workers) {
        wg.Add(1)
        go func(id int) {
            defer wg.Done()
            worker(id, ctx, requestQueue, ForwardDecision, ForwardSetLastResponse)
        }(i)
    }

    fmt.Printf("Server ready %s, workers %d, queue length=%d\n", server.ListenAddr, server.Workers, server.BufSize)

    // Accept connections in a goroutine so we can cancel
    sessionErrors := make(chan error, 1)
    go func() {
        for {
            select {
            case <-ctx.Done():
                return
            default:
            }

            sess, err := listener.Accept(ctx)
            if err != nil {
                sessionErrors <- err
                return
            }

            go newRequest(sess, requestQueue, ForwardBlock)
        }
    }()

    // Wait for signal or error
    select {
    case err := <-sessionErrors:
        if err != nil {
            fmt.Printf("Accept error: %v\n", err)
        }
    case <-ctx.Done():
        fmt.Println("Server shutdown initiated")
    }

    // Wait for all workers to finish
    wg.Wait()
    fmt.Println("All workers stopped")

    return nil
}

func worker(id int, ctx context.Context, queue <-chan JobRequest, ...) {
    for {
        select {
        case <-ctx.Done():
            return
        case request, ok := <-queue:
            if !ok {
                return  // Channel closed
            }
            // Process request...
        }
    }
}
```

**Test**:
```bash
# Verify goroutines don't leak
go test ./pkg/server -v -race -count=10
# Check goroutine count before/after
```

---

### 3.4 - Fix Logic Errors

#### a) probabilityLatency.go - Request/Response asymmetry

**File**: `pkg/routing/probabilityLatency.go:124-136`

**Current** (LOGIC BUG):
```go
// In PLDecision (request processing):
req.Destination = sinks[i]  // Route to destination i

// In PLSetLastResponse (response processing):
for i < len(sinks) && sinks[i] != lastResp.Destination {
    i++
}
cd += int64(len(lastResp.Body))  // Attribute to sinks[i]
// But should find based on Destination, not Source
```

**Analysis**: Check if `lastResp.Destination` should be `lastResp.Source`

**Fix** (if symmetry needed):
```go
func PLSetLastResponse(lastResp *util.RoPEMessage) {
    if lastResp.Type == util.Response {
        // Find which sink this response came from
        i := 0
        for i < len(sinks) && sinks[i] != lastResp.Source {  // Use Source, not Destination
            i++
        }
        if i < len(sinks) {
            countdown[i]++
            cd += int64(len(lastResp.Body))
            util.Mavg_push(&b_down_i[i], int64(len(lastResp.Body)))
        }
    }
}
```

**Or** (if reverse path tracking):
```go
// Document the intention - responses come FROM destination back TO source
// So we need bidirectional tracking
```

**Test**:
```bash
go test ./pkg/routing -v -run Latency
# Verify bandwidth accounting is symmetric
```

#### b) Remove dead code

**File**: `pkg/routing/fixed.go`

**Delete** lines 59-61:
```go
// DELETE THIS:
for i, s := range d {
    fmt.Printf("\tUplink %s:  %f bytes/s\n", s, util.Mavg_eval(b_up_i[i], ...))
}
```

**Test**:
```bash
go build ./pkg/routing/fixed.go
```

---

### 3.5 - Fix os.Exit() in Goroutine

**File**: `pkg/server/server.go:186-188`

**Current** (DANGEROUS):
```go
if err.Error() == "Application error 0x1337: Finish" {
    os.Exit(0)  // Immediate shutdown - data loss risk
}
```

**Fix - Use context cancellation**:
```go
func newRequest(sess quic.Connection, queue chan<- JobRequest,
    ForwardBlock func(*util.RoPEMessage, quic.Stream),
    shutdownChan chan<- struct{}) {  // Add shutdown signal channel

    for {
        stream, err := sess.AcceptStream(context.Background())
        if err != nil {
            fmt.Println("Accept Stream error:", err)
            if err.Error() == "Application error 0x1337: Finish" {
                shutdownChan <- struct {}{}  // Signal graceful shutdown
                return
            }
            return
        }
        go newRequestServer(stream, queue, ForwardBlock)
    }
}
```

**Test**:
```bash
go test ./pkg/server -v -run Shutdown
```

---

### 3.6 - Phase 3 Testing

```bash
# Comprehensive testing
go test -race ./pkg/... -v

# Resource leak detection
GODEBUG=gctrace=1 go test ./pkg/... -timeout 30s

# Stress test with many connections
go test ./pkg/... -v -count=20

# Test with -race flag consistently
go test -race -timeout 60s ./...
```

**Success Criteria**:
- ✅ No index out of bounds errors
- ✅ No resource leaks detected
- ✅ Graceful shutdown works
- ✅ Logic errors fixed
- ✅ All edge cases handled
- ✅ -race flag shows no data races

---

**Estimated Effort**: 10-14 hours
**Complexity**: High
**Risk**: Medium-High (touches core logic)

---

# PHASE 4: POLISH - PERFORMANCE & EDGE CASES

**Duration**: 2-3 days
**Risk**: Low
**Impact**: Low (nice-to-have improvements)
**Blockers**: Phase 1, 2, & 3

## Goal
Optimize performance, improve observability, and handle edge cases.

### 4.1 - Add Comprehensive Logging

**Changes**:
- Add structured logging with levels (debug, info, warn, error)
- Log all error paths
- Add request/response tracking

### 4.2 - Performance Optimizations

**Changes**:
- Profile critical paths
- Optimize histogram bin calculations
- Reduce lock contention in hot paths

### 4.3 - Edge Case Handling

**Changes**:
- Handle zero-sized requests
- Handle empty destination lists
- Handle connection timeouts gracefully

### 4.4 - Documentation Updates

**Changes**:
- Update API docs to show error returns
- Add migration guide for breaking changes
- Document best practices

---

**Estimated Effort**: 6-8 hours
**Complexity**: Low
**Risk**: Low

---

# IMPLEMENTATION CHECKLIST

## Phase 1: Foundation
- [ ] Fix undefined variable `d` in probabilityLatency.go:126
- [ ] Fix undefined variables in fixed.go:59-60
- [ ] Add mutex to server.go global state
- [ ] Add mutex to probability.go globals
- [ ] Add mutex to probabilityLatency.go globals
- [ ] Add mutex to fixed.go globals
- [ ] Add mutex to logger.go globals
- [ ] Fix nil channel checks in proxy.go
- [ ] Test with -race flag
- [ ] Code review completed

## Phase 2: Stability
- [ ] Update GenerateTLSConfig to return error
- [ ] Update InitDelay with validation
- [ ] Update Delay with type checking
- [ ] Fix configuration type assertions in client.go
- [ ] Fix error message in server.go:114
- [ ] Update all call sites for new error returns
- [ ] Test all error paths
- [ ] Code review completed

## Phase 3: Quality
- [ ] Add bounds checking for array access
- [ ] Fix client connection cleanup
- [ ] Fix proxy connection handling
- [ ] Fix stream close error handling
- [ ] Add graceful shutdown with context
- [ ] Fix request/response asymmetry in latency routing
- [ ] Delete dead code
- [ ] Remove os.Exit() from goroutines
- [ ] Stress test with concurrency
- [ ] Code review completed

## Phase 4: Polish
- [ ] Add comprehensive logging
- [ ] Performance profiling
- [ ] Edge case testing
- [ ] Update documentation
- [ ] Code review completed
- [ ] Final integration testing

---

# ROLLOUT STRATEGY

### Week 1: Phase 1 & 2
- Days 1-4: Phase 1 implementation
- Days 4-5: Phase 2 implementation
- Testing & reviews

### Week 2: Phase 3
- Days 1-5: Phase 3 implementation
- Stress testing & performance validation
- Code reviews & adjustments

### Week 3: Phase 4 & Release
- Days 1-2: Phase 4 improvements
- Days 2-3: Integration testing
- Release candidate

---

# VERSION BUMPING

- **Current**: v0.x.x
- **After Phase 1**: v0.x+1.0 (bug fixes, internal refactoring)
- **After Phase 2**: v0.x+1.0 (API changes: error returns)
- **After Phase 3**: v0.x+2.0 (major improvements)
- **After Phase 4**: v1.0.0 (production ready)

---

# TESTING STRATEGY

### Unit Tests
```bash
go test ./pkg/... -v -run Unit
```

### Integration Tests
```bash
go test ./pkg/... -v -run Integration
```

### Race Detection
```bash
go test -race ./... -timeout 60s
```

### Stress Tests
```bash
# Run 1000 concurrent requests
go test -v -count=1000 ./pkg/...
```

### Benchmarks
```bash
go test -bench=. -benchmem ./pkg/...
```

---

# RISK MITIGATION

| Risk | Probability | Impact | Mitigation |
|------|-------------|--------|-----------|
| Regression in core logic | Medium | High | Comprehensive testing, code reviews |
| Performance degradation | Low | Medium | Benchmarking before/after |
| API compatibility break | High | Medium | Deprecation period, migration guide |
| Incomplete sync fix | High | High | Multiple code reviewers, extended testing |
| Resource leak reintroduction | Medium | High | -race flag testing in CI |

---

# SUCCESS CRITERIA

### Overall
- ✅ All 31 bugs fixed
- ✅ No compile errors
- ✅ -race flag passes
- ✅ All tests pass
- ✅ Documentation updated
- ✅ No performance regression

### Phase 1: Sync
- ✅ No undefined variables
- ✅ -race flag passes
- ✅ No panics on concurrent access

### Phase 2: Error Handling
- ✅ No panics on invalid input
- ✅ All errors returned with context
- ✅ Clear error messages

### Phase 3: Resource Management
- ✅ No resource leaks
- ✅ Graceful shutdown works
- ✅ Bounds checking passes

### Phase 4: Polish
- ✅ Comprehensive logging
- ✅ Performance verified
- ✅ Edge cases handled

---

# ESTIMATED TIMELINE

| Phase | Duration | Start | End | Effort |
|-------|----------|-------|-----|--------|
| 1: Foundation | 3-4 days | Day 1 | Day 4 | 8-12 hrs |
| 2: Stability | 2-3 days | Day 5 | Day 7 | 6-8 hrs |
| 3: Quality | 3-5 days | Day 8 | Day 13 | 10-14 hrs |
| 4: Polish | 2-3 days | Day 14 | Day 16 | 6-8 hrs |
| **TOTAL** | **~3 weeks** | | | **30-42 hrs** |

---

# NEXT STEPS

1. ✅ Review this plan
2. ✅ Get stakeholder approval
3. ✅ Set up git branches per phase
4. ✅ Create corresponding issues
5. ✅ Begin Phase 1 implementation
6. ✅ Daily standups for progress
7. ✅ Code reviews per phase
8. ✅ Release Phase 1 as hotfix

---

**Plan Version**: 1.0
**Last Updated**: 2026-03-20
**Created By**: Code Analysis Agent
