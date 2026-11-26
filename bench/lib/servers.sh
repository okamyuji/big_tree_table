# shellcheck shell=bash
# 3つの API サーバーを起動・停止する関数。bench/parity.sh と bench/run.sh が source する。
# sandbox では別のコマンドで起動したプロセスに触れられないため、起動・計測・停止を1つのスクリプト内で行う。
#
# 必要な環境変数: PG_URL
# 任意: WORKERS（プロセス数、既定 1）、THREADS（Rails のスレッド数と各 DB プール、既定 5）

BENCH_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
WORKERS="${WORKERS:-1}"
THREADS="${THREADS:-5}"
SERVER_PIDS=()

export API_BASIC_AUTH_USER="${API_BASIC_AUTH_USER:-bench}"
export API_BASIC_AUTH_PASSWORD="${API_BASIC_AUTH_PASSWORD:-$(openssl rand -hex 16)}"

wait_listening() {
  local port="$1" i
  for i in $(seq 1 120); do
    [[ "$(curl -s -o /dev/null -w '%{http_code}' "http://127.0.0.1:$port/")" != "000" ]] && return 0
    sleep 0.5
  done
  echo "port $port did not open" >&2
  return 1
}

start_rails() {
  local port="$1"
  # macOS: libpq の GSS 初期化が Puma の fork 後に落ちるため、GSS を使わず fork 安全検査も外す。
  (cd "$BENCH_ROOT/rails" && PGGSSENCMODE=disable OBJC_DISABLE_INITIALIZE_FORK_SAFETY=YES RAILS_ENV=bench SECRET_KEY_BASE="$(openssl rand -hex 64)" \
    RAILS_LOG_LEVEL=warn WEB_CONCURRENCY="$WORKERS" RAILS_MAX_THREADS="$THREADS" PORT="$port" \
    exec bundle exec puma -C config/puma.rb) >"$BENCH_LOG_DIR/rails.log" 2>&1 &
  SERVER_PIDS+=($!)
  wait_listening "$port"
}

start_go() {
  local port="$1" bin="${TMPDIR:-/tmp}/btt-go"
  (cd "$BENCH_ROOT/go/backend" && go build -o "$bin" .)
  # Go は1プロセスで全コアを使う。DB プールは他の2者の合計と同じ大きさにする。
  DB_DRIVER=postgres PORT="$port" DB_MAX_CONNS=$((WORKERS * THREADS)) DB_MAX_IDLE_CONNS=$((WORKERS * THREADS)) \
    "$bin" >"$BENCH_LOG_DIR/go.log" 2>&1 &
  SERVER_PIDS+=($!)
  wait_listening "$port"
}

start_snails() {
  local port="$1"
  PORT="$port" SNAILS_ENV=production SNAILS_LOG_LEVEL=warn WORKERS="$WORKERS" DATABASE_POOL_SIZE="$THREADS" \
    "$BENCH_ROOT/snails/build/snails" >"$BENCH_LOG_DIR/snails.log" 2>&1 &
  SERVER_PIDS+=($!)
  wait_listening "$port"
}

stop_servers() {
  local pid
  for pid in "${SERVER_PIDS[@]}"; do
    kill "$pid" 2>/dev/null || true
  done
  for pid in "${SERVER_PIDS[@]}"; do
    wait "$pid" 2>/dev/null || true
  done
  SERVER_PIDS=()
}
