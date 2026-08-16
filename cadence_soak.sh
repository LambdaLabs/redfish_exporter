#!/bin/bash
# Cadence soak for OBS-951 Dell B300 NPI: N cycles (default 12), start-to-start 300s,
# mirroring the production scrape cadence. Requires the exporter running on :9610.
# Usage: bash cadence_soak.sh [cycles]
# Any failed cycle (bad HTTP code or incomplete series counts): kills the exporter, exits 1.
N=${1:-12}
RESULTS=${RESULTS:-/tmp/cadence_soak_results.txt}
echo "=== soak start $(date +%T) cycles=$N ===" >> "$RESULTS"
for i in $(seq 1 "$N"); do
  T0=$(date +%s)
  CODE=$(curl -s -m 400 "http://localhost:9610/redfish?target=10.254.32.111&group=dell_b300&module=chassis&module=gpu&module=system&module=rf_version" -o /tmp/soak-$i.txt -w '%{http_code}')
  T1=$(date +%s); DUR=$((T1-T0))
  NV=$(grep -c '^redfish_gpu_nvlink_' /tmp/soak-$i.txt || true)
  TE=$(grep -c '^redfish_telemetry_' /tmp/soak-$i.txt || true)
  SY=$(grep -c '^redfish_system_' /tmp/soak-$i.txt || true)
  CH=$(grep -c '^redfish_chassis_' /tmp/soak-$i.txt || true)
  echo "cycle=$i start=$(date -r $T0 +%T) dur=${DUR}s code=$CODE nvlink=$NV telemetry=$TE system=$SY chassis=$CH" >> "$RESULTS"
  if [ "$CODE" != "200" ] || [ "$NV" != "1296" ] || [ "$TE" != "448" ] || [ "$SY" -lt 400 ] || [ "$CH" -lt 400 ]; then
    echo "FAIL at cycle $i" >> "$RESULTS"
    kill $(lsof -tnP -iTCP:9610 -sTCP:LISTEN) 2>/dev/null
    exit 1
  fi
  if [ "$i" -lt "$N" ]; then
    ELAPSED=$(( $(date +%s) - T0 ))
    SLEEP=$(( 300 - ELAPSED )); [ "$SLEEP" -gt 0 ] && sleep "$SLEEP"
  fi
done
echo "SUCCESS: $N/$N cycles clean" >> "$RESULTS"
