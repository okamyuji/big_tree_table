# BigTreeTable

100万件の受発注データを、顧客、商品、注文の階層で表示するTreeTableデモアプリケーションです。

このディレクトリは、Reactの大規模テーブルと仮想スクロールを説明するための[BigTableアプリ](https://github.com/okamyuji/BigTable)を、折りたたみ可能なTreeTable対応版として拡張したものです。バックエンドはMySQL上の100万件データをソート、フィルター、ページネーションし、TreeTable専用APIで `customer -> product -> order` の階層レスポンスを返します。フロントエンドはReactとTypeScriptで構築し、展開状態を反映した可視ノードだけを独自の仮想スクロールへ渡して描画します。

## 主な機能

- 100万件の受発注データをMySQLにseed
- 顧客、商品、注文の3階層TreeTable表示
- 親行の展開、折りたたみ、すべて展開、すべて折りたたみ
- サーバーサイドのソート、フィルター、ページネーション
- 固定行高の独自仮想スクロール
- PlaywrightによるブラウザE2E
- pre-commitとGitHub ActionsでのGitleaks secret scan

## 技術構成

| 領域 | 技術 |
| --- | --- |
| Backend | Go標準ライブラリ、go-sql-driver/mysql |
| Database | MySQL 8 |
| Frontend | React 19、TypeScript、Tailwind CSS v4、Vite+ |
| Unit test | Go test、Vitest |
| E2E | Playwright |
| Security scan | Gitleaks、pre-commit、GitHub Actions |

## 起動方法

この節のコマンドは、どれも`go/`ディレクトリを起点にしています。Docker Composeは、MySQLのアプリ用ユーザー`bigtable`のパスワードを環境変数`DB_PASS`から読み、未設定なら起動前にエラーで止まります。初回は`.env`にランダムな値を書き、その値をシェルに読み込んでからMySQLを起動してください。`.env`はGitの管理対象外で、composeも自動で読み込みます。既存のローカルMySQLと衝突しないよう、ホスト側は `3307` を使います。

```bash
[ -f .env ] || (umask 077; echo "DB_PASS=$(openssl rand -hex 16)" > .env)
set -a; . ./.env; set +a
docker compose up -d --wait mysql
```

MySQLのデータは、初回起動時の`DB_PASS`で初期化されます。`.env`を消したり値を変えたりしたときは、`docker compose down -v`でボリュームを作り直してください。

初回は100万件のダミーデータを投入します。

```bash
(cd backend && DB_HOST=127.0.0.1 DB_PORT=3307 DB_USER=bigtable go run seed/seed.go)
```

途中で止まった投入の残りがあると、シードは行を消さずにエラーで終了します。消して入れ直したい場合は、同じコマンドの先頭に`SEED_RESET=1`を付けてください。

バックエンドを起動します。

```bash
(cd backend && DB_HOST=127.0.0.1 DB_PORT=3307 DB_USER=bigtable go run main.go)
```

フロントエンドは、別のターミナルで`go/`に移ってから起動します。フロントエンドは`DB_PASS`を使わないので、`.env`の読み込みは要りません。

```bash
(cd frontend && vp dev)
```

ブラウザで <http://localhost:5173> を開きます。

## Docker Compose

すべてのサービスをDockerで起動することもできます。`DB_PASS`は、前の節と同じく`.env`から読み込まれます。

```bash
[ -f .env ] || (umask 077; echo "DB_PASS=$(openssl rand -hex 16)" > .env)
docker compose up -d
```

初回の起動では、`seed`サービスが100万件を投入し終えてからバックエンドが起動します。投入済みのときは、シードがすぐに終わるので待ち時間はありません。途中で止まった投入の残りを消して入れ直す方法は、`SEED_RESET=1 docker compose up -d`の実行です。

Docker Composeで起動した場合、フロントエンドの配信先は <http://localhost:3000> です。composeは、どのポートも`127.0.0.1`に限定して公開します。バックエンドとNginxのコンテナを動かすのは、rootではない専用のユーザーです。

## ポート

| サービス | ポート | 用途 |
| --- | --- | --- |
| MySQL | 3307 -> 3306 | ローカル開発用DB |
| Backend | 8080 | APIサーバー |
| Frontend Docker | 3000 -> 8080 | Nginx配信 |
| Frontend local | 5173 | Vite開発サーバー |

## データ構造

物理テーブルは元の `orders` fact tableを維持します。TreeTable用の階層はバックエンドでレスポンスとして構築します。

```text
customer
└── product
    └── order
```

`GET /api/order-tree` は現在ページに含まれる注文を、顧客、商品、注文の順に階層化して返します。ページング単位は注文行です。そのため同じ顧客が別ページにも現れることがあります。

## API

### GET /api/order-tree

TreeTable用の階層データを取得します。ソート、フィルター、ページネーションはサーバー側で処理されます。

| パラメータ | 型 | 初期値 | 説明 |
| --- | --- | --- | --- |
| page | number | 1 | ページ番号 |
| per_page | number | 50 | 1ページあたりの注文件数 |
| sort | string | id | ソート対象カラム |
| order | asc / desc | asc | ソート方向 |
| order_type | string | なし | 種別フィルター |
| status | string | なし | ステータスフィルター |
| customer_name | string | なし | 顧客名の部分一致 |
| product_name | string | なし | 商品名の部分一致 |
| date_from | YYYY-MM-DD | なし | 注文日の開始日 |
| date_to | YYYY-MM-DD | なし | 注文日の終了日 |

レスポンスの例は次のとおりです。

```json
{
  "data": [
    {
      "id": "customer:CST-001",
      "kind": "customer",
      "depth": 0,
      "label": "田中製作所",
      "summary": {
        "order_count": 2,
        "quantity": 108,
        "total_amount": 4108910.16,
        "statuses": ["キャンセル", "出荷準備中"]
      },
      "children": []
    }
  ],
  "total": 1000000,
  "page": 1,
  "per_page": 25,
  "total_pages": 40000
}
```

### GET /api/orders

既存互換の平坦な注文一覧APIです。BigTable版と同じ形式の `Order[]` を返します。

## フロントエンド構成

- `TreeTable.tsx`: TreeTable画面本体
- `TreeTableRow.tsx`: 顧客、商品、注文行の描画
- `TreeTableHeader.tsx`: TreeTable用ヘッダー
- `useTreeTableData.ts`: `/api/order-tree` の取得と状態管理
- `treeData.ts`: 展開状態を反映した可視ノードのflatten処理
- `VirtualScroller.tsx`: 固定行高の仮想スクロール

## テスト

バックエンドのテストは次のコマンドで実行します。

```bash
cd backend
go test ./...
```

フロントエンドは、次のコマンドで整形、lint、テスト、型チェック、ビルドを順に実行します。

```bash
cd frontend
vp fmt
eslint .
vp test -- --run
tsc -b
vp build
```

E2E:

```bash
cd frontend
playwright test
```

E2Eでは以下を確認します。

- 100万件データの初期表示
- 初期描画時間とTree API応答時間
- 展開後の仮想スクロールでブランク行が出ないこと
- ソート後に行が戻ること
- ページング後に行が戻ること
- ブラウザconsole error/warnがないこと

## Gitleaks

pre-commitの設定はリポジトリのルートにあり、`rails/`と共有しています。pre-commit hookは、リポジトリのルートでインストールします。

```bash
pre-commit install
```

手動で実行するときは、リポジトリのルートで次のコマンドを使います。

```bash
pre-commit run --all-files
gitleaks git --redact --no-banner --verbose
gitleaks dir . --redact --no-banner --verbose
```

GitHub Actionsの[ci.yml](../.github/workflows/ci.yml)も、push、pull request、手動実行のたびにGitleaksを実行します。同じワークフローで、`backend/`にはgo vet、go build、go testを、`frontend/`には型チェック、lint、Vitestをかけています。

## ディレクトリ構成

```text
go/
├── backend/
│   ├── handler/
│   ├── model/
│   ├── repository/
│   ├── seed/
│   └── service/
├── docs/
│   └── tree-table-design.md
├── frontend/
│   ├── e2e/
│   ├── src/
│   │   ├── api/
│   │   ├── components/
│   │   ├── hooks/
│   │   ├── types/
│   │   └── utils/
│   └── tests/
└── compose.yml
```
