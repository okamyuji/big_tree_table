# BigTreeTable

100万件の受発注データを、顧客、商品、注文の階層で表示するTreeTableデモアプリケーションです。

このリポジトリは、Reactの大規模テーブルと仮想スクロールを説明するためのBigTableアプリを、折りたたみ可能なTreeTable対応版として拡張したものです。バックエンドはMySQL上の100万件データをソート、フィルター、ページネーションし、TreeTable専用APIで `customer -> product -> order` の階層レスポンスを返します。フロントエンドはReactとTypeScriptで構築し、展開状態を反映した可視ノードだけを独自の仮想スクロールへ渡して描画します。

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

MySQLをDocker Composeで起動します。既存のローカルMySQLと衝突しないよう、ホスト側は `3307` を使います。

```bash
docker compose up -d mysql
```

初回は100万件のダミーデータを投入します。

```bash
cd backend
DB_HOST=127.0.0.1 DB_PORT=3307 go run seed/seed.go
```

バックエンドを起動します。

```bash
cd backend
DB_HOST=127.0.0.1 DB_PORT=3307 go run main.go
```

フロントエンドを起動します。

```bash
cd frontend
vp dev
```

ブラウザで <http://localhost:5173> を開きます。

## Docker Compose

すべてのサービスをDockerで起動することもできます。

```bash
docker compose up -d
```

Docker Composeで起動した場合、フロントエンドは <http://localhost:3000> で配信されます。

## ポート

| サービス | ポート | 用途 |
| --- | --- | --- |
| MySQL | 3307 -> 3306 | ローカル開発用DB |
| Backend | 8080 | APIサーバー |
| Frontend Docker | 3000 | Nginx配信 |
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

レスポンス例:

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

バックエンド:

```bash
cd backend
go test ./...
```

フロントエンド:

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

pre-commit hookをインストールします。

```bash
pre-commit install
```

手動実行:

```bash
pre-commit run --all-files
gitleaks git --redact --no-banner --verbose
gitleaks dir . --redact --no-banner --verbose
```

GitHub Actionsでもpush、pull request、手動実行時にGitleaksを実行します。

## ディレクトリ構成

```text
BigTreeTable/
├── .github/
│   └── workflows/
│       └── gitleaks.yml
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
