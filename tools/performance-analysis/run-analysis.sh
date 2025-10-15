#!/bin/bash

# Performance analysis script for redfish_exporter
# Can test against mock servers or real hardware

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
EXPORTER_PORT=${EXPORTER_PORT:-9610}
EXPORTER_URL="http://localhost:${EXPORTER_PORT}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

usage() {
    cat <<EOF
Usage: $0 [OPTIONS] <mode> <target>

Modes:
  mock        Test against mock server
  live        Test against live system

Options:
  -r RUNS     Number of test runs (default: 5)
  -v          Verbose output
  -p PORT     Exporter port (default: 9610)
  -c CONFIG   Config file for exporter (default: config.yml for live, tools/mock-server/exporter-config.yml for mock)
  -h          Show this help

Examples:
  # Test against mock server
  $0 mock localhost:8000

  # Test against live system
  $0 live 192.168.1.100

  # Run 10 iterations with verbose output
  $0 -r 10 -v live 192.168.1.100
EOF
    exit 1
}

# Parse arguments
RUNS=5
VERBOSE=""
CONFIG_FILE=""

while getopts "r:vp:c:h" opt; do
    case $opt in
        r) RUNS="$OPTARG" ;;
        v) VERBOSE="-verbose" ;;
        p) EXPORTER_PORT="$OPTARG" ;;
        c) CONFIG_FILE="$OPTARG" ;;
        h) usage ;;
        *) usage ;;
    esac
done

shift $((OPTIND-1))

