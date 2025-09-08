# Redfish Capture Tool

A tool to crawl and capture all Redfish endpoints from a BMC for use with the mock server.

## Purpose

This tool crawls a real Redfish-enabled BMC and saves all responses in a format that can be used by the mock server for testing. It performs a depth-first search following all `@odata.id` links to discover and capture the complete Redfish tree.

## Installation

```bash
cd tools/capture
go build -o capture main.go
```

## Usage

### Basic Usage

```bash
./capture -host <bmc-ip> -user <username> -pass <password> -output <system_name>
```

### Examples

```bash
# Capture from a host you want to save as GB300
./capture -host 10.0.0.100 -user admin -pass mypassword -output gb300

# Capture only first 100 endpoints (useful for testing)
./capture -host 10.0.0.100 -user admin -pass secret -output test_system -max 100

# Skip additional paths besides Actions
./capture -host 10.0.0.100 -user admin -pass secret -output clean_system \
  -skipPaths "Actions,Logs,Events"
```

## Command-line Options

| Flag | Description | Default |
|------|-------------|---------|
| `-host` | BMC host or IP address (required) | - |
| `-user` | Username for authentication (required) | - |
| `-pass` | Password for authentication (required) | - |
| `-output` | Output directory name under testdata/ (required) | - |
| `-insecure` | Skip TLS certificate verification | true |
| `-timeout` | Per-request timeout | 10s |
| `-sleep` | Sleep between requests in milliseconds | 0 |
| `-max` | Maximum number of endpoints to capture (0 = unlimited) | 0 |
| `-skipPaths` | Comma-separated list of path substrings to skip | Actions |

## Output

The tool creates a capture file at:
```
tools/mock-server/testdata/<output>/capture.txt
```

The format is compatible with the mock server:
```
# https://10.0.0.100/redfish/v1
{
  "@odata.id": "/redfish/v1",
  "@odata.type": "#ServiceRoot.v1_15_0.ServiceRoot",
  ...
}
# https://10.0.0.100/redfish/v1/Systems
{
  "@odata.id": "/redfish/v1/Systems",
  ...
}
```

## How It Works

1. **Authentication**: Uses HTTP Basic Authentication to connect to the BMC
2. **Discovery**: Starts from `/redfish/v1` and recursively discovers all endpoints
3. **Crawling**: Uses depth-first search to explore the Redfish tree
4. **Link Extraction**: Finds all `@odata.id` references in JSON responses
5. **Filtering**:
   - Skips configurable paths (default: `/Actions/` endpoints which are POST-only)
   - Stays on the same host
   - Avoids duplicate visits

## Testing with Mock Server

After capturing data, you can test it with the mock server:

```bash
# 1. Capture data from a real BMC
./capture -host 10.0.0.100 -user admin -pass password -output my_system

# 2. Run mock server with captured data
cd ../..  # Back to project root
make mock-test TESTDATA=my_system/capture.txt

# 3. The mock server will serve the captured data
# and the exporter will be configured to scrape it
```

## Troubleshooting

### Connection Refused
- Verify the BMC IP and port are correct
- Check if the BMC requires HTTPS (default) or HTTP
- Ensure the BMC is accessible from your network

### Authentication Failed
- Verify username and password
- Some BMCs have default credentials (check documentation)
- Some BMCs lock accounts after failed attempts

### TLS Certificate Errors
- The `-insecure` flag is enabled by default
- For production use, consider using proper certificates

### Incomplete Capture
- Increase timeout with `-timeout 30s` for slow BMCs
- Add sleep between requests with `-sleep 100` if BMC has rate limiting
- Check stderr output for specific endpoint errors

## Notes

- The tool respects the BMC's response time - slower BMCs will take longer to crawl
- Large systems may have thousands of endpoints and take several minutes to capture
- The capture file can be edited manually if needed to remove sensitive data
- Captured data includes the full URL in comments for debugging
