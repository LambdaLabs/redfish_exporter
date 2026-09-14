#!/usr/bin/env bash
# Serve the redfish_exporter locally so you can point it at live Redfish systems by hand.
#
# Builds the exporter, starts it with its logs streaming to your terminal, prints the URLs
# to scrape, and stays up until Ctrl-C. It makes no assertions and never scrapes for you —
# where validate-live answers "is this target healthy", this answers "let me drive the
# exporter against a real BMC and watch what it does".
#
# Run from the project root, or via `make serve-live`.
#
# Usage:
#   REDFISH_USER=admin REDFISH_PASS=secret TARGET=10.0.0.100 ./tools/serve-live/serve.sh
#   CONFIG_FILE=my-config.yml ./tools/serve-live/serve.sh
#
# Environment:
#   TARGET         (optional) BMC host or IP, used only to print a ready-to-paste scrape URL.
#                  The exporter serves any target it has credentials for; this is a
#                  convenience, not a restriction.
#   MODULES        (optional) comma-separated module names for the printed URL, e.g.
#                  "powershelf,chassis". Default: the exporter's built-in bundle
#                  (gpu,chassis,manager,system,telemetry). NOTE: powershelf is NOT in it.
#   REDFISH_USER   (optional) Redfish username. If set with REDFISH_PASS, a temp config is
#                  generated defining every known module.
#   REDFISH_PASS   (optional) Redfish password.
#   CONFIG_FILE    (optional) Path to an existing exporter config (default: config.yml).
#                  Used when REDFISH_USER/REDFISH_PASS are not provided.
#   EXPORTER_PORT  (optional) Port for the local exporter (default: 9614).
#   LOGLEVEL       (optional) debug, info, warn or error. Unset leaves the config's own
#                  setting alone. `debug` is what surfaces the per-request session logging.

set -euo pipefail

RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'; BLUE='\033[0;34m'; NC='\033[0m'

TARGET="${TARGET:-}"
MODULES="${MODULES:-}"
REDFISH_USER="${REDFISH_USER:-}"
REDFISH_PASS="${REDFISH_PASS:-}"
CONFIG_FILE="${CONFIG_FILE:-config.yml}"
EXPORTER_PORT="${EXPORTER_PORT:-9614}"

# Treat an empty LOGLEVEL as unset. `make serve-live` always exports it, and an empty value
# would otherwise reach viper's AutomaticEnv and blank out whatever level the config set.
if [ -z "${LOGLEVEL:-}" ]; then unset LOGLEVEL; fi

EXPORTER_PID=""
TMP_DIR=""

cleanup() {
  if [ -n "$EXPORTER_PID" ]; then
    # SIGTERM, then SIGKILL — the exporter does a 60s graceful shutdown that keeps the
    # listen socket bound, which would block the port on a quick re-run.
    kill "$EXPORTER_PID" 2>/dev/null || true
    for _ in 1 2 3; do kill -0 "$EXPORTER_PID" 2>/dev/null || break; sleep 0.3; done
    kill -9 "$EXPORTER_PID" 2>/dev/null || true
  fi
  # The generated config holds credentials in cleartext, so it does not outlive the run.
  [ -n "$TMP_DIR" ] && rm -rf "$TMP_DIR" || true
}
trap cleanup EXIT INT TERM

# --- Resolve exporter config -------------------------------------------------
# If credentials are provided, generate a temp config defining all known modules (so
# non-default modules like powershelf can be requested). Otherwise use CONFIG_FILE.
if [ -n "$REDFISH_USER" ] && [ -n "$REDFISH_PASS" ]; then
  # Use a temp dir with a fixed .yml filename: the exporter infers config type from the
  # file extension, and `mktemp -t ...yml` does not preserve the extension on macOS.
  TMP_DIR="$(mktemp -d)"
  USE_CONFIG="$TMP_DIR/config.yml"
  cat > "$USE_CONFIG" <<EOF
hosts:
  default:
    username: '${REDFISH_USER_YAML}'
    password: '${REDFISH_PASS_YAML}'
loglevel: ${LOGLEVEL:-info}
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
  CONFIG_DESC="generated, user '${REDFISH_USER}' for all targets"
else
  if [ ! -f "$CONFIG_FILE" ]; then
    echo -e "${RED}Error: no credentials given and config file '$CONFIG_FILE' not found${NC}"
    echo "Provide REDFISH_USER and REDFISH_PASS, or set CONFIG_FILE to a valid exporter config."
    exit 2
  fi
  USE_CONFIG="$CONFIG_FILE"
  CONFIG_DESC="$CONFIG_FILE"
fi

# --- Build & start the exporter ----------------------------------------------
echo -e "${YELLOW}Building exporter...${NC}"
go build -o redfish_exporter ./cmd/redfish-exporter

# Fail fast with a clear message if the port is already taken (e.g. a prior exporter still
# in its graceful-shutdown window).
if lsof -nP -iTCP:"${EXPORTER_PORT}" -sTCP:LISTEN >/dev/null 2>&1; then
  echo -e "${RED}Port :${EXPORTER_PORT} is already in use:${NC}"
  lsof -nP -iTCP:"${EXPORTER_PORT}" -sTCP:LISTEN 2>/dev/null || true
  echo -e "${YELLOW}Set a different port with EXPORTER_PORT=<n>, or wait for the holder to exit.${NC}"
  exit 1
fi

# --- Build the scrape URL to advertise ---------------------------------------
SCRAPE_URL="http://localhost:${EXPORTER_PORT}/redfish?target=${TARGET:-<ip-or-host>}"
MODULE_DESC="default bundle (gpu,chassis,manager,system,telemetry)"
if [ -n "$MODULES" ]; then
  MODULE_DESC="$MODULES"
  IFS=',' read -r -a _mods <<< "$MODULES"
  for m in "${_mods[@]}"; do
    m="$(echo "$m" | tr -d '[:space:]')"
    [ -n "$m" ] && SCRAPE_URL="${SCRAPE_URL}&module=${m}"
  done
fi

echo -e "${YELLOW}Starting exporter on :${EXPORTER_PORT}...${NC}"
# IMPORTANT: scrub MODULES from the exporter's environment. The exporter uses viper
# AutomaticEnv (no prefix), which maps the env var MODULES onto the config key `modules`,
# overwriting the file's module map with a scalar so it decodes to an empty map (→ no
# collectors). This script sets MODULES for its own URL-building, so it must not leak in.
#
# LOGLEVEL is deliberately NOT scrubbed: the exporter has no log-level flag, and the same
# AutomaticEnv mapping is what lets it override the level in a CONFIG_FILE we don't own.
env -u MODULES ./redfish_exporter \
  --config.file="$USE_CONFIG" \
  --web.listen-address=":${EXPORTER_PORT}" &
EXPORTER_PID=$!

ready=false
for _ in $(seq 1 15); do
  # If the exporter exited (e.g. bind failure, bad config), stop waiting.
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
  exit 1
fi

echo
echo -e "${GREEN}Exporter is up.${NC}  config: ${CONFIG_DESC}  |  log level: ${LOGLEVEL:-from config}"
echo
echo -e "${BLUE}Scrape a live target:${NC}"
echo "  curl -s '${SCRAPE_URL}'"
if [ -z "$TARGET" ]; then
  echo -e "  ${YELLOW}(set TARGET=<ip> to have the exact URL printed here)${NC}"
fi
echo
echo -e "${BLUE}Exporter's own metrics${NC} (session teardown counters live here, not on /redfish):"
echo "  curl -s http://localhost:${EXPORTER_PORT}/metrics | grep redfish_exporter_"
echo
echo -e "  modules: ${MODULE_DESC}"
echo -e "${YELLOW}Logs follow. Ctrl-C to stop.${NC}"
echo

# Hand the terminal to the exporter until it exits or the user interrupts. `wait` returns
# non-zero when the process is signalled, which is the ordinary way this script ends.
wait "$EXPORTER_PID" || true
