# Random Traffic Consumer

A high-performance network bandwidth stability testing tool written in Go. It uses multiple concurrent download threads to stress-test your network connection over extended periods, helping you verify ISP performance and network stability.

## Features

- **Multi-threaded Downloads**: Configurable concurrent worker pool for maximum throughput
- **Staggered Worker Startup**: Optional staggered startup with jitter to prevent connection burst
- **Flexible Stop Conditions**: Stop by duration, traffic limit, or both
- **Real-time Bandwidth Limiting**: Token bucket algorithm to cap download speed
- **Mixed Download Strategies**: Complete file or chunked downloads with HTTP Range support
- **Live Statistics**: Real-time display of speed, traffic, and success/failure rates
- **Thread-safe Stats Collection**: Atomic operations for accurate metrics without locks
- **Cross-platform**: Works on Windows (10 1607+), Linux, and macOS
- **Configuration-driven**: All settings via YAML config file
- **Graceful Shutdown**: Responds to Ctrl+C signal

## Development Methodology

This project follows **OpenSpec**-driven development. All features are specified in `openspec/specs/` before implementation, ensuring clear requirements and testable scenarios. See `openspec/specs/` for detailed specifications of each component.

## Installation

### From Source

```bash
git clone https://github.com/yourusername/random-traffic-consumer.git
cd random-traffic-consumer
go mod download
go build -o random-traffic-consumer .
```

### Prerequisites

- Go 1.25 or higher
- A `config.yaml` file in the working directory

## Usage

1. Configure `config.yaml` with your settings
2. Run the tool:

```bash
# Use default config.yaml
./random-traffic-consumer

# Specify custom config
./random-traffic-consumer -config /path/to/config.yaml
```

3. Press `Ctrl+C` to stop at any time

## Configuration

### Basic Configuration

```yaml
# Download target URLs (CDN resources for testing)
urls:
  - "https://example.com/file1.mp4"
  - "https://example.com/file2.mp4"

# Download settings
download:
  workers: 64              # Number of concurrent workers
  timeout: 30s             # Per-request timeout
  retry: 3                 # Retry attempts on failure
  chunk_size: 0            # 0 = complete download, >0 = chunked (bytes)
  stagger_start:           # Optional: staggered worker startup
    enabled: false         # Enable to distribute worker startup over time
    base_delay: 200ms      # Base delay between worker starts
    jitter: 50ms           # Random jitter (±) to add to base delay

# Stop condition
stop:
  mode: duration           # Options: duration, traffic, both
  duration: 1h             # Run duration (for mode: duration or both)
  traffic_limit: 107374182400  # 100 GB in bytes (for mode: traffic or both)

# Optional bandwidth limiting
bandwidth_limit:
  enabled: false           # Enable bandwidth limiting
  max: "200 Mbps"          # Maximum download speed (string with unit)

# HTTP client settings
http:
  max_idle_conns: 100      # Maximum idle connections
  idle_conn_timeout: 90s   # Idle connection timeout
  disable_keepalive: false # Disable HTTP keep-alive

# Output settings
output:
  refresh_interval: 500ms  # Console refresh interval
```

### Configuration Options

