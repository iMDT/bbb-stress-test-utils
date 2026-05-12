#!/usr/bin/env bash
set -euo pipefail

# ---------------------------------------------------------------------------
# run-parallel.sh — launch N concurrent stress-test instances, one per meeting
#
# Usage:
#   ./run-parallel.sh <numberOfMeetings> [options]
#
# Script options (consumed here, not forwarded to bbb-stress-test):
#   --timeout=SECONDS     Kill each bbb-stress-test instance after N seconds
#   --interval=SECONDS    Wait N seconds between launching each meeting
#
# All other flags are forwarded to bbb-stress-test.
#
# Examples:
#   # 10 meetings, all at once
#   ./run-parallel.sh 10 --serverHost=bbb30.bbb.imdt.dev --numOfUsers=200
#
#   # 1000 meetings ao longo do dia: nova a cada 60s, cada uma vive 300s (~5 ativas ao mesmo tempo)
#   ./run-parallel.sh 1000 \
#     --interval=60 \
#     --timeout=300 \
#     --serverHost=bbb30.bbb.imdt.dev \
#     --numOfUsers=150
# ---------------------------------------------------------------------------

if [[ $# -lt 1 ]]; then
  echo "Usage: $0 <numberOfMeetings> [--timeout=SECONDS] [--interval=SECONDS] [--serverHost=... ...]"
  exit 1
fi

NUMBER_OF_MEETINGS="$1"
shift

MEETING_TIMEOUT=""   # seconds; empty = no timeout
LAUNCH_INTERVAL=""   # seconds between launches; empty = no delay
EXTRA_ARGS=()

for arg in "$@"; do
  case "$arg" in
    --timeout=*)
      MEETING_TIMEOUT="${arg#--timeout=}"
      ;;
    --interval=*)
      LAUNCH_INTERVAL="${arg#--interval=}"
      ;;
    *)
      EXTRA_ARGS+=("$arg")
      ;;
  esac
done

if ! [[ "$NUMBER_OF_MEETINGS" =~ ^[1-9][0-9]*$ ]]; then
  echo "Error: numberOfMeetings must be a positive integer, got: $NUMBER_OF_MEETINGS"
  exit 1
fi
if [[ -n "$MEETING_TIMEOUT" ]] && ! [[ "$MEETING_TIMEOUT" =~ ^[1-9][0-9]*$ ]]; then
  echo "Error: --timeout must be a positive integer (seconds), got: $MEETING_TIMEOUT"
  exit 1
fi
if [[ -n "$LAUNCH_INTERVAL" ]] && ! [[ "$LAUNCH_INTERVAL" =~ ^[1-9][0-9]*$ ]]; then
  echo "Error: --interval must be a positive integer (seconds), got: $LAUNCH_INTERVAL"
  exit 1
fi

rm -rf logs/
# rm -f meetingid-mismatch-*.log

LOGS_DIR="logs/parallel-$(date +%Y%m%d-%H%M%S)"
mkdir -p "$LOGS_DIR"

# make sure it's using the latest version
./build.sh

echo "Starting $NUMBER_OF_MEETINGS stress-test instances"
[[ -n "$MEETING_TIMEOUT" ]] && echo "  Meeting timeout : ${MEETING_TIMEOUT}s"
[[ -n "$LAUNCH_INTERVAL" ]] && echo "  Launch interval : ${LAUNCH_INTERVAL}s (total wall time ≈ $(( (NUMBER_OF_MEETINGS - 1) * LAUNCH_INTERVAL ))s)"
echo "  Logs directory  : $LOGS_DIR"
echo "  Extra args      : ${EXTRA_ARGS[*]:-<none>}"
echo ""

PIDS=()

cleanup() {
  echo ""
  echo "Interrupted — killing all running instances..."
  for pid in "${PIDS[@]}"; do
    kill "$pid" 2>/dev/null || true
  done
  wait
  exit 1
}
trap cleanup SIGINT SIGTERM

launch_meeting() {
  local i="$1"
  local log_file="$LOGS_DIR/meeting-$i.log"
  echo "  [meeting $i] starting → $log_file"
  if [[ -n "$MEETING_TIMEOUT" ]]; then
    timeout "$MEETING_TIMEOUT" ./bbb-stress-test "${EXTRA_ARGS[@]}" >"$log_file" 2>&1 &
  else
    ./bbb-stress-test "${EXTRA_ARGS[@]}" >"$log_file" 2>&1 &
  fi
  PIDS+=($!)
}

for i in $(seq 1 "$NUMBER_OF_MEETINGS"); do
  launch_meeting "$i"

  # sleep between launches (skip after the last one)
  if [[ -n "$LAUNCH_INTERVAL" && "$i" -lt "$NUMBER_OF_MEETINGS" ]]; then
    sleep "$LAUNCH_INTERVAL"
  fi
done

echo ""
echo "All $NUMBER_OF_MEETINGS instances launched. Waiting for them to finish..."
echo ""

FAILED=0
for i in "${!PIDS[@]}"; do
  MEETING_NUM=$((i + 1))
  PID="${PIDS[$i]}"
  # exit code 124 = timeout killed the process — treat as success
  exit_code=0
  wait "$PID" || exit_code=$?
  if [[ "$exit_code" -eq 0 || "$exit_code" -eq 124 ]]; then
    echo "  [meeting $MEETING_NUM] done (pid $PID)"
  else
    echo "  [meeting $MEETING_NUM] FAILED (pid $PID, exit $exit_code)"
    FAILED=$((FAILED + 1))
  fi
done

echo ""
echo "All instances finished. $FAILED failed."
echo ""

MISMATCH_FILES=(meetingid-mismatch-*.log)
if [[ -f "${MISMATCH_FILES[0]}" ]]; then
  TOTAL_MISMATCHES=$(cat "${MISMATCH_FILES[@]}" | wc -l)
  echo "meetingId mismatches detected: $TOTAL_MISMATCHES"
  echo "  (see: ${MISMATCH_FILES[*]})"
else
  echo "No meetingId mismatches detected."
fi
