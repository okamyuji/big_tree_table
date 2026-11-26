#!/usr/bin/env bash
# 使い方: PG_URL=postgres://... bench/db/setup.sh [行数（既定 1000000）]
set -euo pipefail
: "${PG_URL:?PG_URL を設定してください}"
dir="$(cd "$(dirname "$0")" && pwd)"
rows="${1:-1000000}"
PGOPTIONS="-c client_min_messages=warning" psql "$PG_URL" -v ON_ERROR_STOP=1 -X -q -f "$dir/schema.sql"
psql "$PG_URL" -v ON_ERROR_STOP=1 -X -q -v seed_orders="$rows" -f "$dir/seed.sql"
psql "$PG_URL" -X -At -c "SELECT count(*), sum(total_amount) FROM orders"
