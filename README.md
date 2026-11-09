# minepulse

[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/PixelEzz/minepulse)](https://goreportcard.com/report/github.com/PixelEzz/minepulse)
[![Platform](https://img.shields.io/badge/platform-linux%20%7C%20windows%20%7C%20darwin-lightgrey)](https://github.com/PixelEzz/minepulse)

`minepulse` is a high-performance, single-binary CLI diagnostic utility designed for Minecraft server administrators, infrastructure engineers, and hosting providers. 

It provides protocol-level Server List Ping (SLP) breakdown, real-time TCP jitter and latency benchmarking (p50/p95/p99), and non-destructive configuration sanitation audits for Paper, Purpur, Velocity, and Vanilla deployments.

---

## Key Features

- **Native SLP Protocol Implementation**: Implements Minecraft 1.7+ Server List Ping over native Go TCP sockets with zero external networking dependencies.
- **Microsecond Latency Profiling**: Isolates TCP connection time, handshake round-trip, and ping/pong round-trip into separate diagnostic measurements.
- **Rule-Based Config Auditor**: Scans `server.properties`, `paper-global.yml`, `paper-world-defaults.yml`, and `velocity.toml` against industry-standard production baselines.
- **Benchmarking Engine**: Measures connection jitter, latency percentiles (`p50`, `p95`, `p99`), and packet drop rates over repeated probes.
- **Automation & CI/CD Ready**: Every subcommand supports `--json` for headless ingestion into Prometheus exporters, Grafana, or bash orchestration pipelines.
- **Zero Panics & Modern Error Wrapping**: Fully context-aware deadlines (`context.WithTimeout`) and robust error propagation (`fmt.Errorf("...: %w", err)`).

---

## Installation

### Via `go install`
```bash
go install github.com/PixelEzz/minepulse@latest
```

### Pre-built Binaries
Download the latest pre-compiled binary for your architecture from [GitHub Releases](https://github.com/PixelEzz/minepulse/releases):
- Linux (`amd64`, `arm64`)
- Windows (`amd64`)
- macOS (`amd64`, `arm64`)

### Build from Source
```bash
git clone https://github.com/PixelEzz/minepulse.git
cd minepulse
go build -ldflags "-s -w" -o minepulse main.go
```

Or run cross-compilation with make:
```bash
make cross-compile
```

---

## Usage Guide

### 1. Protocol-Level Server Ping (`minepulse ping`)
Inspect server status, player counts, version protocol, and detailed latency metrics.

```bash
# Ping default port (25565)
minepulse ping mc.example.com

# Ping custom port with 3s timeout
minepulse ping 192.168.1.100:25577 --timeout 3s

# Machine-readable JSON output
minepulse ping mc.example.com --json
```

**Terminal Output Example:**
```text
SLP Diagnostics: mc.hypixel.net:25565 (209.222.115.35)
───────────────────────────────────────────────────────
  Server Version:          Requires MC 1.8-1.21 (Protocol 47)
  Players:                 42150 / 200000
  MOTD:                    Hypixel Network [1.8-1.21]
  Secure Chat:             Not Enforced

Network Latency Breakdown
─────────────────────────
  TCP Dial Latency:        12.4ms
  Handshake RTT:           14.2ms
  Ping/Pong RTT:           13.8ms
  Total Round-Trip:        40.4ms

minepulse v0.1.0 • Built by PixelEzz (https://github.com/PixelEzz) • Run with --json for raw output
```

---

### 2. Configuration Sanitation Audit (`minepulse audit`)
Analyze your server root directory for dangerous anti-patterns, memory leaks, and misconfigurations.

```bash
# Audit current directory
minepulse audit

# Audit specific server directory
minepulse audit /srv/minecraft/survival

# Export audit findings to JSON for automated CI checks
minepulse audit /srv/minecraft/survival --json
```

**Diagnostic Rules Inspected:**
- **`view-distance` & `simulation-distance`**: Detects excessive simulation radii on 1.20+ causing entity tick spikes and memory exhaustion.
- **`network-compression-threshold`**: Warns against hazardous thresholds (`< 64`) or unoptimized compression loops that overload Netty event threads.
- **`sync-chunk-writes`**: Detects synchronous chunk flushing that causes server-wide freeze frames during autosaves.
- **`player-info-forwarding-mode` & secrets**: Ensures Velocity proxy setups employ `modern` forwarding with valid shared secret tokens to prevent UUID injection attacks.
- **`optimize-explosions` & chunk loading boundaries**: Flags costly unoptimized explosion physics and entity chunk loading traps.

**Terminal Output Example:**
```text
Configuration Audit: /srv/minecraft/survival
────────────────────────────────────────────
  Files Inspected:         2 ([server.properties config/paper-world-defaults.yml])
  Health Score:            68/100
  Status:                  Suboptimal. Contains potential performance bottlenecks or memory hazards.

Findings & Action Items (2 detected)
─────────────────────────────────────

  [WARN] view-distance [server.properties]
      Current:     16
      Target:      6 - 10
      Impact:      Severe chunk packet serialization load, memory pressure, and tick spikes.
      Remediation: Reduce view-distance to 8 or 10, pair with No-Tick-View-Distance if on Paper.

  [CRIT] network-compression-threshold [server.properties]
      Current:     32
      Target:      256 - 512 (or -1 behind Velocity)
      Impact:      Compressing tiny packets generates extreme Netty CPU overhead without bandwidth benefit.
      Remediation: Set network-compression-threshold to 256, or disable (-1) if behind a proxy like Velocity.

minepulse v0.1.0 • Built by PixelEzz (https://github.com/PixelEzz) • Run with --json for raw output
```

---

### 3. Network Latency & Jitter Benchmark (`minepulse bench`)
Perform continuous TCP handshake diagnostics to measure peering stability, jitter, and packet loss.

```bash
# Benchmark target with 15 probe trials
minepulse bench mc.example.com -c 15

# High-frequency jitter check (50ms interval)
minepulse bench mc.example.com -c 20 -i 50ms

# JSON export for latency monitoring scripts
minepulse bench mc.example.com --json
```

**Terminal Output Example:**
```text
Benchmarking TCP handshake against mc.hypixel.net:25565 (10 trials)...

Benchmark Results: mc.hypixel.net:25565
──────────────────────────────────────
  Probes Transmitted:      10
  Probes Received:         10
  Packet Loss:             0.0%

Latency Percentiles & Jitter
────────────────────────────
  Minimum Latency:         12.1ms
  Mean Latency:            13.4ms
  p50 (Median):            13.1ms
  p95 Percentile:          15.8ms
  p99 Percentile:          16.2ms
  Maximum Latency:         16.2ms
  Calculated Jitter:       1.1ms

minepulse v0.1.0 • Built by PixelEzz (https://github.com/PixelEzz) • Run with --json for raw output
```

---

## Architecture Overview

```
minepulse/
├── cmd/                     # Command-line interface definitions (Cobra)
│   ├── root.go              # Global options, flags (--json, --timeout, --verbose)
│   ├── ping.go              # Protocol SLP diagnostic command
│   ├── audit.go             # Config safety and performance audit command
│   └── bench.go             # Handshake jitter & percentile benchmark command
├── internal/
│   ├── slp/                 # Zero-dependency Minecraft SLP implementation
│   │   ├── client.go        # Socket dialer, context timeouts, latency profiling
│   │   ├── packet.go        # VarInt encoding/decoding and packet framing
│   │   ├── types.go         # Modern & legacy status response JSON parser
│   │   └── slp_test.go      # SLP and VarInt edge-case tests
│   ├── analyzer/            # Config audit rule evaluation engine
│   │   ├── analyzer.go      # Paper/Velocity/Vanilla rules and scoring
│   │   └── analyzer_test.go # Test cases for config validation
│   └── terminal/            # TTY-aware ANSI chips, styling, and JSON export
│       ├── terminal.go      # Color badges, tables, and NO_COLOR detection
│       └── terminal_test.go # Terminal formatter tests
├── Makefile                 # Cross-platform compilation targets
├── LICENSE                  # MIT License
└── main.go                  # Application entry point
```

---

## Testing

Execute the test suite across all packages:
```bash
go test -v ./...
```

Run test suite with race detector:
```bash
go test -v -race ./...
```

---

## Author & License

- **Author**: PixelEzz ([GitHub: @PixelEzz](https://github.com/PixelEzz))
- **License**: MIT License. See [LICENSE](LICENSE) for full details.
