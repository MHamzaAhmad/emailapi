# Performance Testing Suite

A professional performance testing framework for the SimpleEmailAPI, supporting both **gRPC** and **HTTP** endpoints.

## 📊 Golden Metrics

Don't just look at averages. Track the full latency distribution:

| Metric | Description | Target |
|--------|-------------|--------|
| **P50** | Median latency (typical user) | < 50ms |
| **P95** | Typical worst case | < 200ms |
| **P99** | Tail latency (exposes bottlenecks) | < 500ms |
| **RPS** | Requests per second | Varies by plan |

---

## 🛠️ Tools Required

### ghz (gRPC Load Testing)
```bash
# macOS
brew install ghz

# Linux
go install github.com/bojand/ghz/cmd/ghz@latest

# Or download binary from: https://github.com/bojand/ghz/releases
```

### Automated VM Setup (Recommended)

For setting up a fresh Linux VM with all required tools:

```bash
# Clone repo and run setup
git clone https://github.com/MHamzaAhmad/emailapi.git
cd emailapi/tools/perf
./setup-vm.sh
```

This installs: k6, ghz, Go, jq, and sets up your environment.

---

### k6 (HTTP Load Testing)
```bash
# macOS
brew install k6

# Linux
sudo gpg -k
sudo gpg --no-default-keyring --keyring /usr/share/keyrings/k6-archive-keyring.gpg --keyserver hkp://keyserver.ubuntu.com:80 --recv-keys C5AD17C747E3415A3642D57D77C6C491D6AC1D69
echo "deb [signed-by=/usr/share/keyrings/k6-archive-keyring.gpg] https://dl.k6.io/deb stable main" | sudo tee /etc/apt/sources.list.d/k6.list
sudo apt-get update
sudo apt-get install k6
```

---

## 🚀 Quick Start

### 1. Set Environment Variables
```bash
# Required
export PERF_API_KEY="your-api-key-here"
export PERF_FROM_EMAIL="test@yourverified.domain"
export PERF_TO_EMAIL="loadtest-sink@example.com"

# Optional (defaults shown)
export PERF_API_HOST="api.simpleemailapi.dev"
export PERF_GRPC_PORT="443"
export PERF_HTTP_PORT="443"
```

### 2. Run Full Benchmark (Recommended)

Run a complete benchmark across all modes with a single command:

```bash
# Full benchmark with dry-run (saves SES quota)
./benchmark.sh --dry-run

# Quick benchmark (5s per test instead of 15s)
./benchmark.sh --dry-run --quick

# Full benchmark with actual email sending
./benchmark.sh

# HTTP or gRPC only
./benchmark.sh --dry-run --http-only
./benchmark.sh --dry-run --grpc-only
```

This runs all combinations:
- **HTTP + gRPC** endpoints
- **Sync + Async** modes  
- **10, 50, 100 RPS** load levels

And generates a consolidated markdown report in `results/`.

### 3. Individual Tests (Optional)

For more control, run specific test scenarios:

```bash
# Smoke test (10 RPS, 30s)
./run-perf.sh smoke --dry-run

# Load test (100 RPS, 2m)
./run-perf.sh load --dry-run

# Stress test (find breaking point)
./run-perf.sh stress --dry-run

# Quick single request test
./quick-test.sh --dry-run
```

### 4. 🆘 SES Sandbox Mode

If you're in SES sandbox, **always use `--dry-run`** mode:

**How dry-run works:**
- **Per-request mode**: `X-Dry-Run: true` header (used by test scripts)
- **Global mode**: Set `DRY_RUN=true` environment variable on API server

Both modes:
- ✅ Exercise **full API path** (auth, validation, rate limiting, DB lookups)
- ✅ Simulate **realistic SES latency** (~50-150ms random delay)
- ✅ Work for both **sync and async** modes
- ❌ Do NOT send actual emails
- 🎯 Perfect for benchmarking without burning SES quota

---

## 📁 File Structure

```
tools/perf/
├── README.md           # This file
├── benchmark.sh        # 🚀 Unified benchmark runner (recommended)
├── run-perf.sh         # Individual test runner
├── quick-test.sh       # Single request test
├── config.env          # Environment template
├── http/
│   ├── smoke.js        # k6 script for smoke test
│   ├── load.js         # k6 script for load test
│   └── stress.js       # k6 script for stress test
└── results/            # Output directory (gitignored)
    └── benchmark-report-*.md  # Generated reports
```

---

## 🎯 Test Scenarios

### A. Smoke Test (Baseline)
- **Purpose:** Measure "clean" latency with minimal load
- **Config:** 10 RPS for 30 seconds
- **What to look for:** This is your performance "floor"

### B. Load Test (Realistic)
- **Purpose:** Simulate typical production traffic
- **Config:** 100 RPS sustained for 2 minutes
- **What to look for:** Does P99 stay under 500ms? If P99 spikes while P50 stays flat, you have a concurrency bottleneck.

### C. Stress Test (Breaking Point)
- **Purpose:** Find maximum capacity before errors
- **Config:** Ramp from 50 to 500 RPS over 5 minutes
- **What to look for:** At what RPS do you start seeing 5xx errors or gRPC UNAVAILABLE?

---

## ⚠️ Testing Best Practices

1. **Don't test from the same VM** - Run tests from a different machine to avoid CPU contention
2. **Test through NGINX** - Always test the public URL, not internal ports
3. **Warm up first** - Let the test run for 60s before recording "real" results
4. **Watch resources** - Run `htop` on your server during tests to monitor CPU/RAM
5. **Use a sink email** - Don't spam real addresses; use a test address that discards mail

---

## 📈 Interpreting Results

### gRPC (ghz) Output
```
Summary:
  Count:        1000
  Total:        10.23 s
  Slowest:      234.56 ms
  Fastest:      12.34 ms
  Average:      45.67 ms
  Requests/sec: 97.78

Response time histogram:
  12.34  [100] |∎∎∎∎∎∎
  34.56  [450] |∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎
  ...

Latency distribution:
  50% in 42.12 ms    <- P50 (median)
  75% in 56.78 ms
  90% in 89.12 ms
  95% in 123.45 ms   <- P95
  99% in 198.76 ms   <- P99 (tail)
```

### HTTP (k6) Output
```
     ✓ status is 200
     ✓ latency < 500ms

     http_req_duration..............: avg=45.12ms  min=12.34ms  med=42.56ms  max=234.56ms  p(95)=123.45ms  p(99)=198.76ms
     http_reqs......................: 1000    97.78/s
     iteration_duration.............: avg=1.04s   min=1.01s    med=1.04s    max=1.23s     p(95)=1.12s     p(99)=1.19s
```

### Red Flags to Watch For
- **P99 >> P50:** Tail latency issue (GC pauses, connection pool exhaustion, or cold cache)
- **Errors > 0.1%:** Capacity limit reached
- **RPS drops at higher load:** Backpressure or rate limiting kicking in
