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
PG_URL=postgres://bench@127.0.0.1:55432/big_tree_table_bench \
  API_BASIC_AUTH_USER=viewer API_BASIC_AUTH_PASSWORD="$(openssl rand -hex 16)" \
  SNAILS_ENV=production PORT=3000 build/snails
```

`API_BASIC_AUTH_USER`と`API_BASIC_AUTH_PASSWORD`が未設定のとき、developmentとtest以外ではどのリクエストにも401を返します。認証失敗の回数はTCPの接続元アドレスごとに数えるので、APIは直接公開する前提です。リバースプロキシの後ろに置くと全員が1つの接続元として数えられ、誰か1人が10回間違えると全員が15分間429になります。

snailsのPostgreSQLアダプタは、値をSQLの文字列に埋め込むときに単一引用符を重ねるだけです。この方法は`standard_conforming_strings`がonのときにしか安全ではないため、そこで`PG_URL`にこの設定をonにするlibpqの`options`を足すのが、`config/environment.rb`の役目です。libpqは`options`を1つしか持てないため、`PG_URL`が別の`options`を指定しているときは、`-c standard_conforming_strings=on`を含めるよう求めて起動を止めます。

全体の回数制限の既定は1分300回で、`API_RATE_LIMIT`で変えられるようにしてあります。リクエストごとのログが要らないときは`SNAILS_LOG_LEVEL=warn`を渡してください。

macOSでビルドし直した直後に`build/snails`が何も出力せず`Killed: 9`で止まる場合は、`codesign -f -s - build/snails`で署名をかけ直してください。同じパスのバイナリを上書きしたときに、macOSが古い署名の情報で実行を止めることがあるためです。

## テスト

`bin/test`はアプリをビルドし、実際のサーバーへHTTPでリクエストを送って確かめます。`PG_URL`には使い捨てのデータベースを渡してください。テストは`orders`表を作り直し、ケースごとに中身を消します。テスト用のデータベースでは`standard_conforming_strings`をoffにしておくのがよい方法です。`config/environment.rb`が接続ごとにonへ戻していることを、バックスラッシュを含む検索のテストで同時に確かめられます。

```bash
psql postgres://bench@127.0.0.1:55432/big_tree_table_bench -c 'CREATE DATABASE big_tree_table_snails_test'
psql postgres://bench@127.0.0.1:55432/big_tree_table_bench \
  -c 'ALTER DATABASE big_tree_table_snails_test SET standard_conforming_strings = off'
PG_URL=postgres://bench@127.0.0.1:55432/big_tree_table_snails_test bin/test
```
