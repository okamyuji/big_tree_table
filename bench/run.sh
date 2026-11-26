#!/usr/bin/env bash
# Rails / Go / snails を1つずつ起動して k6 で負荷をかける。同時には動かさない。
# シナリオごとにサーバーを起動し直し、前のシナリオで膨らんだメモリなどの影響を持ち越さない。
# 1シナリオは「起動 → ウォームアップ → 計測 → 停止」で、実装を替えるときはクールダウンを置く。
#
# 使い方: PG_URL=postgres://.../big_tree_table_bench bench/run.sh
# 任意の環境変数（既定値）:
#   TARGETS="rails go snails"  SCENARIOS="list deep tree filter"
#   RAILS_WORKERS=<コア数> THREADS=5 VUS=32 DURATION=30s WARMUP=10s COOLDOWN=60
#   RESTART_PER_SCENARIO=1  MAX_LOAD=<コア数の半分>  LOAD_WAIT_LIMIT=300  K6="mise exec k6@2.3.0 -- k6"
set -euo pipefail
: "${PG_URL:?PG_URL にベンチ用の DB（bench/db/setup.sh で投入済み）を指定してください}"
export PG_URL
here="$(cd "$(dirname "$0")" && pwd)"

TARGETS="${TARGETS:-rails go snails}"
SCENARIOS="${SCENARIOS:-list deep tree filter}"
export RAILS_WORKERS="${RAILS_WORKERS:-$(getconf _NPROCESSORS_ONLN)}" THREADS="${THREADS:-5}"
VUS="${VUS:-32}" DURATION="${DURATION:-30s}" WARMUP="${WARMUP:-10s}" COOLDOWN="${COOLDOWN:-60}"
# 1以外にすると、シナリオを同じプロセスで続けて流す（順番の影響を見るため）。
RESTART_PER_SCENARIO="${RESTART_PER_SCENARIO:-1}"
# ほかのプロセスが CPU を使っていると数字が揺れるので、1分間のロードアベレージがこの値以下になるまで待つ。
MAX_LOAD="${MAX_LOAD:-$(($(getconf _NPROCESSORS_ONLN) / 2))}"
LOAD_WAIT_LIMIT="${LOAD_WAIT_LIMIT:-300}"
K6="${K6:-mise exec k6@2.3.0 -- k6}"
# カウンタの処理は残しつつ、429 で計測が崩れないよう上限を上げる。
export API_RATE_LIMIT=1000000000

# shellcheck source=lib/servers.sh
source "$here/lib/servers.sh"

max_conns="$(psql "$PG_URL" -XAtc 'SHOW max_connections')"
if ((DB_CONNS >= max_conns)); then
  echo "DB の接続数 ${DB_CONNS}（RAILS_WORKERS × THREADS）が PostgreSQL の max_connections ${max_conns} 以上です。" >&2
  echo "RAILS_WORKERS か THREADS を下げるか、max_connections を上げてください。" >&2
  exit 1
fi

out="$here/results/$(date +%Y%m%d-%H%M%S)"
mkdir -p "$out"
BENCH_LOG_DIR="$out/logs"
mkdir -p "$BENCH_LOG_DIR"
trap stop_servers EXIT

declare -A port=([rails]=3301 [go]=3302 [snails]=3303)

# サーバーと子プロセス（Puma や snails の worker）の RSS の合計（KiB）。
# ps が使えない環境（Claude Code の sandbox など）では空を返す。
tree_rss() {
  local pids=("${SERVER_PIDS[@]}") pid
  for pid in "${SERVER_PIDS[@]}"; do
    pids+=($(pgrep -P "$pid" 2>/dev/null || true))
  done
  ps -o rss= -p "$(IFS=,; echo "${pids[*]}")" 2>/dev/null | awk '{s+=$1} END {if (NR) print s}' || true
}

load1() { uptime | sed -E 's/.*load averages?: *([0-9.]+).*/\1/'; }

# ロードアベレージが MAX_LOAD 以下になるまで待ち、そのときの値を返す。上限時間を超えたら止める。
wait_for_quiet() {
  local waited=0 load
  while load="$(load1)"; awk -v l="$load" -v m="$MAX_LOAD" 'BEGIN { exit !(l > m) }'; do
    if ((waited >= LOAD_WAIT_LIMIT)); then
      echo "ロードアベレージ ${load} が ${LOAD_WAIT_LIMIT} 秒待っても ${MAX_LOAD} 以下に下がりません。ほかの重い処理を止めてください。" >&2
      exit 1
    fi
    sleep 10
    waited=$((waited + 10))
  done
  echo "$load"
}

k6_run() { # k6_run <target> <scenario> <duration> <summary file or empty>
  local args=(run --quiet --no-color --no-usage-report -e "BASE_URL=http://127.0.0.1:${port[$1]}" -e "SCENARIO=$2"
    -e "VUS=$VUS" -e "DURATION=$3" -e "AUTH=$API_BASIC_AUTH_USER:$API_BASIC_AUTH_PASSWORD")
  [[ -n "$4" ]] && args+=(--summary-export "$4")
  $K6 "${args[@]}" "$here/k6/api.js" >/dev/null
}

{
  echo "date=$(date '+%Y-%m-%d %H:%M:%S %Z')"
  echo "host=$(uname -srm) cpus=$(getconf _NPROCESSORS_ONLN)"
  # PostgreSQL が Docker の VM 内で動くとき、DB に重いシナリオの上限はこの CPU 数で決まる。
  echo "docker_vm_cpus=$(docker info --format '{{.NCPU}}' 2>/dev/null || echo unknown)"
  echo "restart_per_scenario=$RESTART_PER_SCENARIO rails_workers=$RAILS_WORKERS threads=$THREADS db_conns=$((RAILS_WORKERS * THREADS)) vus=$VUS duration=$DURATION warmup=$WARMUP cooldown=${COOLDOWN}s"
  echo "rows=$(psql "$PG_URL" -XAtc 'SELECT count(*) FROM orders')"
  echo "commit=$(git -C "$here" rev-parse --short HEAD)"
} >"$out/env.txt"

first=1
for t in $TARGETS; do
  if [[ $first -eq 0 ]]; then
    echo "cooldown ${COOLDOWN}s"
    sleep "$COOLDOWN"
  fi
  first=0
  echo "== $t"
  : >"$out/$t-rss.txt"
  [[ "$RESTART_PER_SCENARIO" == 1 ]] || "start_$t" "${port[$t]}"
  for s in $SCENARIOS; do
    quiet_load="$(wait_for_quiet)"
    echo "load1_before_${s}=${quiet_load}" >>"$out/$t-rss.txt"
    if [[ "$RESTART_PER_SCENARIO" == 1 ]]; then
      "start_$t" "${port[$t]}"
      echo "rss_idle_before_${s}_kib=$(tree_rss)" >>"$out/$t-rss.txt"
    fi
    k6_run "$t" "$s" "$WARMUP" ""
    k6_run "$t" "$s" "$DURATION" "$out/$t-$s.json"
    echo "rss_after_${s}_kib=$(tree_rss)" >>"$out/$t-rss.txt"
    if [[ "$RESTART_PER_SCENARIO" == 1 ]]; then stop_servers; fi
    echo "   $s done"
  done
  stop_servers
done

"$here/summarize.sh" "$out" | tee "$out/summary.md"
