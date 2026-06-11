#!/usr/bin/env bash
# Validate the redfish_exporter against a live Redfish system.
#
# Starts the exporter, scrapes /redfish?target=<host> for the requested modules,
# and asserts the scrape produced healthy metrics:
#   - HTTP 200 from the scrape endpoint
#   - redfish_up == 1
#   - redfish_exporter_collectors_failed == 0
#   - redfish_exporter_collectors_succeeded >= 1
#   - at least one <collector>_collector_scrape_status == 1
#
# Run from the project root, or via `make validate-live`.
#
# Usage:
#   TARGET=10.0.0.100 ./tools/validate-live/validate.sh
#   REDFISH_USER=admin REDFISH_PASS=secret TARGET=10.0.0.100 MODULES=powershelf ./tools/validate-live/validate.sh
#
# Environment:
#   TARGET         (required) BMC host or IP, optionally host:port
#   MODULES        (optional) comma-separated module names to request, e.g. "powershelf,chassis".
#                  Defaults to the exporter's built-in bundle (gpu,chassis,manager,system,telemetry).
#                  NOTE: powershelf is NOT in the default bundle — request it explicitly.
#   REDFISH_USER   (optional) Redfish username. If set with REDFISH_PASS, a temp config is generated.
#   REDFISH_PASS   (optional) Redfish password.
#   CONFIG_FILE    (optional) Path to an existing exporter config (default: config.yml). Used when
#                  REDFISH_USER/REDFISH_PASS are not provided.
#   EXPORTER_PORT  (optional) Port for the local exporter (default: 9613).

set -euo pipefail

RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'; BLUE='\033[0;34m'; NC='\033[0m'

TARGET="${TARGET:-}"
MODULES="${MODULES:-}"
REDFISH_USER="${REDFISH_USER:-}"
REDFISH_PASS="${REDFISH_PASS:-}"
CONFIG_FILE="${CONFIG_FILE:-config.yml}"
EXPORTER_PORT="${EXPORTER_PORT:-9613}"

if [ -z "$TARGET" ]; then
  echo -e "${RED}Error: TARGET is required${NC}"
  echo "Usage: TARGET=<ip-or-host[:port]> $0"
  echo "       REDFISH_USER=admin REDFISH_PASS=secret TARGET=10.0.0.1 MODULES=powershelf $0"
  exit 2
fi

EXPORTER_PID=""
TMP_DIR=""
TMP_CONFIG=""
SCRAPE_BODY="$(mktemp)"
EXPORTER_LOG="$(mktemp)"

cleanup() {
  if [ -n "$EXPORTER_PID" ]; then
    # SIGTERM, then SIGKILL — the exporter does a 60s graceful shutdown that keeps
    # the listen socket bound, which would block the port on a quick re-run.
    kill "$EXPORTER_PID" 2>/dev/null || true
    for _ in 1 2 3; do kill -0 "$EXPORTER_PID" 2>/dev/null || break; sleep 0.3; done
    kill -9 "$EXPORTER_PID" 2>/dev/null || true
  fi
  [ -n "$TMP_DIR" ] && rm -rf "$TMP_DIR" || true
  rm -f "$SCRAPE_BODY" "$EXPORTER_LOG" || true
}
trap cleanup EXIT INT TERM

# --- Resolve exporter config -------------------------------------------------
# If credentials are provided, generate a temp config defining all known modules
# (so non-default modules like powershelf can be requested). Otherwise use CONFIG_FILE.
if [ -n "$REDFISH_USER" ] && [ -n "$REDFISH_PASS" ]; then
  # Use a temp dir with a fixed .yml filename: the exporter infers config type from the
  # file extension, and `mktemp -t ...yml` does not preserve the extension on macOS.
  TMP_DIR="$(mktemp -d)"
  TMP_CONFIG="$TMP_DIR/config.yml"
  cat > "$TMP_CONFIG" <<EOF
hosts:
  default:
    username: ${REDFISH_USER}
    password: ${REDFISH_PASS}
loglevel: info
modules:
  chassis:
    prober: chassis_collector
  gpu:
    prober: gpu_collector
  manager:
    prober: manager_collector
  system:
    prober: system_collector
  telemetry:
    prober: telemetry_collector
  powershelf:
    prober: powershelf_collector
EOF
  USE_CONFIG="$TMP_CONFIG"
  echo -e "${BLUE}Using generated config with credentials from REDFISH_USER/REDFISH_PASS${NC}"
else
  if [ ! -f "$CONFIG_FILE" ]; then
    echo -e "${RED}Error: no credentials given and config file '$CONFIG_FILE' not found${NC}"
    echo "Provide REDFISH_USER and REDFISH_PASS, or set CONFIG_FILE to a valid exporter config."
    exit 2
  fi
  USE_CONFIG="$CONFIG_FILE"
  echo -e "${BLUE}Using existing config: ${USE_CONFIG}${NC}"
fi

# --- Build & start the exporter ----------------------------------------------
echo -e "${YELLOW}Building exporter...${NC}"
go build -o redfish_exporter ./cmd/redfish-exporter

# Fail fast with a clear message if the port is already taken (e.g. a prior exporter
# still in its graceful-shutdown window).
if lsof -nP -iTCP:"${EXPORTER_PORT}" -sTCP:LISTEN >/dev/null 2>&1; then
  echo -e "${RED}Port :${EXPORTER_PORT} is already in use:${NC}"
  lsof -nP -iTCP:"${EXPORTER_PORT}" -sTCP:LISTEN 2>/dev/null || true
  echo -e "${YELLOW}Set a different port with EXPORTER_PORT=<n>, or wait for the holder to exit.${NC}"
  exit 1
fi

echo -e "${YELLOW}Starting exporter on :${EXPORTER_PORT}...${NC}"
# IMPORTANT: scrub MODULES from the exporter's environment. The exporter uses viper
# AutomaticEnv (no prefix), which maps the env var MODULES onto the config key `modules`,
# overwriting the file's module map with a scalar so it decodes to an empty map (→ no
# collectors). This script sets MODULES for its own URL-building, so it must not leak in.
env -u MODULES ./redfish_exporter --config.file="$USE_CONFIG" --web.listen-address=":${EXPORTER_PORT}" >"$EXPORTER_LOG" 2>&1 &
EXPORTER_PID=$!

ready=false
for _ in $(seq 1 15); do
  # If the exporter exited (e.g. bind failure, bad config), stop waiting and show why.
  if ! kill -0 "$EXPORTER_PID" 2>/dev/null; then
    break
  fi
  if curl -s -f "http://localhost:${EXPORTER_PORT}/metrics" >/dev/null 2>&1; then
    ready=true; break
  fi
  sleep 1
done
if [ "$ready" != true ]; then
  echo -e "${RED}Exporter did not become ready on :${EXPORTER_PORT}${NC}"
  echo -e "${YELLOW}--- exporter log ---${NC}"
  tail -n 20 "$EXPORTER_LOG" 2>/dev/null || echo "  (no log captured)"
  exit 1
fi

# --- Build scrape URL --------------------------------------------------------
SCRAPE_URL="http://localhost:${EXPORTER_PORT}/redfish?target=${TARGET}"
MODULE_DESC="default bundle (gpu,chassis,manager,system,telemetry)"
if [ -n "$MODULES" ]; then
  MODULE_DESC="$MODULES"
  IFS=',' read -r -a _mods <<< "$MODULES"
  for m in "${_mods[@]}"; do
    m="$(echo "$m" | tr -d '[:space:]')"
    [ -n "$m" ] && SCRAPE_URL="${SCRAPE_URL}&module=${m}"
  done
fi

if [ -n "${DEBUG:-}" ]; then
  echo -e "${YELLOW}--- generated/used config (${USE_CONFIG}) ---${NC}"; cat "$USE_CONFIG"
  echo -e "${YELLOW}--- exporter log ---${NC}"; cat "$EXPORTER_LOG"
  echo -e "${YELLOW}--- scrape URL ---${NC}"; echo "$SCRAPE_URL"
fi

echo -e "${YELLOW}Scraping ${TARGET} (modules: ${MODULE_DESC})...${NC}"
HTTP_CODE="$(curl -s -o "$SCRAPE_BODY" -w '%{http_code}' "$SCRAPE_URL" || echo "000")"

if [ -n "${DEBUG:-}" ]; then
  echo -e "${YELLOW}--- raw scrape body (${HTTP_CODE}) ---${NC}"; cat "$SCRAPE_BODY"; echo -e "${YELLOW}--- end body ---${NC}"
fi

# --- Helpers to read Prometheus text -----------------------------------------
# awk is used throughout (not grep) because grep returns non-zero on no-match, which
# would abort the script under `set -euo pipefail`. awk always exits 0.
metric_value() { awk -v n="$1" '$1==n {print $2; exit}' "$SCRAPE_BODY"; }

# --- Validate ----------------------------------------------------------------
FAILS=0
check() { # check <ok:0/1> <message>
  if [ "$1" -eq 1 ]; then
    echo -e "  ${GREEN}PASS${NC}  $2"
  else
    echo -e "  ${RED}FAIL${NC}  $2"
    FAILS=$((FAILS + 1))
  fi
}

echo -e "\n${BLUE}=== Validation ===${NC}"

# 1) HTTP status
[ "$HTTP_CODE" = "200" ] && check 1 "scrape returned HTTP 200" || check 0 "scrape returned HTTP ${HTTP_CODE} (expected 200)"

UP="$(metric_value redfish_up)"
SUCCEEDED="$(metric_value redfish_exporter_collectors_succeeded)"
FAILED="$(metric_value redfish_exporter_collectors_failed)"
SERIES_TOTAL="$(awk '!/^#/ && NF {c++} END {print c+0}' "$SCRAPE_BODY")"
FAMILIES="$(awk '/^# HELP/ {c++} END {print c+0}' "$SCRAPE_BODY")"

# 2) redfish_up == 1
[ "${UP:-}" = "1" ] && check 1 "redfish_up == 1" || check 0 "redfish_up == ${UP:-<missing>} (expected 1)"

# 3) no failed collectors
[ "${FAILED:-1}" = "0" ] && check 1 "redfish_exporter_collectors_failed == 0" || check 0 "redfish_exporter_collectors_failed == ${FAILED:-<missing>} (expected 0)"

# 4) at least one collector succeeded
if [ -n "${SUCCEEDED:-}" ] && [ "${SUCCEEDED%.*}" -ge 1 ] 2>/dev/null; then
  check 1 "redfish_exporter_collectors_succeeded == ${SUCCEEDED} (>= 1)"
else
  check 0 "redfish_exporter_collectors_succeeded == ${SUCCEEDED:-<missing>} (expected >= 1)"
fi

# 5) at least one collector reported scrape_status 1
SUCCEEDING_COLLECTORS="$(awk '/collector_scrape_status/ && !/^#/ && $NF=="1" {c++} END {print c+0}' "$SCRAPE_BODY")"
[ "${SUCCEEDING_COLLECTORS:-0}" -ge 1 ] && check 1 "at least one collector_scrape_status == 1 (${SUCCEEDING_COLLECTORS})" || check 0 "no collector reported scrape_status == 1"

# --- Report ------------------------------------------------------------------
echo -e "\n${BLUE}=== Per-collector scrape status ===${NC}"
STATUS_LINES="$(awk '/collector_scrape_status/ && !/^#/' "$SCRAPE_BODY")"
if [ -n "$STATUS_LINES" ]; then
  printf '%s\n' "$STATUS_LINES" | while IFS= read -r line; do
    status="${line##* }"
    if [ "$status" = "1" ]; then echo -e "  ${GREEN}up${NC}    ${line}"; else echo -e "  ${RED}down${NC}  ${line}"; fi
  done
else
  echo "  (none emitted)"
fi

echo -e "\n${BLUE}=== Summary ===${NC}"
echo "  target:            ${TARGET}"
echo "  modules requested: ${MODULE_DESC}"
echo "  metric families:   ${FAMILIES}"
echo "  metric series:     ${SERIES_TOTAL}"

if [ "$FAILS" -eq 0 ]; then
  echo -e "\n${GREEN}VALIDATION PASSED${NC}"
  exit 0
else
  echo -e "\n${RED}VALIDATION FAILED (${FAILS} check(s))${NC}"
  echo -e "${YELLOW}Hint: re-run the scrape manually to inspect output:${NC}"
  echo "  curl '${SCRAPE_URL}'"
  exit 1
fi
