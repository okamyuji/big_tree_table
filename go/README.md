# BigTable

100万行の受発注データを瞬時に表示できる高性能テーブルアプリケーションです。

Goの標準ライブラリとMySQLで構成されたバックエンドが、ソート、フィルター、ページネーションの処理を担当します。フロントエンドはReactとTypeScriptで構築されており、カスタムの仮想スクロールによって効率的に描画します。ソートを変更した際にはサーバーへ新しいソート順を送り、現在のページに該当するデータだけを再取得します。

## 技術構成

バックエンドはGoの標準ライブラリとgo-sql-driver/mysqlだけで構成されています。データベースにはDocker上のMySQL 8を使用しています。

フロントエンドはVite+(vpコマンド)でビルドしています。React 19、TypeScript、TailwindCSS v4を採用しており、テストにはVitestを使用しています。

E2EテストにはPlaywrightを使用しています。

インフラはDocker Composeで管理しており、MySQL、バックエンド、フロントエンドの3つのサービスをまとめて起動できます。

## 起動方法

Docker Composeを使って、すべてのサービスをまとめて起動できます。

```bash
docker compose up -d
```

起動後はブラウザで http://localhost:3000 にアクセスしてください。

初回起動時はデータベースが空の状態です。100万行のダミーデータを投入するには、以下のコマンドを実行してください。

```bash
cd backend
go run seed/seed.go
```

### 個別に起動する場合

MySQLだけをDockerで起動し、バックエンドとフロントエンドはローカルで動かすこともできます。

```bash
docker compose up -d mysql
```

バックエンドを起動します。ポート8080で待ち受けます。

```bash
cd backend
go run main.go
```

フロントエンドを起動します。ポート5173で待ち受けます。APIリクエストはバックエンドのポート8080へ自動的にプロキシされます。

```bash
cd frontend
vp dev
```

## ポート一覧

| サービス | ポート | 用途 |
|---|---|---|
| MySQL | 3306 | データベース |
| バックエンド | 8080 | APIサーバー |
| フロントエンド(Docker) | 3000 | Nginx経由の配信とAPIプロキシ |
| フロントエンド(ローカル) | 5173 | Vite開発サーバー |

## データベース

### ordersテーブル

受発注データを格納するテーブルです。

| カラム名 | 型 | 説明 |
|---|---|---|
| id | BIGINT AUTO_INCREMENT | 主キー |
| order_number | VARCHAR(20) UNIQUE | 注文番号。ORD-0000001の形式です |
| order_type | VARCHAR(10) | 受注または発注のいずれかです |
| order_date | DATE | 注文が作成された日付です |
| customer_name | VARCHAR(100) | 顧客の会社名です |
| customer_code | VARCHAR(20) | 顧客を一意に識別するコードです |
| product_name | VARCHAR(100) | 商品名です |
| product_code | VARCHAR(20) | 商品を一意に識別するコードです |
| quantity | INT | 注文数量です |
| unit_price | DECIMAL(12,2) | 1個あたりの単価です |
| total_amount | DECIMAL(14,2) | 数量と単価を掛けた合計金額です |
| status | VARCHAR(20) | 受注確認、出荷準備中、出荷済み、納品完了、キャンセルのいずれかです |
| delivery_date | DATE | 納品予定日です |
| notes | TEXT | 備考欄です |
| created_at | DATETIME | レコードが作成された日時です |
| updated_at | DATETIME | レコードが最後に更新された日時です |

検索の高速化のため、order_date、status、customer_code、product_code、order_type、delivery_dateの各カラムにインデックスを設定しています。

## API仕様

### GET /api/orders

受発注データを取得するエンドポイントです。ページネーション、ソート、フィルターのすべてをサーバー側で処理します。

