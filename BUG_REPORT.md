# Code Review: Bug Report

Generated: 2026-03-20
**Total Issues Found: 31**

---

## 🔴 CRITICAL ISSUES (10 - Must Fix)

### 1. Race Condition - Global sessions map (server.go:26, :251)
**Severity**: CRITICAL
**Issue**: `sessions` map accessed by multiple worker goroutines without mutex
```go
var sessions map[string]quic.EarlyConnection  // Line 26 - NO SYNCHRONIZATION
// Used in worker() at line 251 without locking
```
**Fix**: Wrap with sync.RWMutex

---

### 2. Race Condition - Global function pointers (server.go:28-30)
**Severity**: CRITICAL
**Issue**: Function pointers can be reassigned during execution
```go
var ForwardBlock func(*util.RoPEMessage, quic.Stream)
var ForwardDecision func(*util.RoPEMessage, *map[string]quic.EarlyConnection, int64) bool
var ForwardSetLastResponse func(*util.RoPEMessage)
// Called from goroutines while potentially reassigned
```
**Fix**: Use sync.RWMutex or pass as parameters

---

### 3. Undefined Variable - probabilityLatency.go:126
**Severity**: CRITICAL - **WILL CRASH**
**Issue**: References undefined variable `d`
```go
for i < len(d) && d[i] != lastResp.Destination {  // LINE 126
    // d doesn't exist! Should be 'sinks'
```
**Fix**: Change `d` to `sinks`

---

### 4. Undefined Variables - fixed.go:59-60
**Severity**: CRITICAL - **WILL CRASH**
**Issue**: References undefined variables `d` and `b_up_i`
```go
for i, s := range d {  // LINE 59 - d undefined
    fmt.Printf("\tUplink %s:  %f bytes/s\n", s, util.Mavg_eval(b_up_i[i], ...))  // b_up_i undefined
```
**Fix**: These belong in probability.go, not fixed.go

---

### 5. Race Condition - Global variables in probability.go (Lines 12-29)
**Severity**: CRITICAL
**Issue**: All globals accessed from multiple goroutines without synchronization
```go
var p, d []string  // Lines 14, 15
var cu, cd int64  // Lines 12, 13
var cup, cdw []int64  // Lines 16, 17
var stime = time.Time{}  // Line 18
var b_up, b_down util.Mavg  // Lines 22, 23
var b_up_i, b_down_i []util.Mavg  // Lines 24, 25
// All accessed without locks
```
**Fix**: Wrap with sync.RWMutex

---

### 6. Race Condition - Global variables in probabilityLatency.go (Lines 12-35)
**Severity**: CRITICAL
**Issue**: All globals accessed without synchronization
```go
var routingProb []string  // Line 17 - NO MUTEX
var sinks []string  // Line 18 - NO MUTEX
var countup, countdown []int64  // Lines 19-20
// ... many more globals ... all accessed from goroutines without locks
```
**Fix**: Implement sync.RWMutex protection

---

### 7. Race Condition - Global logger variables (util/logger.go:20-21, :70-72)
**Severity**: CRITICAL
**Issue**: Logger state accessed without synchronization
```go
var client influxdb2.Client  // Line 20 - global, no mutex
var writeAPI api.WriteAPIBlocking  // Line 21 - global, no mutex
// Accessed from multiple goroutines at lines 70-72, 112
```
**Fix**: Use sync.RWMutex or sync.Once

---

### 8. Race Condition - Global variables in fixed.go (Lines 14-23)
**Severity**: CRITICAL
**Issue**: Bandwidth metrics accessed without synchronization
```go
var cu, cd int64  // Lines 14, 15
var stim = time.Time{}  // Line 17
var b_u, b_d = util.NewMavg(twind), util.NewMavg(twind)  // Lines 22, 23
// All modified/read from multiple goroutines without locks
```
**Fix**: Add sync.RWMutex

---

### 9. Nil Channel Panic Risk - proxy.go:132, :240, :250, :271
**Severity**: CRITICAL
**Issue**: Channels used without nil check
```go
var tokensUp, tokensDn chan struct{}  // Lines 20-21

// Later:
if self.ParallelConn > 0 {
    tokensUp = make(chan struct{}, self.ParallelConn)
}
// If ParallelConn == 0, tokensUp is nil

// But used at line 132:
tokensUp <- struct{}{}  // PANIC - send on nil channel
```
**Fix**: Check if channels are nil before sending

---

### 10. Panics Instead of Errors - util.go:53-67
**Severity**: CRITICAL
**Issue**: GenerateTLSConfig crashes instead of returning error
```go
func GenerateTLSConfig() *tls.Config {
    key, err := rsa.GenerateKey(rand.Reader, 1024)
    if err != nil {
        panic(err)  // LINE 55 - CRASHES APP
    }
    // ... more panics at lines 59, 65, 67
}
```
**Fix**: Return error tuple: `(*tls.Config, error)`

---

## 🟠 HIGH SEVERITY ISSUES (6)

### 11. Unchecked Type Assertions - util.go:233, :238, :243
**Issue**: Delay() function panics on type assertion failure
```go
case "uniform":
    d := (distribution.(distuv.Uniform)).Rand()  // PANIC if wrong type
case "exponential":
    d := (distribution.(distuv.Exponential)).Rand()  // PANIC
case "constant":
    delay, _ = time.ParseDuration(distribution.(string))  // PANIC
```

### 12. Index Out of Bounds - probability.go:56
**Issue**: Empty array access without validation
```go
var pdest float64 = p[0]  // PANIC if p is empty
```