| Section | Option | Type | Description |
|---------|--------|------|-------------|
| `urls` | array | string list | Download target URLs (at least one required) |
| `download.workers` | int | Number of concurrent goroutines | Must be positive |
| `download.timeout` | duration | Per-request timeout | Valid units: s, m, h |
| `download.retry` | int | Max retry attempts | Non-negative |
| `download.chunk_size` | int64 | 0=complete, >0=chunked bytes | Non-negative |
| `download.stagger_start.enabled` | bool | Enable staggered startup | Optional (default: false) |
| `download.stagger_start.base_delay` | duration | Base delay between workers | Default: 200ms |
| `download.stagger_start.jitter` | duration | Random jitter (±) | Default: 50ms, must be ≤ base_delay |
| `stop.mode` | string | duration/traffic/both | Required |
| `stop.duration` | duration | Run time limit | Required for mode: duration, both |
| `stop.traffic_limit` | int64 | Traffic limit in bytes | Required for mode: traffic, both |
| `bandwidth_limit.enabled` | bool | Enable speed cap | Optional |
| `bandwidth_limit.max` | string | Max speed (e.g., "200 Mbps") | Required if enabled |
| `http.max_idle_conns` | int | Maximum idle connections | Non-negative |
| `http.idle_conn_timeout` | duration | Idle connection timeout | Non-negative |
| `http.disable_keepalive` | bool | Disable HTTP keep-alive | Optional |
| `output.refresh_interval` | duration | Console update rate | Must be positive |

**Notes:**
- `traffic_limit` is specified in raw bytes (e.g., 107374182400 for 100 GB)
- Conversion: 1 KB = 1024 B, 1 MB = 1024² B, 1 GB = 1024³ B, 1 TB = 1024⁴ B

## Use Cases

### Test sustained bandwidth for 1 hour

```yaml
stop:
  mode: duration
  duration: 1h

bandwidth_limit:
  enabled: false
```

### Download exactly 100 GB of data

```yaml
stop:
  mode: traffic
  traffic_limit: 107374182400  # 100 GB in bytes

bandwidth_limit:
  enabled: false
```

### Simulate 200 Mbps connection for 2 hours

```yaml
stop:
  mode: duration
  duration: 2h

bandwidth_limit:
  enabled: true
  max: "200 Mbps"
```

### Dual limit: 2 hours OR 50 GB (whichever first)

```yaml
stop:
  mode: both
  duration: 2h
  traffic_limit: 53687091200  # 50 GB in bytes
```

### Test connection stability with frequent reconnects

```yaml
download:
  workers: 64
  chunk_size: 10485760  # 10MB chunks forces reconnects
```

### Smooth startup with staggered worker launch

By default, all workers start simultaneously, which can cause connection bursts. Staggered startup distributes worker launches over time:

```yaml
download:
  workers: 64
  stagger_start:
    enabled: true
    base_delay: 200ms   # Each worker starts 200ms after the previous
    jitter: 50ms        # Add random ±50ms to prevent synchronization
```

**Trade-offs:**
- **Smoother startup**: Reduces initial bandwidth spike and connection burst
- **Longer time-to-full-capacity**: With 64 workers at 200ms delay, last worker starts after ~12.6 seconds
- **Recommended for**: High worker counts (16+), rate-limited environments, shared networks

**Example configurations by worker count:**

| Workers | Base Delay | Total Startup Time | Use Case |
|---------|------------|-------------------|----------|
| 4-8 | 100ms | ~0.4-0.8s | Low worker count, fast ramp-up |
| 8-16 | 150ms | ~1.2-2.4s | Medium worker count |
| 16-32 | 200ms | ~3.2-6.4s | High worker count (default) |
| 32-64 | 200ms | ~6.4-12.8s | Very high worker count |
| 64+ | 250ms | 16s+ | Extreme worker counts |

## Output Example

### Startup Banner

```
===========================================
   Random Traffic Consumer
   Network Bandwidth Stability Test Tool
===========================================
Workers: 64
URLs in pool: 5
Stop mode: duration
Duration: 1h
===========================================
Press Ctrl+C to stop
```

### Running Display (duration mode)

```
============================================================
        Random Traffic Consumer - Running
============================================================

Traffic
------------------------------------------------------------
Downloaded:   12.50 GB
Speed:        185.30 Mbps

Time
------------------------------------------------------------
Elapsed:     00:45:23
Remaining:   00:14:37 / 01:00:00  (75.38%)

Requests
------------------------------------------------------------
Workers:      64
Success:      15432
Failed:       23
```

### Running Display (both mode with bandwidth limit)

