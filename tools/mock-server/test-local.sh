#!/bin/bash
# Script to run mock-server and redfish_exporter together for local testing
# Run from the top-level redfish_exporter directory

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Check arguments
if [ $# -eq 0 ]; then
    echo -e "${RED}Usage: $0 <testdata_file>${NC}"
    echo -e "Example: $0 gb300/gb300_host.txt"
    exit 1
fi

# Configuration
TESTDATA_FILE=$1
MOCK_HTTP_PORT=${MOCK_HTTP_PORT:-8080}
MOCK_HTTPS_PORT=${MOCK_HTTPS_PORT:-8443}
EXPORTER_PORT=${EXPORTER_PORT:-9610}
USE_HTTPS=${USE_HTTPS:-true}
MOCK_PID=""
EXPORTER_PID=""

# Cleanup function
cleanup() {
    echo -e "\n${YELLOW}Shutting down services...${NC}"
    if [ ! -z "$MOCK_PID" ]; then
        kill $MOCK_PID 2>/dev/null || true
        echo "Mock server stopped"
    fi
    if [ ! -z "$EXPORTER_PID" ]; then
        kill $EXPORTER_PID 2>/dev/null || true
        echo "Redfish exporter stopped"
    fi
    exit 0
}

# Set up trap for cleanup
trap cleanup INT TERM

# Check if test data file exists
if [ ! -f "tools/mock-server/testdata/${TESTDATA_FILE}" ]; then
    echo -e "${RED}Error: Test data file not found at tools/mock-server/testdata/${TESTDATA_FILE}${NC}"
    echo "Available test data files:"
    find tools/mock-server/testdata -name "*.txt" -type f 2>/dev/null || echo "  None found"
    exit 1
fi

# Build the redfish_exporter if it doesn't exist
if [ ! -f "./redfish_exporter" ]; then
    echo -e "${YELLOW}Building redfish_exporter...${NC}"
    go build
fi

# Create exporter config file
echo -e "${GREEN}Creating exporter config file...${NC}"
if [ "$USE_HTTPS" = "true" ]; then
    TARGET_PORT=$MOCK_HTTPS_PORT
else
    TARGET_PORT=$MOCK_HTTP_PORT
fi

cat > tools/mock-server/exporter-config.yml <<EOF
# Configuration for redfish_exporter to connect to mock server
hosts:
  default:
    username: admin
    password: password
  localhost:${TARGET_PORT}:
    username: admin
    password: password
groups:
  mock_servers:
    username: admin
    password: password
loglevel: info
EOF

# Check if using HTTPS
if [ "$USE_HTTPS" = "true" ]; then
    echo -e "${GREEN}Using HTTPS mode with self-signed certificates${NC}"
    echo -e "${YELLOW}Note: The exporter will connect using HTTPS with InsecureSkipVerify${NC}"
else
    echo -e "${YELLOW}Using HTTP mode${NC}"
    echo -e "${YELLOW}Note: You need to modify collector/redfish_collector.go line 108 to use http:// instead of https://${NC}"
    echo -e "${YELLOW}Change: url := fmt.Sprintf(\"https://%s\", host)${NC}"
    echo -e "${YELLOW}To:     url := fmt.Sprintf(\"http://%s\", host)${NC}"
    read -p "Have you made this change? (y/n) " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        echo -e "${RED}Please make the change and run this script again${NC}"
        exit 1
    fi
fi

# Start mock server
echo -e "\n${GREEN}Starting mock server with ${TESTDATA_FILE}...${NC}"
if [ "$USE_HTTPS" = "true" ]; then
    echo -e "${GREEN}  HTTP on port ${MOCK_HTTP_PORT}, HTTPS on port ${MOCK_HTTPS_PORT}${NC}"
    go run tools/mock-server/main.go \
        -file tools/mock-server/testdata/${TESTDATA_FILE} \
        -auth tools/mock-server/auth.yml \
        -port ${MOCK_HTTP_PORT} \
        -https-port ${MOCK_HTTPS_PORT} \
        -http=true \
        -https=true \
        -debug &
else
    echo -e "${GREEN}  HTTP only on port ${MOCK_HTTP_PORT}${NC}"
    go run tools/mock-server/main.go \
        -file tools/mock-server/testdata/${TESTDATA_FILE} \
        -auth tools/mock-server/auth.yml \
        -port ${MOCK_HTTP_PORT} \
        -http=true \
        -https=false \
        -debug &
fi
MOCK_PID=$!

# Wait for mock server to be ready
echo -e "${YELLOW}Waiting for mock server to be ready...${NC}"
for i in {1..15}; do
    if [ "$USE_HTTPS" = "true" ]; then
        if curl -s -f -k -u admin:password https://localhost:${MOCK_HTTPS_PORT}/redfish/v1 > /dev/null 2>&1; then
            echo -e "${GREEN}Mock server is ready!${NC}"
            break
        fi
    else
        if curl -s -f -u admin:password http://localhost:${MOCK_HTTP_PORT}/redfish/v1 > /dev/null 2>&1; then
            echo -e "${GREEN}Mock server is ready!${NC}"
            break
        fi
    fi
    sleep 1
done

# Start redfish_exporter
echo -e "\n${GREEN}Starting redfish_exporter on port ${EXPORTER_PORT}...${NC}"
./redfish_exporter --config.file=tools/mock-server/exporter-config.yml &
EXPORTER_PID=$!

# Wait for exporter to be ready
echo -e "${YELLOW}Waiting for exporter to be ready...${NC}"
for i in {1..10}; do
    if curl -s -f http://localhost:${EXPORTER_PORT}/metrics > /dev/null 2>&1; then
        echo -e "${GREEN}Exporter is ready!${NC}"
        break
    fi
    sleep 1
done

# Print usage instructions
echo -e "\n${GREEN}=== Services Running ===${NC}"
if [ "$USE_HTTPS" = "true" ]; then
    echo -e "Mock Server HTTP:   http://localhost:${MOCK_HTTP_PORT}"
    echo -e "Mock Server HTTPS:  https://localhost:${MOCK_HTTPS_PORT} (self-signed cert)"
    echo -e "Mock Server Debug:  http://localhost:${MOCK_HTTP_PORT}/debug"
    echo -e "Exporter:           http://localhost:${EXPORTER_PORT}"
    echo -e ""
    echo -e "${GREEN}=== Test Commands ===${NC}"
    echo -e "Test scrape:        ${YELLOW}curl 'http://localhost:${EXPORTER_PORT}/redfish?target=localhost:${MOCK_HTTPS_PORT}'${NC}"
    echo -e "Test with group:    ${YELLOW}curl 'http://localhost:${EXPORTER_PORT}/redfish?target=localhost:${MOCK_HTTPS_PORT}&group=mock_servers'${NC}"
    echo -e "Check mock auth:    ${YELLOW}curl -k -u admin:password https://localhost:${MOCK_HTTPS_PORT}/redfish/v1/Systems${NC}"
    echo -e "View debug info:    ${YELLOW}curl http://localhost:${MOCK_HTTP_PORT}/debug | jq .${NC}"
else
    echo -e "Mock Server:        http://localhost:${MOCK_HTTP_PORT}"
    echo -e "Mock Server Debug:  http://localhost:${MOCK_HTTP_PORT}/debug"
    echo -e "Exporter:           http://localhost:${EXPORTER_PORT}"
    echo -e ""
    echo -e "${GREEN}=== Test Commands ===${NC}"
    echo -e "Test scrape:        ${YELLOW}curl 'http://localhost:${EXPORTER_PORT}/redfish?target=localhost:${MOCK_HTTP_PORT}'${NC}"
    echo -e "Test with group:    ${YELLOW}curl 'http://localhost:${EXPORTER_PORT}/redfish?target=localhost:${MOCK_HTTP_PORT}&group=mock_servers'${NC}"
    echo -e "Check mock auth:    ${YELLOW}curl -u admin:password http://localhost:${MOCK_HTTP_PORT}/redfish/v1/Systems${NC}"
    echo -e "View debug info:    ${YELLOW}curl http://localhost:${MOCK_HTTP_PORT}/debug | jq .${NC}"
fi
echo -e ""
echo -e "${YELLOW}Press Ctrl+C to stop all services${NC}"

# Keep script running
wait