### 13. Index Out of Bounds - probabilityLatency.go:79
**Issue**: Same as above
```go
var pdest float64 = routingProb[0]  // PANIC if empty
```

### 14. Goroutine Leak - server.go:164-166
**Issue**: Workers spawn and never properly clean up
```go
for i := range make([]int, server.Workers) {
    go worker(i, requestQueue, ForwardDecision, ForwardSetLastResponse)
}
// If requestQueue channel closes, all workers will panic
// No graceful shutdown mechanism
```

### 15. Resource Leak - proxy.go:94-101
**Issue**: Nil connection before error check
```go
s, err := quic.DialAddrEarly(ctx, addr, tlsConf, quicConf)
sessions[addr] = s  // s might be nil
if err != nil {
    return err  // But s is already in map as nil
}
defer s.CloseWithError(...)  // PANIC - s is nil
```

### 16. os.Exit() in Goroutine - server.go:186-188
**Issue**: Direct exit without cleanup
```go
if err.Error() == "Application error 0x1337: Finish" {
    os.Exit(0)  // Immediate shutdown, no cleanup! Data loss risk
}
```

---

## 🟡 MEDIUM SEVERITY ISSUES (15)

### 17. Undefined Variable in Error Message - server.go:114
```go
} else {
    return errors.New("missing or invalid 'BufSize' in configuration[" +
        fmt.Sprintf("%d", bufSize) + "]")  // bufSize undefined here!
}
```

### 18. Index Without Bounds Check - client.go:232
```go
for i, d := range client.Destinations {
    if d == dest {
        idxdest = i
        break
    }
}
stream, err := session[idxdest].OpenStream()  // Could be wrong index
```

### 19-20. Unchecked Type Assertions - client.go:84, 102, 104
Multiple places where `.() ` assertions lack proper validation

### 21. Silent Errors - server.go:264-266
```go
err := stream.Close()
if err != nil {
    return  // Error silently ignored, not logged
}
```

### 22. Connection Leak on Error - client.go:193-199
```go
for _, d := range destinations {
    c, err := quic.DialAddrEarly(...)
    if err != nil {
        return err  // Previous connections not closed
    }
}
```

### 23-24. Empty Error Handlers - proxy.go:149-151, :229-231
Defer blocks with empty error handling

### 25. Logic Error - probabilityLatency.go:126
Asymmetrical comparison between request and response handling

### 26-27. Race Conditions in examples
- examples/client/main.go:95, :115-119 (app struct modified from multiple goroutines)
- examples/client/main.go:151, :230 (global histogram accessed from goroutines)

### 28. Panic Instead of Error - util.go:194-197
```go
if dist_params == nil {
    panic("No delay or distribution specified")  // Should return error
}
```

### 29. No Graceful Shutdown - server.go:170-177
```go
for {
    sess, err := listener.Accept(context.Background())
    if err != nil {
        return err  // Only way out is an error
    }
    // No way to stop without killing process
}
```

### 30. Mavg Without Synchronization - util.go:254-268
Mavg struct has no mutex but used from multiple goroutines

### 31. Silent Destination Error - client.go:223-228
```go
idxdest := 0  // Default value
for i, d := range client.Destinations {
    if d == dest {
        idxdest = i
        break
    }
}
// If dest not found, uses index 0 silently!
```

---

## Summary by Component

| Component | Critical | High | Medium | Total |
|-----------|----------|------|--------|-------|
| server.go | 1 | 1 | 3 | 5 |
| client.go | 0 | 0 | 5 | 5 |
| util.go | 1 | 3 | 2 | 6 |
| probability.go | 1 | 1 | 1 | 3 |
| probabilityLatency.go | 2 | 1 | 1 | 4 |
| fixed.go | 1 | 0 | 0 | 1 |
| proxy.go | 1 | 1 | 2 | 4 |
| logger.go | 1 | 0 | 0 | 1 |
| examples/*.go | 0 | 0 | 2 | 2 |
| **TOTAL** | **10** | **6** | **15** | **31** |

---

## Recommended Fix Priority

### Phase 1 (Must fix immediately - Production blocker):
- [ ] Issue #3: Fix undefined variable `d` in probabilityLatency.go:126
- [ ] Issue #4: Fix undefined variables in fixed.go:59-60
- [ ] Issue #1, #5-8: Add mutexes to all global state
- [ ] Issue #9: Check channels before using
- [ ] Issue #10: Return error from GenerateTLSConfig instead of panics

### Phase 2 (Should fix - Crashes possible):
- [ ] Issue #11-13: Add type checks before assertions
- [ ] Issue #14-16: Fix resource leaks and goroutine issues

### Phase 3 (Should fix - Quality/Maintainability):
- [ ] Issue #17-24: Add proper error handling
- [ ] Issue #28-31: Fix logic errors and silent failures

---

## File Status

| File | Status | Issues | Priority |
|------|--------|--------|----------|
| probabilityLatency.go | 🔴 Broken | 3 critical | FIX FIRST |
| fixed.go | 🔴 Broken | 1 critical | FIX FIRST |
| server.go | 🟡 Needs work | 5 | High |
| probability.go | 🟡 Needs work | 3 | High |
| util.go | 🟠 Has issues | 6 | High |
| client.go | 🟠 Has issues | 5 | Medium |
| proxy.go | 🟠 Has issues | 4 | Medium |
| logger.go | 🟠 Has issues | 1 | Medium |

---

**Report Generated**: 2026-03-20
