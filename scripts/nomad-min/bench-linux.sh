#!/usr/bin/env bash
# Copyright IBM Corp. 2015, 2026
# SPDX-License-Identifier: BUSL-1.1

set -euo pipefail

PID=""
UNIT="nomad"
WARMUP=15
DURATION=45
INTERVAL=1

usage() {
  cat <<'EOF'
Usage: scripts/nomad-min/bench-linux.sh [options]

Options:
  --pid PID          Process ID to sample instead of resolving a systemd unit.
  --unit UNIT        systemd unit to resolve (default: nomad).
  --warmup SECONDS   Warmup before sampling (default: 15).
  --duration SECONDS Sampling duration (default: 45).
  --interval SECONDS Sampling interval (default: 1).
EOF
}

while (($#)); do
  case "$1" in
    --pid) PID="${2:?missing pid}"; shift 2 ;;
    --unit) UNIT="${2:?missing unit}"; shift 2 ;;
    --warmup) WARMUP="${2:?missing warmup}"; shift 2 ;;
    --duration) DURATION="${2:?missing duration}"; shift 2 ;;
    --interval) INTERVAL="${2:?missing interval}"; shift 2 ;;
    -h|--help) usage; exit 0 ;;
    *) echo "unknown argument: $1" >&2; exit 2 ;;
  esac
done

[[ "$(uname -s)" == "Linux" ]] || {
  echo "bench-linux.sh requires Linux /proc" >&2
  exit 1
}

if [[ -z "${PID}" ]]; then
  PID="$(systemctl show -p MainPID --value "${UNIT}")"
fi
[[ -r "/proc/${PID}/status" && "${PID}" != 0 ]] || {
  echo "process is not available: ${PID}" >&2
  exit 1
}

sleep "${WARMUP}"

ticks() { awk '{print $14 + $15}' "/proc/${PID}/stat"; }
field_kb() { awk -v key="$1" '$1 == key":" {print $2}' "/proc/${PID}/status"; }
pss_kb() { awk '$1 == "Pss:" {print $2}' "/proc/${PID}/smaps_rollup"; }

clk_tck="$(getconf CLK_TCK)"
start_ticks="$(ticks)"
start_ns="$(date +%s%N)"
samples=0
rss_sum=0
pss_sum=0
rss_max=0
pss_max=0

end_at=$(( $(date +%s) + DURATION ))
while (( $(date +%s) < end_at )); do
  rss="$(field_kb VmRSS)"
  pss="$(pss_kb)"
  rss_sum=$((rss_sum + rss))
  pss_sum=$((pss_sum + pss))
  ((rss > rss_max)) && rss_max="${rss}"
  ((pss > pss_max)) && pss_max="${pss}"
  samples=$((samples + 1))
  sleep "${INTERVAL}"
done

end_ticks="$(ticks)"
end_ns="$(date +%s%N)"
cpu_pct="$(awk -v dticks="$((end_ticks - start_ticks))" -v hz="${clk_tck}" \
  -v dns="$((end_ns - start_ns))" 'BEGIN {printf "%.3f", (dticks / hz) / (dns / 1000000000) * 100}')"

rss_avg=$((rss_sum / samples))
pss_avg=$((pss_sum / samples))
threads="$(awk '$1 == "Threads:" {print $2}' "/proc/${PID}/status")"
hwm="$(field_kb VmHWM)"

printf '{"pid":%s,"samples":%s,"warmup_seconds":%s,"duration_seconds":%s,' \
  "${PID}" "${samples}" "${WARMUP}" "${DURATION}"
printf '"cpu_percent":%s,"rss_avg_kb":%s,"rss_max_kb":%s,' \
  "${cpu_pct}" "${rss_avg}" "${rss_max}"
printf '"pss_avg_kb":%s,"pss_max_kb":%s,"vm_hwm_kb":%s,"threads":%s}\n' \
  "${pss_avg}" "${pss_max}" "${hwm}" "${threads}"