if [ $# -lt 2 ]; then
    echo -e "${RED}Error: Mode and target are required${NC}"
    usage
fi

MODE="$1"
TARGET="$2"

# Validate mode
if [ "$MODE" != "mock" ] && [ "$MODE" != "live" ]; then
    echo -e "${RED}Error: Invalid mode '$MODE'. Must be 'mock' or 'live'${NC}"
    usage
fi

# Set config file based on mode if not specified
if [ -z "$CONFIG_FILE" ]; then
    if [ "$MODE" = "mock" ]; then
        CONFIG_FILE="${PROJECT_ROOT}/tools/mock-server/exporter-config.yml"
    else
        CONFIG_FILE="${PROJECT_ROOT}/config.yml"

        # For live mode, check if we need to create/update config
        if [ "$MODE" = "live" ]; then
            # Check if credentials were provided via environment
            if [ ! -z "$REDFISH_USER" ] && [ ! -z "$REDFISH_PASS" ]; then
                echo "Creating temporary config with provided credentials..."
                CONFIG_FILE="${PROJECT_ROOT}/tools/performance-analysis/temp-config.yml"
                cat > "$CONFIG_FILE" <<EOF
hosts:
  default:
    username: ${REDFISH_USER}
    password: ${REDFISH_PASS}
  ${TARGET}:
    username: ${REDFISH_USER}
    password: ${REDFISH_PASS}
loglevel: info
EOF
                TEMP_CONFIG=1
            elif [ ! -f "$CONFIG_FILE" ]; then
                echo -e "${RED}Error: No config.yml found and no credentials provided${NC}"
                echo "Please either:"
                echo "  1. Create a config.yml with credentials"
                echo "  2. Set REDFISH_USER and REDFISH_PASS environment variables"
                echo ""
                echo "Example:"
                echo "  REDFISH_USER=admin REDFISH_PASS=password make perf-live TARGET=192.168.1.100"
                exit 1
            fi
        fi
    fi
fi

echo -e "${GREEN}=== Redfish Exporter Performance Analysis ===${NC}"
echo "Mode: $MODE"
echo "Target: $TARGET"
echo "Runs: $RUNS"
echo "Config: $CONFIG_FILE"
echo ""

# If mock mode, ensure mock server is running FIRST before starting exporter
if [ "$MODE" = "mock" ]; then
    MOCK_PORT=$(echo "$TARGET" | cut -d: -f2)
    if [ -z "$MOCK_PORT" ]; then
        MOCK_PORT="8443"
        TARGET="localhost:8443"
    fi

    if ! curl -s -k "https://localhost:${MOCK_PORT}/redfish/v1/" > /dev/null 2>&1; then
        echo -e "${YELLOW}Warning: Mock server not running on port ${MOCK_PORT}${NC}"
        echo "Starting mock server..."

        # Kill any existing mock server
        pkill -f "mock-server.*${MOCK_PORT}" || true
        sleep 1

        # Build mock server if needed
        if [ ! -f "${PROJECT_ROOT}/tools/mock-server/mock-server" ]; then
            echo "Building mock server..."
            (cd "${PROJECT_ROOT}/tools/mock-server" && go build -o mock-server main.go)
        fi

        # Start mock server with default test data
        TESTDATA="${PROJECT_ROOT}/tools/mock-server/testdata/gb300/gb300_host.txt"
        (cd "${PROJECT_ROOT}/tools/mock-server" && ./mock-server -auth auth.yml -https-port "${MOCK_PORT}" -file "${TESTDATA}" -https=true -http=false) &
        MOCK_PID=$!
        echo "Started mock server with PID ${MOCK_PID}"

        # Wait for it to be ready
        echo -n "Waiting for mock server to be ready..."
        for i in {1..30}; do
            if curl -s -k "https://localhost:${MOCK_PORT}/redfish/v1/" > /dev/null 2>&1; then
                echo " Ready!"
                break
            fi
            echo -n "."
            sleep 1
        done
        echo ""
    fi
fi

# For live mode with temp config, always restart the exporter
# For other modes, only start if not running
if [ "$TEMP_CONFIG" = "1" ] || ! curl -s "${EXPORTER_URL}/metrics" > /dev/null 2>&1; then
    if [ "$TEMP_CONFIG" = "1" ]; then
        echo -e "${YELLOW}Restarting exporter with temporary config...${NC}"
    else
        echo -e "${YELLOW}Warning: Exporter not running on port ${EXPORTER_PORT}${NC}"
        echo "Starting exporter..."
    fi

    # Kill any existing exporter (but not the mock server)
    pkill -f "^${PROJECT_ROOT}/redfish_exporter" || true
    sleep 1

    # Build if needed
    if [ ! -f "${PROJECT_ROOT}/redfish_exporter" ]; then
        echo "Building redfish_exporter..."
        (cd "${PROJECT_ROOT}" && go build)
    fi

    # Start exporter in background
    "${PROJECT_ROOT}/redfish_exporter" --config.file="$CONFIG_FILE" --web.listen-address=":${EXPORTER_PORT}" &
    EXPORTER_PID=$!
    echo "Started exporter with PID ${EXPORTER_PID}"

    # Wait for it to be ready
    echo -n "Waiting for exporter to be ready..."
    for i in {1..30}; do
        if curl -s "${EXPORTER_URL}/metrics" > /dev/null 2>&1; then
            echo " Ready!"
            break
        fi
        echo -n "."
        sleep 1
    done
    echo ""
fi

# Build the analyzer if needed
echo "Building performance analyzer..."
(cd "${SCRIPT_DIR}" && go build -o analyze analyze.go)

# Run the analysis
echo -e "\n${GREEN}Running performance analysis...${NC}\n"
"${SCRIPT_DIR}/analyze" \
    -target "$TARGET" \
    -endpoint "${EXPORTER_URL}" \
    -runs "$RUNS" \
    $VERBOSE

# Cleanup if we started services
if [ ! -z "$EXPORTER_PID" ]; then
    echo -e "\n${YELLOW}Stopping exporter (PID ${EXPORTER_PID})${NC}"
    kill "$EXPORTER_PID" 2>/dev/null || true
fi

if [ ! -z "$MOCK_PID" ]; then
    echo -e "${YELLOW}Stopping mock server (PID ${MOCK_PID})${NC}"
    kill "$MOCK_PID" 2>/dev/null || true
fi

# Clean up temporary config file
if [ "$TEMP_CONFIG" = "1" ] && [ -f "${PROJECT_ROOT}/tools/performance-analysis/temp-config.yml" ]; then
    echo -e "${YELLOW}Removing temporary config file${NC}"
    rm -f "${PROJECT_ROOT}/tools/performance-analysis/temp-config.yml"
fi

echo -e "\n${GREEN}Analysis complete!${NC}"
