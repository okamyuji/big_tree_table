#!/usr/bin/env bash
# Rails の応答を正として、Go と snails の応答（ステータス、X-Total-Count、
# WWW-Authenticate、JSON 本文）が一致するかを確かめる。
#
# 使い方: PG_URL=postgres://.../big_tree_table_parity bench/parity.sh [go] [snails]
# PG_URL の orders 表は作り直す。ベンチ用の DB を指さないこと。
set -euo pipefail
: "${PG_URL:?PG_URL に parity 用の DB を指定してください}"
export PG_URL
here="$(cd "$(dirname "$0")" && pwd)"
targets=("$@")
[[ ${#targets[@]} -gt 0 ]] || targets=(go snails)

# snails はスキーマを自分の db/schema.sql から読むので、共有スキーマと同じであることを先に確かめる。
cmp -s "$here/db/schema.sql" "$here/../snails/db/schema.sql" || {
  echo "snails/db/schema.sql が bench/db/schema.sql と違います。コピーし直してください" >&2
  exit 1
}

# 3万行で offset 10,000 以上の deferred join 経路に届く。
"$here/db/setup.sh" 30000 >/dev/null
psql "$PG_URL" -v ON_ERROR_STOP=1 -X -q -f "$here/db/parity_extra.sql"

# shellcheck source=lib/servers.sh
source "$here/lib/servers.sh"
BENCH_LOG_DIR="$(mktemp -d "${TMPDIR:-/tmp}/btt.XXXXXX")"
trap stop_servers EXIT

declare -A port=([rails]=3201 [go]=3202 [snails]=3203)
start_rails "${port[rails]}"
for t in "${targets[@]}"; do "start_$t" "${port[$t]}"; done

paths=(
  "/api/v1/orders"
  "/api/v1/orders?per_page=1"
  "/api/v1/orders?page=3&per_page=25&sort=order_date&order=desc"
  "/api/v1/orders?sort=total_amount&order=DESC&per_page=7"
  "/api/v1/orders?sort=unit_price&order=asc&per_page=5"
  "/api/v1/orders?sort=created_at&order=desc&per_page=5"
  "/api/v1/orders?sort=bogus&order=sideways&per_page=3"
  "/api/v1/orders?page=abc&per_page=-5"
  "/api/v1/orders?page=%20%202&per_page=+3x"
  "/api/v1/orders?page=1_0&per_page=2__0"
  "/api/v1/orders?per_page=501&sort=quantity"
  "/api/v1/orders?page=99999999999999999999"
  "/api/v1/orders?page=0&per_page=0"
  "/api/v1/orders?page=201&per_page=50&sort=customer_name&order=desc"
  "/api/v1/orders?page=401&per_page=50"
  "/api/v1/orders?page=1001&per_page=10&status=shipped&sort=order_date"
  "/api/v1/orders?order_type=rush&status=shipped&per_page=5"
  "/api/v1/orders?customer_name=Customer%20000&per_page=5"
  "/api/v1/orders?customer_name=100%25"
  "/api/v1/orders?customer_name=%25"
  "/api/v1/orders?customer_name=_x"
  "/api/v1/orders?customer_name=x%5Cy"
  "/api/v1/orders?customer_name=O%27Brien"
  "/api/v1/orders?customer_name=%3F&product_name=Widget%20%3F"
  "/api/v1/orders?product_name=a%3Fb"
  "/api/v1/orders?customer_name=customer"
  "/api/v1/orders?date_from=2026-02-01&date_to=2026-02-28&per_page=100"
  "/api/v1/orders?date_from=2024-02-29&date_to=2024-02-29"
  "/api/v1/orders?date_from=2026-13-45&date_to=2026-02-30&per_page=2"
  "/api/v1/orders?date_from=2026-1-5&date_to=20260105&per_page=2"
  "/api/v1/orders?status="
  "/api/v1/orders?order_type=nothing"
  "/api/v1/orders/tree"
  "/api/v1/orders/tree?per_page=500&sort=customer_code"
  "/api/v1/orders/tree?customer_name=Odd"
  "/api/v1/orders/tree?per_page=500&sort=total_amount&order=desc"
  "/api/v1/orders/tree?per_page=500&sort=total_amount&order=asc"
  "/api/v1/orders/tree?page=250&per_page=100&order_type=rush"
)

fetch() { # fetch <port> <path> <curl auth args...> -> "<status>|<x-total-count>|<www-authenticate>" と本文
  local p="$1" path="$2"; shift 2
  local hdr body
  hdr="$(mktemp "$BENCH_LOG_DIR/h.XXXXXX")"; body="$(mktemp "$BENCH_LOG_DIR/b.XXXXXX")"
  curl -s -D "$hdr" -o "$body" "$@" "http://127.0.0.1:$p$path"
  printf '%s|%s|%s\n' \
    "$(awk 'NR==1{print $2}' "$hdr")" \
    "$(awk 'tolower($1)=="x-total-count:"{print $2}' "$hdr" | tr -d '\r')" \
    "$(awk 'tolower($1)=="www-authenticate:"{$1=""; print substr($0,2)}' "$hdr" | tr -d '\r')"
  if jq -S . "$body" 2>/dev/null; then :; else cat "$body"; fi
  rm -f "$hdr" "$body"
}

failures=0
compare() { # compare <label> <path> <curl auth args...>
  local label="$1" path="$2"; shift 2
  local want got
  want="$(fetch "${port[rails]}" "$path" "$@")"
  for t in "${targets[@]}"; do
    got="$(fetch "${port[$t]}" "$path" "$@")"
    if [[ "$got" == "$want" ]]; then
      printf 'ok   %-7s %s %s\n' "$t" "$label" "$path"
    else
      printf 'FAIL %-7s %s %s\n' "$t" "$label" "$path"
      printf "%s\n" "$want" >"$BENCH_LOG_DIR/want"; printf "%s\n" "$got" >"$BENCH_LOG_DIR/got"
      diff "$BENCH_LOG_DIR/want" "$BENCH_LOG_DIR/got" | head -20 || true
      failures=$((failures + 1))
    fi
  done
}

auth=(-u "$API_BASIC_AUTH_USER:$API_BASIC_AUTH_PASSWORD")
for path in "${paths[@]}"; do compare auth "$path" "${auth[@]}"; done

# 認証の失敗は本文の書式が実装ごとに違うので、ステータスとヘッダだけを比べる。
status_only() { # status_only <label> <path> <curl args...>
  local label="$1" path="$2"; shift 2
  local want got
  want="$(fetch "${port[rails]}" "$path" "$@" | head -1)"
  for t in "${targets[@]}"; do
    got="$(fetch "${port[$t]}" "$path" "$@" | head -1)"
    if [[ "$got" == "$want" ]]; then
      printf 'ok   %-7s %s %s (%s)\n' "$t" "$label" "$path" "$got"
    else
      printf 'FAIL %-7s %s %s rails=%s %s=%s\n' "$t" "$label" "$path" "$want" "$t" "$got"
      failures=$((failures + 1))
    fi
  done
}
status_only no-auth "/api/v1/orders"
status_only bearer "/api/v1/orders" -H "Authorization: Bearer nope"
status_only wrong-user "/api/v1/orders" -u "nobody:$API_BASIC_AUTH_PASSWORD"
for i in $(seq 1 7); do status_only "wrong-pass-$i" "/api/v1/orders" -u "$API_BASIC_AUTH_USER:wrong"; done
status_only right-after-8-failures "/api/v1/orders?per_page=1" "${auth[@]}"
for i in $(seq 9 10); do status_only "wrong-pass-$i" "/api/v1/orders" -u "$API_BASIC_AUTH_USER:wrong"; done
status_only right-after-10-failures "/api/v1/orders?per_page=1" "${auth[@]}"
status_only not-found "/api/v1/nothing" "${auth[@]}"

echo "failures: $failures"
[[ $failures -eq 0 ]]
