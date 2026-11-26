#!/usr/bin/env bash
# bench/run.sh の結果ディレクトリから Markdown の表を作る。
# 使い方: bench/summarize.sh bench/results/<日時>
set -euo pipefail
dir="${1:?結果ディレクトリを指定してください}"

echo "| scenario | target | req/s | p50 ms | p90 ms | p99 ms | non-200 | RSS after (MiB) |"
echo "|---|---|---:|---:|---:|---:|---:|---:|"
for f in "$dir"/*-*.json; do
  name="$(basename "$f" .json)"
  t="${name%%-*}"; s="${name#*-}"
  rss="$(awk -F= -v k="rss_after_${s}_kib" '$1==k && $2!="" {printf "%.0f", $2/1024}' "$dir/$t-rss.txt")"
  rss="${rss:--}"
  jq -r --arg t "$t" --arg s "$s" --arg rss "$rss" '
    .metrics as $m
    | [$s, $t,
       ($m.http_reqs.rate | floor | tostring),
       ($m.http_req_duration.med | . * 10 | round / 10 | tostring),
       ($m.http_req_duration["p(90)"] | . * 10 | round / 10 | tostring),
       ($m.http_req_duration["p(99)"] | . * 10 | round / 10 | tostring),
       ($m.checks.fails | tostring),
       $rss]
    | "| " + join(" | ") + " |"' "$f"
done | sort -t'|' -k2,2 -k3,3
