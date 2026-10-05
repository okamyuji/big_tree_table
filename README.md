# BigTreeTable (Ruby on Rails ポート版)

このリポジトリは、100万件の受発注データを顧客、商品、注文の階層で表示するTreeTableデモアプリケーションを、Ruby on Railsへ移植したものです。

移植元はオリジナルの[BigTreeTable](https://github.com/okamyuji/BigTreeTable)（Go + React）で、移植先はRails 8.1 + ActiveRecordです。バックエンドはMySQL上の100万件のデータをソート、フィルター、ページネーションし、TreeTable専用APIで`customer -> product -> order`の階層レスポンスを返します。フロントエンドはReactとTypeScriptを流用し、展開状態を反映した可視ノードだけを独自の仮想スクロールへ渡して描画します。

## 主な機能

- 100万件の受発注データをMySQLにseedする
- 顧客、商品、注文の3階層をTreeTableで表示する
- 親行の展開、折りたたみ、すべて展開、すべて折りたたみ
- サーバーサイドのソート、フィルター、ページネーション
- 固定行高の独自仮想スクロール
- ActiveRecordとArelによるdeferred join（`offset >= 10,000`で自動的に切り替える）
- Sorbetの静的型検査、RuboCop、minitest、SimpleCovの80%下限
- pre-commitとGitHub ActionsでのGitleaksによるsecret scan

## 技術構成

| 領域 | 技術 |
| --- | --- |
| Backend | Ruby 3.4、Rails 8.1、ActiveRecord、Arel、mysql2 |
| Database | MySQL 8 |
| Frontend | React 19、TypeScript、Tailwind CSS v4、Vite+ |
| Type check | Sorbet (sorbet-static-and-runtime + tapioca) |
| Lint / format | RuboCop (rubocop-rails-omakase + minitest + performance) |
| Unit test | Minitest + SimpleCov（line coverageの下限80%）、Vitest |
| E2E | Playwright (frontend/e2e) |
| Security scan | Gitleaks、pre-commit、GitHub Actions、Brakeman、bundler-audit |

## 起動方法

### 1. MySQLをDockerで起動

ホスト側のポートは`3306`をそのまま公開します。オリジナルのGo版は3307を使っていましたが、Railsの`database.yml`の既定値に合わせました。

```bash
docker run --name mysql8 \
  -e MYSQL_ROOT_PASSWORD=password \
  -p 3306:3306 -d mysql:8.0
```

`compose.yml`から起動する場合は、このREADME末尾の「Docker Compose」を参照してください。

### 2. データベース準備

```bash
bundle install
bin/rails db:create db:migrate
```

### 3. 100万件seed

```bash
SEED_RESET=true SEED_ORDERS=1000000 bin/rails db:seed
```

件数は環境変数で調整でき、既定値は10,000件です。

### 4. バックエンド (Rails) を起動

```bash
bin/rails server -p 3000
```

`/api`はHTTP Basic認証で保護します。資格情報は環境変数`API_BASIC_AUTH_USER`と`API_BASIC_AUTH_PASSWORD`から読みます。

| 環境 | 2つとも設定 | どちらかが未設定 |
| --- | --- | --- |
| development / test | Basic認証を要求 | 認証なしで応答 |
| production | Basic認証を要求 | すべて401で拒否 |

認証を有効にして起動する例です。

```bash
API_BASIC_AUTH_USER=viewer API_BASIC_AUTH_PASSWORD='<任意のパスワード>' bin/rails server -p 3000
```

パスワードの総当たりを防ぐため、`/api`は同じIPアドレスからの呼び出しを1分あたり300回までに制限し、超えた分には429を返します。認証に失敗した呼び出しも、この回数に数えます。

これとは別に、誤った資格情報を送った呼び出しは、IPアドレスごとに最初の失敗から15分で10回までです。10回に達したIPアドレスには、その15分が過ぎるまで、正しい資格情報でも429を返します。資格情報を付けない呼び出し（ブラウザが最初に送るもの）は数えません。

初回のAPI呼び出しではブラウザが認証ダイアログを出すので、そこで資格情報を入力してください。以後はブラウザが自動で送ります。Playwrightのe2eは、同じ環境変数があれば`httpCredentials`として送ります。

### 5. フロントエンド (Vite) を起動

```bash
cd frontend
pnpm install
pnpm dev --port 5173 --host 127.0.0.1
```

ブラウザで<http://localhost:5173>を開きます。Viteの開発サーバは、`/api`へのリクエストをRails（`http://localhost:3000`）へプロキシします。

## Docker Compose

全サービスをDockerでまとめて起動する方法もあります。

```bash
docker compose up -d
```

Docker Composeで起動した場合も、フロントエンドはViteの開発サーバとして<http://localhost:5173>から配信されます。composeは3つのポートをすべて`127.0.0.1`に限定して公開します。Linuxでは、ループバックに公開したポートへ同じネットワークの他ホストから届く不具合を修正したDocker Engine 28.0以上を使ってください。

APIのBasic認証を有効にするには、ホストのシェルで`API_BASIC_AUTH_USER`と`API_BASIC_AUTH_PASSWORD`を設定してから`docker compose up -d`を実行してください。composeはこの2つをbackendに渡します。

## 本番構成

本番では、`frontend/Dockerfile`のNginxを前段に置き、そこでTLSを終端します。ブラウザは`/api`を呼ぶたびにBasic認証の資格情報を送るので、ブラウザとNginxの間は必ずHTTPSにします。

- Nginxは443番でTLS 1.2と1.3だけを受けます。80番への要求は、同じパスの`https://`へ301で転送します。
- Nginxは`/api/`を`backend:80`へ転送します。`backend`は`Dockerfile`のイメージで、Thrusterが80番の平文で受けてRailsへ渡します。
- Railsは`config.assume_ssl`ですべての要求をHTTPSとして扱い、`config.force_ssl`でHSTSを付けます。この前提は、Nginxより後ろをホストの外へ公開しないことで成り立ちます。

`compose.production.yml`がこの構成です。公開するのはNginxの80番と443番だけで、backendはNginxからしか届きません。証明書と秘密鍵は、`TLS_CERT_DIR`のディレクトリに`tls.crt`と`tls.key`として置きます。このディレクトリはリポジトリの外に置いてください。

```bash
export TLS_CERT_DIR=/etc/big_tree_table/certs
export RAILS_MASTER_KEY=... API_BASIC_AUTH_USER=... API_BASIC_AUTH_PASSWORD=...
export DB_HOST=... BIG_TREE_TABLE_DATABASE_PASSWORD=...
docker compose -f compose.production.yml up -d --build
```

どれかの変数が未設定なら、composeは起動前にエラーで止まります。証明書が見つからないときは、Nginxが起動しません。

自分のマシンでこの構成を試すときは、[mkcert](https://github.com/FiloSottile/mkcert)で`localhost`の証明書を作ってください。開発用の`compose.yml`はViteの開発サーバを使うので、証明書は要りません。

```bash
mkdir -p ~/.local/share/big_tree_table/certs
mkcert -install
mkcert -cert-file ~/.local/share/big_tree_table/certs/tls.crt \
       -key-file ~/.local/share/big_tree_table/certs/tls.key localhost 127.0.0.1
export TLS_CERT_DIR=~/.local/share/big_tree_table/certs
```

`config/deploy.yml`のKamalはbackendのイメージだけを配備し、Nginxを通りません。Kamalで配備するときも、このNginxを前段に置いてください。

## ポート

| サービス | ポート | 用途 |
| --- | --- | --- |
| MySQL | 3306 | ローカル開発用DB（既定） |
| Backend (Rails) | 3000 | 開発用APIサーバ |
| Frontend (Vite) | 5173 | 開発サーバ（ローカルとcompose） |
| Backend 本番イメージ | 80 | Thruster経由のAPIサーバ（Nginxからのみ） |
| Frontend 本番イメージ | 443 | NginxのTLS終端と配信 |
| Frontend 本番イメージ | 80 | 443番への301転送 |

## データ構造

物理テーブルは元の`orders` fact tableを維持します。TreeTable用の階層は、バックエンドがレスポンスとして組み立てます。

```text
customer
└── product
    └── order
```

`GET /api/v1/orders/tree`は、現在のページに含まれる注文を顧客、商品、注文の順に階層化して返します。ページングの単位は注文行です。そのため、同じ顧客が別のページにも現れることがあります。

## API

エンドポイントはRails流のversioned namespace（`/api/v1/...`）で公開しています。レスポンスは`{ <resource>, meta }`の二段構成です。

### `GET /api/v1/orders`

平坦な注文一覧を返すエンドポイントです。BigTable版と同じ形の`Order[]`を返します。

| パラメータ | 型 | 既定値 | 説明 |
| --- | --- | --- | --- |
| `page` | number | 1 | ページ番号（上限1,000,000） |
| `per_page` | number | 50 | 1ページあたりの注文件数（上限500） |
| `sort` | string | `id` | ソート対象のカラム（ホワイトリスト制） |
| `order` | `asc` / `desc` | `asc` | ソート方向 |
| `order_type` | string | なし | 種別フィルター（完全一致） |
| `status` | string | なし | ステータスフィルター（完全一致） |
| `customer_name` | string | なし | 顧客名の部分一致（LIKEエスケープ済み） |
| `product_name` | string | なし | 商品名の部分一致（LIKEエスケープ済み） |
| `date_from` | YYYY-MM-DD | なし | 注文日の開始日 |
| `date_to` | YYYY-MM-DD | なし | 注文日の終了日 |

レスポンスの例を示します。

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

> `unit_price`と`total_amount`のDecimalは、ActiveRecordの既定どおりBigDecimalから文字列へシリアライズしています。フロントエンドは`Number(...)`で数値に戻して表示します。

### `GET /api/v1/orders/tree`

TreeTable用の階層データを取得します。クエリパラメータは`/api/v1/orders`と同じです。

レスポンスの例を示します。

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

## OFFSET劣化対策 — deferred join

`Order.search`は、`offset >= 10_000`になるとdeferred joinへ自動的に切り替えます。SQLの形はBigTreeTable Go版の`BuildQuery`と同じです。

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

> MySQLは`WHERE id IN (SELECT id FROM ... LIMIT ... OFFSET ...)`の形を`LIMIT & IN/ALL/ANY/SOME subquery`の制約で拒否します。そのため、Arelの`Arel::Nodes::TableAlias`を使ったINNER JOINを採用しています。

実機のブラウザで100万件を検証した結果は、[`docs/verification/REPORT.md`](docs/verification/REPORT.md)にあります。最深ページ（offset = 999,975）でも応答は207 msでした。全展開後の仮想スクロールも平均~74 fpsで、30fpsを下回ったフレームは0件でした。

## フロントエンド構成

- `src/components/TreeTable.tsx` TreeTable画面の本体
- `src/components/TreeTableRow.tsx` 顧客、商品、注文の行の描画
- `src/components/TreeTableHeader.tsx` TreeTable用のヘッダー
- `src/hooks/useTreeTableData.ts` `/api/v1/orders/tree`の取得と状態管理
- `src/utils/treeData.ts` 展開状態を反映した可視ノードのflatten処理
- `src/components/VirtualScroller.tsx` 固定行高の仮想スクロール

## テスト

バックエンドはMinitestで検査し、SimpleCovで80%の下限を課します。

```bash
bin/rails test
# Coverage report generated for Minitest to coverage/
```

フロントエンドは次のコマンドで検査します。

```bash
cd frontend
pnpm exec tsc -b      # 型検査
pnpm exec vp fmt      # 整形
pnpm exec eslint .    # lint
pnpm test             # Vitest
```

E2Eは次のコマンドで実行します。

```bash
cd frontend
pnpm exec playwright test
```

## 統合品質ゲート

`bin/quality`を1回実行すると、6つのゲートを順に実行し、失敗した時点で終了します。

```bash
bin/quality
# 1. rubocop --no-color           (formatter + lint)
# 2. srb tc                       (Sorbet 静的型検査)
# 3. brakeman --quiet --exit-on-warn (Rails セキュリティスキャン)
# 4. bundler-audit check --update (CVE スキャン)
# 5. bin/rails test               (Minitest + SimpleCov 80%)
# 6. bin/rails assets:precompile  (本番アセットビルド)
```

`rake quality`でも同じゲートを実行します。個別のタスクは`quality:rubocop`や`quality:sorbet`などです。

## Gitleaks

pre-commitのhookは次のコマンドでインストールします。

```bash
pre-commit install
```

手動で実行するときは次のコマンドを使います。

```bash
pre-commit run --all-files
gitleaks git --redact --no-banner --verbose
gitleaks dir . --redact --no-banner --verbose
```

GitHub Actionsも、push、pull request、手動実行のたびにGitleaksを実行します（`.github/workflows/ci.yml`）。

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
├── compose.yml                 # 開発用: mysql + backend + frontend (Vite)
└── compose.production.yml      # 本番用: Nginx (TLS終端) + backend
```
