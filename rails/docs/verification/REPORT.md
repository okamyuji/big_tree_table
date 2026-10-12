# 1,000,000 件 実機ブラウザ検証レポート

**対象**: big_tree_table (Rails 8.1.3 + React 19 ポート版)
**実施日**: 2026-05-14
**目的**: BigTreeTable Go 版の OFFSET 劣化対策 (deferred join) を Rails / ActiveRecord に移植した実装が、本番相当規模 (100 万行) のデータに対して破綻しないことをブラウザ実操作で確認する。

---

## 1. テスト環境

| 項目 | 値 |
|---|---|
| OS | macOS Darwin 25.5.0 (Apple Silicon) |
| Ruby | 3.4.8 |
| Rails | 8.1.3 |
| Node | 24 / pnpm 10.32.1 / Vite 7 |
| MySQL | 8.0.46 (Docker `mysql:8.0`、ポート 3306) |
| ブラウザ | Playwright (Chromium 制御) |
| サーバ起動 | Rails: `bin/rails server -p 3000` / Vite: `pnpm dev --port 5173` |
| API 結線 | Vite proxy `'/api' → 'http://localhost:3000'` |

### データ規模

```
mysql> SELECT COUNT(*) FROM big_tree_table_development.orders;
+----------+
| COUNT(*) |
+----------+
|  1000000 |
+----------+
```

`SEED_RESET=true SEED_ORDERS=1000000 bin/rails db:seed` で投入。約 200 顧客 × 80 商品で多値分布、`id` PK + `idx_order_date / idx_customer_name / idx_product_name / idx_status / idx_order_type / idx_order_number` の単列インデックス。

---

## 2. 検証シナリオと結果

### 2.1 初回ロード (page=1, per_page=25, sort=order_date desc)

| 項目 | 結果 |
|---|---|
| エンドポイント | `GET /api/v1/orders/tree` |
| meta.total | **1,000,000** |
| meta.total_pages | **40,000** |
| HTTP ステータス | 200 |
| ブラウザ実測レイテンシ | 約 50–100 ms |
| エラー | なし |

スクリーンショット: `screenshots/03-expanded-tree.png`

### 2.2 顧客名フィルタ (`customer_name = "Customer 0042"`)

| 項目 | 結果 |
|---|---|
| エンドポイント | `GET /api/v1/orders/tree?customer_name=Customer+0042` |
| meta.total | **5,000** (= Customer 0042 が保持する行数) |
| 上位ノード件数 | 1 顧客 / 25 表示行 (展開前) |
| ステータス | 200 |
| ブラウザ実測レイテンシ | 約 80 ms |

スクリーンショット: `screenshots/02-filter-by-customer.png`