```
============================================================
        Random Traffic Consumer - Running
============================================================

Traffic
------------------------------------------------------------
Downloaded:   25.50 GB / 50.00 GB  (51.00%)
Speed:        190.25 Mbps

Time
------------------------------------------------------------
Elapsed:     00:30:15
Remaining:   00:29:45 / 01:00:00  (50.42%)

Requests
------------------------------------------------------------
Workers:      64
Success:      10245
Failed:       12

Bandwidth Limit
------------------------------------------------------------
Limit:        200.00 Mbps
Usage:        95.12%
```

### Final Statistics

```
^C

===========================================
               Final Statistics
===========================================
Total traffic: 25.82 GB
Duration: 00:30:31
Average speed: 188.5 Mbps
Successful downloads: 10256
Failed downloads: 12
===========================================
```

## Platform Support

| Platform | Status | Notes |
|----------|--------|-------|
| Windows 10 1607+ | ✅ Fully supported | ANSI escape sequence support |
| Linux | ✅ Fully supported | Requires glibc |
| macOS | ✅ Fully supported | Intel and Apple Silicon |

## Architecture

```
┌─────────────────────────────────────────┐
│           Main Goroutine                │
│  - Config loading & validation          │
│  - Signal handling (Ctrl+C)             │
│  - Stop condition monitoring            │
│  - Console refresh coordination         │
└─────────────────────────────────────────┘
                  │
                  ▼
┌─────────────────────────────────────────┐
│        Worker Pool (N workers)          │
│  ┌─────┐ ┌─────┐ ┌─────┐ ┌─────┐       │
│  │ W1  │ │ W2  │ │ ... │ │ WN  │       │
│  └──┬──┘ └──┬──┘ └──┬──┘ └──┬──┘       │
│     │       │       │       │           │
└─────┼───────┼───────┼───────┼───────────┘
      ▼       ▼       ▼       ▼
┌─────────────────────────────────────────┐
│      Atomic Statistics Collection       │
│  - totalBytes (int64, atomic)            │
│  - successCount (int64, atomic)          │
│  - errorCount (int64, atomic)            │
│  - currentSpeed (float64)                │
│  - averageSpeed (float64)                │
└─────────────────────────────────────────┘
      │
      ▼
┌─────────────────────────────────────────┐
│     Optional: Token Bucket Limiter      │
│  - Enforces bandwidth cap               │
│  - FIFO token distribution              │
└─────────────────────────────────────────┘
```

### Project Structure

```
random-traffic-consumer/
├── main.go                      # Application entry point
├── config.yaml                  # Configuration file
├── go.mod                       # Go module definition
├── openspec/
│   └── specs/                   # OpenSpec specifications
│       ├── multi-thread-downloader/
│       ├── bandwidth-limiter/
│       ├── download-strategies/
│       ├── stop-strategies/
│       ├── config-management/
│       ├── stats-monitoring/
│       └── console-ui/
└── internal/
    ├── config/
    │   ├── config.go           # Configuration loading & validation
    │   └── stopchecker.go      # Stop condition monitoring
    ├── downloader/
    │   ├── pool.go             # Worker pool management
    │   ├── worker.go           # Individual worker logic
    │   └── client.go           # HTTP client with connection pooling
    ├── limiter/
    │   └── tokenbucket.go      # Token bucket bandwidth limiter
    ├── stats/
    │   └── collector.go        # Thread-safe statistics collection
    └── ui/
        └── console.go          # Console output formatting
```

## License

MIT License - see LICENSE file for details

## Contributing

Contributions are welcome! This project follows OpenSpec-driven development:

1. **Fork** the repository
2. **Create a spec** in `openspec/specs/` for new features
3. **Implement** according to the specification
4. **Test** thoroughly before submitting
5. **Submit a pull request** with your changes

Please ensure all contributions include:
- Updated specifications for new features
- Code that matches the specifications
- Updated documentation as needed
