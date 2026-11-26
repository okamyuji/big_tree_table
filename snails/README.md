# snails版

Rails版の注文API（`GET /api/v1/orders`と`GET /api/v1/orders/tree`）を[snails](https://snails-rb.org/)で書き直したバックエンドです。snailsはRubyのコードをSpinelでネイティブのバイナリにコンパイルするフレームワークで、ActiveRecordの代わりに生のSQLでPostgreSQLを読みます。応答の形、パラメータの扱い、Basic認証と回数制限はRails版と同じにしてあり、一致は[`../bench/parity.sh`](../bench/)で確かめます。

## ファイル

- `app/models/order.rb` 検索条件の正規化、SQLの組み立て、ツリーの集計を受け持ちます。
- `app/controllers/api/v1/orders_controller.rb` 2つのエンドポイントと、認証と回数制限の入口です。
- `app/lib/api_guard.rb` 回数制限のカウンタと、定数時間の文字列比較をまとめています。
- `app/lib/money.rb` PostgreSQLが返す金額の文字列を銭単位の整数に直し、RailsのBigDecimalと同じ書式で出力します。
- `db/schema.sql` `../bench/db/schema.sql`の写しです。表はベンチ側で作るので、`pins.toml`の`DB_OWNER="external"`でsnailsのマイグレーションを止めています。

## ビルドと起動

snailsのCLIは`curl -fsSL https://raw.githubusercontent.com/snails-rb/snails/main/install.sh | bash`で入ります。macOSでは`brew install openssl jemalloc pkg-config`も要ります。

```bash
bin/build-binary
PG_URL=postgres://bench:bench@127.0.0.1:55432/big_tree_table_bench \
  API_BASIC_AUTH_USER=viewer API_BASIC_AUTH_PASSWORD=change-me \
  SNAILS_ENV=production PORT=3000 build/snails
```

`API_BASIC_AUTH_USER`と`API_BASIC_AUTH_PASSWORD`が未設定のとき、developmentとtest以外ではどのリクエストにも401を返します。全体の回数制限の既定は1分300回で、`API_RATE_LIMIT`で変えられるようにしてあります。リクエストごとのログが要らないときは`SNAILS_LOG_LEVEL=warn`を渡してください。

## テスト

`bin/test`はアプリをビルドし、実際のサーバーへHTTPでリクエストを送って確かめます。`PG_URL`には使い捨てのデータベースを渡してください。テストは`orders`表を作り直し、ケースごとに中身を消します。

```bash
PG_URL=postgres://bench:bench@127.0.0.1:55432/big_tree_table_snails_test bin/test
```
