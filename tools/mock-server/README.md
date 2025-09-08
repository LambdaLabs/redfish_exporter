# Redfish Mock Server

A mock Redfish server for testing the redfish_exporter without needing actual hardware.

## Purpose

This tool allows you to:
- Test the redfish_exporter against captured Redfish data
- Debug issues without accessing production BMCs
- Develop new features using real device responses
- Validate compatibility with new hardware models
- Test both HTTP and HTTPS connections with proper authentication

## Quick Start

### Easy Testing with Make

The simplest way to test with mock data:

```bash
# From the project root directory:

# Specify your test data
make mock-test TESTDATA=my_system/capture.txt

# Or run the script directly:
./tools/mock-server/test-local.sh my_system/capture.txt
```

This will:
- Start the mock server with both HTTP and HTTPS
- Start the redfish_exporter configured to connect to the mock server
- Display test commands you can use
- Clean up everything when you press Ctrl+C

## Detailed Usage

### 1. Capture Redfish Data

Place your captured data files in `tools/mock-server/testdata/<system_name>/`:

```bash
# The capture file format:
# # https://bmc.example.com/redfish/v1
# {JSON response}
# # https://bmc.example.com/redfish/v1/Systems
# {JSON response}
```

### 2. Run the Mock Server

```bash
# Basic usage - loads from testdata directory
go run tools/mock-server/main.go -system gb300

# With specific file
go run tools/mock-server/main.go -file tools/mock-server/testdata/my_system/capture.txt

# With HTTPS support (default ports: 8080 for HTTP, 8443 for HTTPS)
go run tools/mock-server/main.go -file testdata.txt -https=true -http=true

# With custom auth config
go run tools/mock-server/main.go -file testdata.txt -auth auth.yml

# With debug endpoint and response delay
go run tools/mock-server/main.go -file testdata.txt -debug -delay 100ms
```

### 3. Authentication

The mock server supports both Basic Auth and Session-based authentication.

Default credentials (if no auth.yml provided):
- Username: `admin`
- Password: `password`

Custom auth configuration (`auth.yml`):
```yaml
users:
  - username: admin
    password: password
```

### 4. Test with redfish_exporter

The exporter configuration is automatically created by the test script, but you can also create it manually:

```yaml
# tools/mock-server/exporter-config.yml
hosts:
  default:
    username: admin
    password: password
  localhost:8443:  # For HTTPS
    username: admin
    password: password
  localhost:8080:  # For HTTP
    username: admin
    password: password
loglevel: info
```

Then run:
```bash
# Run the exporter
./redfish_exporter --config.file=tools/mock-server/exporter-config.yml

# Test a scrape (HTTPS)
curl "http://localhost:9610/redfish?target=localhost:8443"

# Test a scrape (HTTP)
curl "http://localhost:9610/redfish?target=localhost:8080"
```

## Debug Endpoint

When running with `-debug`, you can access debug information:

```bash
curl http://localhost:8081/debug | jq .
```

This shows:
- Total loaded endpoints
- List of all available endpoints
- Recent request history
- Current configuration

## Capture File Format

The capture file should contain URL comments followed by JSON responses:

```
# https://10.0.0.1/redfish/v1
{
  "@odata.id": "/redfish/v1",
  "@odata.type": "#ServiceRoot.v1_15_0.ServiceRoot",
  ...
}
# https://10.0.0.1/redfish/v1/Systems
{
  "@odata.id": "/redfish/v1/Systems",
  ...
}
```

## Features

- **HTTPS Support**: Runs on both HTTP and HTTPS with self-signed certificates
- **Proper Authentication**:
  - Basic Auth support
  - Session-based authentication with X-Auth-Token
  - Configurable credentials via auth.yml
- **Flexible Data Loading**:
  - Load from testdata directory structure (`-system` flag)
  - Load specific files (`-file` flag)
  - Automatic URL path extraction from capture files
- **Developer Tools**:
  - Debug endpoint for request inspection
  - Request logging and tracking
  - Response delay simulation
  - Trailing slash handling
- **Large file support**: Handles capture files with thousands of endpoints
- **Error resilience**: Continues loading even if some endpoints have invalid JSON

## Command-line Options

```
-file         Path to capture file containing Redfish responses
-system       System name to load from testdata/<system>/
-port         HTTP port (default: 8080)
-https-port   HTTPS port (default: 8443)
-http         Enable HTTP server (default: true)
-https        Enable HTTPS server (default: true)
-auth         Path to auth config file (default: uses admin/password)
-cert         Path to TLS certificate file (auto-generated if not provided)
-key          Path to TLS key file (auto-generated if not provided)
-delay        Response delay to simulate network latency
-debug        Enable debug endpoints
-log          Path to access log file
```

## Limitations

- No support for PATCH/POST operations beyond sessions
- Static responses only (no dynamic behavior)
- Self-signed certificates for HTTPS (certificate warnings expected)