| パラメータ | 型 | 初期値 | 説明 |
|---|---|---|---|
| page | 整数 | 1 | 取得するページの番号です |
| per_page | 整数 | 50 | 1ページあたりの件数です。10、25、50、100から選べます |
| sort | 文字列 | id | ソート対象のカラム名です |
| order | 文字列 | asc | ascで昇順、descで降順になります |
| order_type | 文字列 | なし | 種別で絞り込みます |
| status | 文字列 | なし | ステータスで絞り込みます |
| customer_name | 文字列 | なし | 顧客名の部分一致で絞り込みます |
| product_name | 文字列 | なし | 商品名の部分一致で絞り込みます |
| date_from | 文字列 | なし | 注文日の開始日をYYYY-MM-DD形式で指定します |
| date_to | 文字列 | なし | 注文日の終了日をYYYY-MM-DD形式で指定します |

レスポンスは以下の形式で返ります。

```json
{
  "data": [],
  "total": 1000000,
  "page": 1,
  "per_page": 50,
  "total_pages": 20000
}
```

ソートやフィルターの変更時には、MySQLのORDER BYとLIMIT/OFFSETがインデックスを活用して実行されます。100万行でもレスポンスは高速です。

## バックエンドの構成

バックエンドはRepository、Service、Handlerの3層にインターフェース経由で分離しています。

repositoryパッケージはデータベースへのアクセスを担当します。OrderRepositoryインターフェースを定義しており、MySQLの実装はMySQLOrderRepositoryとして提供しています。

serviceパッケージはビジネスロジックを担当します。OrderServiceインターフェースを定義しており、リポジトリを注入して使用します。

handlerパッケージはHTTPリクエストの解析とレスポンスの構築を担当します。サービスを注入して使用します。

main.goでは、リポジトリ、サービス、ハンドラの順に生成してつなぎ合わせています。

## フロントエンドの構成

仮想スクロールはサードパーティのライブラリに依存せず、独自に実装しています。useVirtualScrollフックが表示範囲を計算し、VirtualScrollerコンポーネントが可視領域の行だけをDOMに描画します。

useTableDataフックがデータの取得と状態管理を一括して担当します。ソート、フィルター、ページの変更時にはAbortControllerで前のリクエストをキャンセルしてから新しいリクエストを送信します。

## テストの実行方法

バックエンドのユニットテストを実行します。

```bash
cd backend
go test ./... -v
```

フロントエンドのユニットテストを実行します。

```bash
cd frontend
vp test -- --run
```

フロントエンドのlint、フォーマット、型チェックをまとめて実行します。

```bash
cd frontend
vp check
```

E2Eテストを実行します。事前にバックエンドとフロントエンドを起動しておく必要があります。

```bash
cd frontend
npx playwright test
```

## ディレクトリ構成

```
BigTable/
├── compose.yml
├── backend/
│   ├── Dockerfile
│   ├── main.go
│   ├── db/
│   │   └── connection.go
│   ├── handler/
│   │   ├── orders.go
│   │   └── orders_test.go
│   ├── middleware/
│   │   └── cors.go
│   ├── model/
│   │   ├── order.go
│   │   └── order_test.go
│   ├── repository/
│   │   ├── order_repository.go
│   │   └── order_repository_test.go
│   ├── seed/
│   │   └── seed.go
│   └── service/
│       ├── order_service.go
│       └── order_service_test.go
└── frontend/
    ├── Dockerfile
    ├── nginx.conf
    ├── vite.config.ts
    ├── playwright.config.ts
    ├── src/
    │   ├── App.tsx
    │   ├── main.tsx
    │   ├── api/
    │   │   └── orders.ts
    │   ├── components/
    │   │   ├── BigTable.tsx
    │   │   ├── ColumnFilter.tsx
    │   │   ├── DateRangeFilter.tsx
    │   │   ├── Pagination.tsx
    │   │   ├── TableHeader.tsx
    │   │   ├── TableRow.tsx
    │   │   ├── VirtualScroller.tsx
    │   │   └── columns.ts
    │   ├── hooks/
    │   │   ├── useTableData.ts
    │   │   └── useVirtualScroll.ts
    │   └── types/
    │       └── order.ts
    ├── tests/
    │   ├── setup.ts
    │   ├── components/
    │   │   ├── BigTable.test.tsx
    │   │   └── Pagination.test.tsx
    │   └── hooks/
    │       ├── useTableData.test.ts
    │       └── useVirtualScroll.test.ts
    └── e2e/
        └── bigtable.spec.ts
```
