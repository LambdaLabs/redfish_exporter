# Performance Analysis Tool

This tool analyzes the performance of redfish_exporter scrapes to identify bottlenecks and measure latency.

## Quick Start

### Test against mock server
```bash
make perf-mock
```

### Test against live system

Using existing config.yml:
```bash
make perf-live TARGET=192.168.1.100
```

With credentials via environment:
```bash
REDFISH_USER=admin REDFISH_PASS=password make perf-live TARGET=192.168.1.100
```

### With options
```bash
# Run 10 iterations with verbose output
make perf-live TARGET=192.168.1.100 RUNS=10 VERBOSE=1

# With credentials and options
REDFISH_USER=admin REDFISH_PASS=password make perf-live TARGET=192.168.1.100 RUNS=10 VERBOSE=1
```

## Credentials

The tool handles credentials in two ways:

1. **Using config.yml** (default)
   - Looks for `config.yml` in the project root
   - Must contain host credentials in the standard format

2. **Using environment variables**
   - Set `REDFISH_USER` and `REDFISH_PASS`
   - A temporary config file is created and cleaned up automatically
   - Useful for CI/CD or when testing different credentials

## What it measures

- **Scrape Duration**: Total time to complete a scrape
- **Metrics Count**: Number of metrics collected
- **Statistics**: Min/max/average/percentiles across multiple runs
- **Collector Timing**: Time spent in each collector (when available)

## Direct Usage

You can also run the analyzer directly:

```bash
cd tools/performance-analysis
go build -o analyze analyze.go

# Test against a target
./analyze -target 192.168.1.100 -endpoint http://localhost:9610 -runs 5 -verbose

# Output as JSON
./analyze -target 192.168.1.100 -format json
```

## Options

- `-target`: Target to scrape (required)
- `-endpoint`: Redfish exporter endpoint (default: http://localhost:9610)
- `-runs`: Number of test runs (default: 1)
- `-verbose`: Show detailed timing information
- `-format`: Output format - "text" or "json" (default: text)

## Understanding the Results

### Basic Metrics
- **Duration**: Total time for the HTTP request to /redfish endpoint
- **Metrics**: Total number of Prometheus metrics returned

### Statistics (multiple runs)
- **P50**: Median response time
- **P95**: 95th percentile (95% of requests are faster)
- **P99**: 99th percentile (99% of requests are faster)

### Performance Analysis

The tool helps identify whether performance issues are due to:
1. **Network/API latency**: Time spent waiting for Redfish API responses
2. **Processing overhead**: Time spent parsing and creating metrics
3. **Specific collectors**: Which collector is taking the most time

## Example Output

```
=== Performance Analysis Summary ===
Successful scrapes: 5/5
Average duration: 5.645ms
Average metrics: 33
Min duration: 1.291ms
Max duration: 14.296ms
P50 duration: 2.541ms
P95 duration: 14.296ms
P99 duration: 14.296ms
```

## Troubleshooting

If the mock server doesn't start:
```bash
# Manually start mock server
cd tools/mock-server
go build -o mock-server main.go
./mock-server -port 8000 -file testdata/gb300/gb300_host.txt
```

If the exporter isn't running:
```bash
# Manually start exporter
./redfish_exporter --config.file=config.yml
```