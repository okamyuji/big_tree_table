# BigTreeTable (Ruby on Rails ポート版)

100 万件の受発注データを、顧客、商品、注文の階層で表示する TreeTable デモアプリケーションの Ruby on Rails 移植版です。

このリポジトリは、オリジナルの [BigTreeTable](https://github.com/okamyuji/BigTreeTable) (Go + React) を Rails 8.1 + ActiveRecord に移植したものです。バックエンドは MySQL 上の 100 万件データをソート、フィルター、ページネーションし、TreeTable 専用 API で `customer -> product -> order` の階層レスポンスを返します。フロントエンドは React と TypeScript を流用し、展開状態を反映した可視ノードだけを独自の仮想スクロールへ渡して描画します。

## 主な機能

- 100 万件の受発注データを MySQL に seed
- 顧客、商品、注文の 3 階層 TreeTable 表示
- 親行の展開、折りたたみ、すべて展開、すべて折りたたみ
- サーバーサイドのソート、フィルター、ページネーション
- 固定行高の独自仮想スクロール
- ActiveRecord + Arel による deferred join (`offset >= 10,000` で自動切替)
- Sorbet 静的型検査 + RuboCop + minitest + SimpleCov 80% 下限
- pre-commit と GitHub Actions での Gitleaks secret scan

## 技術構成

| 領域 | 技術 |
| --- | --- |
| Backend | Ruby 3.4、Rails 8.1、ActiveRecord、Arel、mysql2 |
| Database | MySQL 8 |
| Frontend | React 19、TypeScript、Tailwind CSS v4、Vite+ |
| Type check | Sorbet (sorbet-static-and-runtime + tapioca) |
| Lint / format | RuboCop (rubocop-rails-omakase + minitest + performance) |
| Unit test | Minitest + SimpleCov (line coverage 80% 下限)、Vitest |
| E2E | Playwright (frontend/e2e) |
| Security scan | Gitleaks、pre-commit、GitHub Actions、Brakeman、bundler-audit |

## 起動方法

### 1. MySQL を Docker で起動

ホスト側ポートは `3306` をそのまま公開します (オリジナル Go 版では 3307 を使っていましたが、Rails の `database.yml` 既定値に合わせています)。

```bash
docker run --name mysql8 \
  -e MYSQL_ROOT_PASSWORD=password \
  -p 3306:3306 -d mysql:8.0
```

`compose.yml` から起動する場合は本 README 末尾の「Docker Compose」を参照してください。

### 2. データベース準備

```bash
bundle install
bin/rails db:create db:migrate
```

### 3. 100 万件 seed

```bash
SEED_RESET=true SEED_ORDERS=1000000 bin/rails db:seed
```

環境変数で件数を調整できます (デフォルトは 10,000 件)。

### 4. バックエンド (Rails) を起動

```bash
bin/rails server -p 3000
```

### 5. フロントエンド (Vite) を起動

```bash
cd frontend
pnpm install
pnpm dev --port 5173 --host 127.0.0.1
```

ブラウザで <http://localhost:5173> を開きます。Vite の開発サーバが `/api` リクエストを Rails (`http://localhost:3000`) にプロキシします。

## Docker Compose

すべてのサービスを Docker で起動することもできます。

```bash
docker compose up -d
```

Docker Compose で起動した場合、フロントエンドは <http://localhost:3000> で配信されます (Nginx 経由)。

## ポート

| サービス | ポート | 用途 |
| --- | --- | --- |
| MySQL | 3306 | ローカル開発用 DB (デフォルト) |
| Backend (Rails) | 3000 | API サーバ |
| Frontend Docker | 3000 (compose) | Nginx 配信 |
| Frontend local | 5173 | Vite 開発サーバ |

## データ構造

物理テーブルは元の `orders` fact table を維持します。TreeTable 用の階層はバックエンドでレスポンスとして構築します。

```text
customer
└── product
    └── order
```

`GET /api/v1/orders/tree` は現在ページに含まれる注文を、顧客、商品、注文の順に階層化して返します。ページング単位は注文行です。そのため同じ顧客が別ページにも現れることがあります。

## API

エンドポイントは Rails 流の versioned namespace (`/api/v1/...`) で公開しています。レスポンスは `{ <resource>, meta }` の二段構成。

### `GET /api/v1/orders`

平坦な注文一覧。BigTable 版と同形の `Order[]` を返します。

| パラメータ | 型 | 既定値 | 説明 |
| --- | --- | --- | --- |
| `page` | number | 1 | ページ番号 |
| `per_page` | number | 50 | 1 ページあたりの注文件数 (上限 500) |
| `sort` | string | `id` | ソート対象カラム (ホワイトリスト制) |
| `order` | `asc` / `desc` | `asc` | ソート方向 |
| `order_type` | string | なし | 種別フィルター (完全一致) |
| `status` | string | なし | ステータスフィルター (完全一致) |
| `customer_name` | string | なし | 顧客名の部分一致 (LIKE エスケープ済) |
| `product_name` | string | なし | 商品名の部分一致 (LIKE エスケープ済) |
| `date_from` | YYYY-MM-DD | なし | 注文日の開始日 |
| `date_to` | YYYY-MM-DD | なし | 注文日の終了日 |

レスポンス例:

```json
{
  "orders": [
    {
      "id": 1,
      "order_number": "ORD-0000000001",
      "order_type": "rush",
      "order_date": "2026-05-13",
      "customer_name": "Customer 0001",
      "customer_code": "CUST-00001",
      "product_name": "Product 001",
      "product_code": "PROD-00001",
      "quantity": 2,
      "unit_price": "101.0",
      "total_amount": "202.0",
      "status": "confirmed",
      "delivery_date": "2026-05-15",
      "notes": "auto-seed #1",
      "created_at": "2026-05-14T12:54:59.283Z",
      "updated_at": "2026-05-14T12:54:59.283Z"
    }
  ],
  "meta": {
    "total": 1000000,
    "page": 1,
    "per_page": 1,
    "total_pages": 1000000
  }
}
```

> **Decimal**: `unit_price` と `total_amount` は ActiveRecord の既定 (BigDecimal → 文字列) でシリアライズしています。フロントは `Number(...)` で復号して表示。

### `GET /api/v1/orders/tree`

TreeTable 用の階層データを取得します。クエリパラメータは `/api/v1/orders` と同じ。

レスポンス例:

```json
{
  "tree": [
    {
      "id": "customer:CUST-00135",
      "kind": "customer",
      "depth": 0,
      "label": "Customer 0135",
      "summary": {
        "order_count": 2,
        "quantity": 108,
        "total_amount": "554580.0",
        "statuses": ["pending"]
      },
      "children": [
        {
          "id": "customer:CUST-00135:product:PROD-00055",
          "kind": "product",
          "depth": 1,
          "label": "Product 055",
          "summary": { "order_count": 2, "quantity": 72, "total_amount": "369720.0", "statuses": ["pending"] },
          "children": [
            { "id": "order:999735", "kind": "order", "depth": 2, "label": "ORD-0000999735", "order": { "...": "..." }, "summary": { "...": "..." }, "children": [] }
          ]
        }
      ]
    }
  ],
  "meta": { "total": 1000000, "page": 1, "per_page": 25, "total_pages": 40000 }
}
```

## OFFSET 劣化対策 — deferred join

`Order.search` は `offset >= 10_000` で **deferred join** に自動切替します。SQL 形は BigTreeTable Go 版の `BuildQuery` と同等。

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

> MySQL は `WHERE id IN (SELECT id FROM ... LIMIT ... OFFSET ...)` 形を `LIMIT & IN/ALL/ANY/SOME subquery` 制約で拒否するため、Arel の `Arel::Nodes::TableAlias` を使った INNER JOIN を採用しています。

実機ブラウザでの 100 万件検証結果は [`docs/verification/REPORT.md`](docs/verification/REPORT.md) を参照。最深ページ (offset = 999,975) でも **207 ms**、全展開後の仮想スクロールでも **平均 ~74 fps / 30fps 超過 0 件**。

## フロントエンド構成

- `src/components/TreeTable.tsx`: TreeTable 画面本体
- `src/components/TreeTableRow.tsx`: 顧客、商品、注文行の描画
- `src/components/TreeTableHeader.tsx`: TreeTable 用ヘッダー
- `src/hooks/useTreeTableData.ts`: `/api/v1/orders/tree` の取得と状態管理
- `src/utils/treeData.ts`: 展開状態を反映した可視ノードの flatten 処理
- `src/components/VirtualScroller.tsx`: 固定行高の仮想スクロール

## テスト

バックエンド (Minitest + SimpleCov 80% 下限):

```bash
bin/rails test
# Coverage report generated for Minitest to coverage/
```

フロントエンド:

```bash
cd frontend
pnpm exec tsc -b      # 型検査
pnpm exec vp fmt      # 整形
pnpm exec eslint .    # lint
pnpm test             # Vitest
```

E2E:

```bash
cd frontend
pnpm exec playwright test
```

## 統合品質ゲート

`bin/quality` 1 コマンドで全 6 ゲートを直列実行 (失敗時は即時 exit):

```bash
bin/quality
# 1. rubocop --no-color           (formatter + lint)
# 2. srb tc                       (Sorbet 静的型検査)
# 3. brakeman --quiet --exit-on-warn (Rails セキュリティスキャン)
# 4. bundler-audit check --update (CVE スキャン)
# 5. bin/rails test               (Minitest + SimpleCov 80%)
# 6. bin/rails assets:precompile  (本番アセットビルド)
```

`rake quality` でも同じことができます (個別タスクは `quality:rubocop` / `quality:sorbet` / …)。

## Gitleaks

pre-commit hook をインストール:

```bash
pre-commit install
```

手動実行:

```bash
pre-commit run --all-files
gitleaks git --redact --no-banner --verbose
gitleaks dir . --redact --no-banner --verbose
```

GitHub Actions でも push、pull request、手動実行時に Gitleaks を実行します (`.github/workflows/ci.yml`)。

## ディレクトリ構成

```text
big_tree_table/
├── .github/
│   └── workflows/
│       └── ci.yml              # gitleaks + bin/quality
├── .pre-commit-config.yaml     # pre-commit + gitleaks
├── app/
│   ├── controllers/
│   │   └── api/
│   │       ├── base_controller.rb
│   │       └── v1/
│   │           └── orders_controller.rb
│   └── models/
│       └── order.rb            # search / deferred_join / build_tree
├── bin/
│   └── quality                 # 統合品質ゲート
├── config/
│   ├── database.yml
│   ├── initializers/cors.rb
│   └── routes.rb
├── db/
│   ├── migrate/
│   ├── schema.rb
│   └── seeds.rb                # 1M 件 seeder
├── docs/
│   └── verification/
│       ├── REPORT.md           # 100万件検証レポート
│       └── screenshots/        # 検証時 UI スクリーンショット
├── frontend/                   # React 19 + Vite + TypeScript
│   ├── e2e/
│   ├── src/
│   │   ├── api/
│   │   ├── components/
│   │   ├── hooks/
│   │   ├── types/
│   │   └── utils/
│   ├── tests/
│   └── vite.config.ts          # /api → http://localhost:3000 proxy
├── lib/
│   └── tasks/quality.rake      # rake quality:*
├── sorbet/                     # tapioca 生成 RBI
├── test/
│   ├── controllers/api/v1/
│   └── models/
└── compose.yml                 # mysql + backend + frontend
```