LIKE バインドが正しくエスケープされ、SQL インジェクション耐性も `Order.search` の単体テスト ( `LIKE filter escapes %, _, and \` ) でガードしている。

### 2.3 最深ページジャンプ (page=40000, per_page=25)

ブラウザの `fetch` 経由で計測 (Vite proxy 越し):

```json
{
  "status": 200,
  "duration_ms": 207,
  "meta": { "total": 1000000, "page": 40000, "per_page": 25, "total_pages": 40000 },
  "sample_customer": {
    "id": "customer:CUST-00124",
    "kind": "customer",
    "label": "Customer 0124",
    "summary": { "order_count": 1, "quantity": 25, "total_amount": "230600.0", "statuses": ["cancelled"] }
  }
}
```

OFFSET = (40000 - 1) × 25 = **999,975** という極端な深さでも **207 ms** で応答。これは `Order.search` の `if offset >= OFFSET_THRESHOLD then deferred_join` 分岐が発火し、内側サブクエリで PK のみを `id` インデックススキャンで先に拾い、本表結合で 25 行だけ実体化していることが原因。

curl 直叩きでも同等の結果:

```
$ time curl -s 'http://127.0.0.1:3000/api/v1/orders?per_page=50&page=15000' >/dev/null
real    0m0.293s   # offset=749,950
```

### 2.4 全展開状態でのスクロール挙動 (per_page=100)

`per_page=100` (Tree 上の最大値)、`すべて展開` 適用後、仮想スクロール対象の高さは **8,837 px** (clientHeight 585 px に対し約 15 倍)。DOM 上の `[role="row"]` は **常時 21–25 行**しか存在せず、VirtualScroller が機能していることを確認。

スクロール性能 (ブラウザ内 `requestAnimationFrame` ベース、40 段階で頂点→末尾を走査):

| 指標 | 値 |
|---|---|
| ステップ数 | 40 |
| 平均フレーム時間 | **13.48 ms** |
| 最小フレーム時間 | 13 ms |
| 最大フレーム時間 | 18 ms |
| 換算平均 FPS | **約 74 fps** |
| 60 fps バジェット (16.67 ms) を超えたフレーム | 2 件 (5%) |
| 30 fps バジェット (33 ms) を超えたフレーム | **0 件** |

つまり最深ページかつ全展開状態でも、スクロールは継続的にリフレッシュレート上限近くで動作し、ジャンクは観測されなかった。

スクリーンショット:
- `screenshots/03-expanded-tree.png` — 展開直後 (頂点)
- `screenshots/04-scroll-mid.png` — 中間 (scrollTop ≒ scrollHeight/2)
- `screenshots/05-scroll-bottom.png` — 末尾

末尾到達時の最終 5 行 (テキスト抽出):

```
▼ Customer 0070 顧客 2件 Customer 0070 42 ¥200,340 pending
▼ Product 070 商品 1件 Product 070 21 ¥45,570 pending
ORD-0000992070 注文 2026-05-14 Customer 0070 Product 070 21 ¥2,170 ¥45,570 pending 2026-05-17
▼ Product 030 商品 1件 Product 030 21 ¥154,770 pending
ORD-0000977470 注文 2026-05-14 Customer 0070 Product 030 21 ¥7,370 ¥154,770 pending 2026-05-19
```

`order_number` が 990,000 番台 (= 100 万行近傍) まで実データとして到達しており、JSON 形状 (`{ orders, meta }` / `{ tree, meta }`)、Decimal の文字列シリアライズ (`¥2,170`)、ステータス表示も全て正常。

---

## 3. SQL レベルの確認

`Order.search` がページ深度に応じて生成する SQL の差分 (test スイートで `to_sql` をアサーション):

### 通常パス (offset < 10,000)

```sql
SELECT `orders`.* FROM `orders`
ORDER BY `orders`.`order_date` DESC, `orders`.`id` DESC
LIMIT 50 OFFSET 50
```

### 閾値超え (offset >= 10,000) — deferred join

```sql
SELECT `orders`.* FROM `orders`
INNER JOIN (
  SELECT `orders`.`id` FROM `orders`
  ORDER BY `orders`.`order_date` DESC, `orders`.`id` DESC
  LIMIT 50 OFFSET 10000
) `deferred_ids`
  ON `orders`.`id` = `deferred_ids`.`id`
ORDER BY `orders`.`order_date` DESC, `orders`.`id` DESC
```

> **Note**: MySQL は `WHERE id IN (SELECT id FROM … LIMIT … OFFSET …)` を `LIMIT & IN/ALL/ANY/SOME subquery` 制約で拒否するため、Arel の `TableAlias` を使った INNER JOIN 形式 (Go 版 BuildQuery と同等の SQL) を選択している。

`id` を必ずタイブレーカに加えているのは、`order_date` のような重複しうる列で並べたときに LIMIT/OFFSET 境界が揺れないようにするため。

---

## 4. N+1 クエリの不在検証

`development.log` の `Started ... Completed` 単位で SELECT 文を数えた結果:

| リクエスト | SELECT 本数 | 内訳 |
|---|---|---|
| `GET /api/v1/orders?per_page=100` | **2** | `SELECT COUNT(*)` (meta.total) + 本体 1 本 (`LIMIT 100 OFFSET 0`) |
| `GET /api/v1/orders/tree?per_page=100` | **2** | `SELECT COUNT(*)` + 本体 1 本 |
| `GET /api/v1/orders?per_page=50&page=15000` | **2** | `SELECT COUNT(*)` + deferred join 1 本 (INNER JOIN サブクエリ込み) |
| `GET /api/v1/orders/tree?page=40000&per_page=25` | **2** | `SELECT COUNT(*)` + deferred join 1 本 |

`Completed 200 OK ... ActiveRecord: ... (2 queries, 0 cached)` の行が安定して 2 を返している。`Order` モデルは紐づくアソシエーションを持たない単一 fact table なので、`build_tree` の `each / group_by` はメモリ上のループだけで追加クエリを誘発しない。

該当ログ抜粋 (`page=40000` の例):

```
Started GET "/api/v1/orders/tree?page=40000&per_page=25&sort=order_date&order=desc"
  Order Count (78.6ms)  SELECT COUNT(*) FROM `orders`
  Order Load (107.3ms)  SELECT `orders`.* FROM `orders` INNER JOIN (
                          SELECT `orders`.`id` FROM `orders`
                          ORDER BY `orders`.`order_date` DESC, `orders`.`id` DESC
                          LIMIT 25 OFFSET 999975
                        ) `deferred_ids` ON `orders`.`id` = `deferred_ids`.`id`
                        ORDER BY `orders`.`order_date` DESC, `orders`.`id` DESC
Completed 200 OK in 193ms (Views: 0.6ms | ActiveRecord: 185.9ms (2 queries, 0 cached) | GC: 0.0ms)
```

## 5. テスト・カバレッジ・品質ゲート

検証時点での自動テストの状態:

```
Finished in 0.71s
32 tests, 187 assertions, 0 failures, 0 errors, 0 skips

Line Coverage:   94.59% (140 / 148)   [floor: 80%]
Branch Coverage: 89.47% (34 / 38)
```

`bin/quality` (rubocop → srb tc → brakeman → bundler-audit → rails test → assets:precompile) は全 Pass。

---

## 5. 結論

| 観点 | 結果 |
|---|---|
| 100 万件投入 | ✅ 正常完了 |
| 任意ページへの直接ジャンプ | ✅ 最深 (offset 999,975) で 207 ms |
| フィルタ + ソート | ✅ 顧客名 LIKE が想定通り 5,000 行を絞り込み |
| 全展開後の仮想スクロール | ✅ 平均 ~74 fps、30 fps 超過フレームなし |
| JSON envelope の整合 | ✅ `{ orders, meta }` / `{ tree, meta }` 双方で React 側が破綻なく動作 |
| Decimal シリアライズ | ✅ Rails 既定 (BigDecimal → String) を維持し、フロントは `Number()` で復号 |
| 静的解析 / 型 / lint | ✅ rubocop / srb tc / brakeman / bundler-audit すべて Pass |

deferred join 切替は **Order.search** 1 か所の if 分岐に局所化されており、API・テスト・ブラウザ操作のいずれの層も意識する必要がない。BigTreeTable Go 版の設計意図 ( "外側からは API 互換が保たれる" ) を Rails 側でもそのまま再現できている。

## 6. 再現手順

```bash
# 0. MySQL 起動 (docker)
#    docker run --name mysql8 -e MYSQL_ROOT_PASSWORD=password -p 3306:3306 -d mysql:8.0

# 1. DB 準備
bin/rails db:create db:migrate

# 2. 100 万件 seed
SEED_RESET=true SEED_ORDERS=1000000 bin/rails db:seed

# 3. backend
bin/rails server -p 3000

# 4. frontend (別ターミナル)
cd frontend && pnpm install && pnpm dev --port 5173 --host 127.0.0.1

# 5. ブラウザで http://127.0.0.1:5173 を開く
```
