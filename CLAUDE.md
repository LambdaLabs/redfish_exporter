# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Overview
This is a Prometheus exporter for Redfish-based hardware servers. It collects metrics from server hardware using the Redfish API standard and exposes them in Prometheus format.

## Key Architecture

### Core Components
- **Main Application** (`main.go`): HTTP server on port 9610 with endpoints for metrics scraping and configuration reload
- **Collectors** (`collector/`): Implements Prometheus collectors for different Redfish components:
  - `chassis_collector.go`: Collects chassis-level metrics (power, thermal)
  - `system_collector.go`: Collects system-level metrics (health, state)
  - `manager_collector.go`: Collects BMC manager metrics
  - `common_collector.go`: Shared collector functionality
- **Configuration** (`config.go`): YAML-based configuration for host credentials and groups

### Key Dependencies
- `github.com/stmcginnis/gofish`: Redfish API client library
- `github.com/prometheus/client_golang`: Prometheus metrics library
- `github.com/prometheus/exporter-toolkit`: Prometheus exporter utilities

## Development Commands

### Build
```bash
go build
```

### Run Tests
```bash
go test -v ./...
```

### Run the Exporter
```bash
# Run with default config file (config.yml)
./redfish_exporter

# Run with custom config
./redfish_exporter --config.file=path/to/config.yml
```

### Lint (requires golangci-lint)
```bash
golangci-lint run
```

## Configuration
The exporter uses a YAML configuration file for host credentials:
- Default location: `config.yml`
- Example configuration provided in `config.example.yml`
- Supports individual host configs and group configs
- Configuration can be reloaded via `/-/reload` endpoint or SIGHUP signal

## API Endpoints
- `/redfish?target=<ip>` - Scrape metrics from a specific target
- `/redfish?target=<ip>&group=<name>` - Scrape using group credentials
- `/metrics` - Local exporter metrics
- `/-/reload` - Reload configuration (POST/PUT)
- `/` - Web UI for testing scrapes