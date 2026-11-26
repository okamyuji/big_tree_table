# bench

Rails版、Go版、snails版の注文APIを、同じPostgreSQLと同じデータで比べるためのスクリプトを置いています。3つの実装は`/api/v1/orders`と`/api/v1/orders/tree`で同じ応答を返し、認証と回数制限の手順もRails版の`Api::BaseController`にそろえてあります。

## 比べる処理

どの実装も1リクエストごとに次の処理を行います。

- 全体の回数制限のカウンタを1つ増やします。既定は1分300回で、ベンチでは`API_RATE_LIMIT`で上限を上げ、カウンタの処理だけを残します。
- Basic認証を定数時間で比較し、IPごとの失敗回数カウンタを増やしてから、成功時に1つ戻します。
- 件数を`COUNT(*)`で数え、ページの行を取得します。OFFSETが10,000以上のときは、主キーだけを先に絞るdeferred joinに切り替えます。
- 一覧はJSONに整え、ツリーは顧客、商品、注文の順にまとめて金額を集計します。

カウンタは3者ともプロセス内のメモリに置きます。Rails版はベンチ用の`RAILS_ENV=bench`で`memory_store`を使います。

## 準備

PostgreSQLを1つ起動し、ベンチ用とparity用のデータベースを作ります。

```bash
docker run -d --name btt-bench-pg -e POSTGRES_USER=bench -e POSTGRES_PASSWORD=bench \
  -e POSTGRES_DB=big_tree_table_bench -p 127.0.0.1:55432:5432 postgres:16-alpine
psql postgres://bench:bench@127.0.0.1:55432/big_tree_table_bench -c 'CREATE DATABASE big_tree_table_parity'
PG_URL=postgres://bench:bench@127.0.0.1:55432/big_tree_table_bench bench/db/setup.sh
```

`bench/db/setup.sh`は`db/schema.sql`で表を作り直してから`db/seed.sql`で行を投入します。seedの式は`rails/db/seeds.rb`と同じで、日時を固定しているので何度流しても同じデータになります。行数は引数で変えられ、既定は1,000,000行です。

各実装のビルドも先に済ませておきます。snails版は`snails/`で`bin/build-binary`を実行し、Rails版は`rails/`で`bundle install`を実行します。Go版は`bench/lib/servers.sh`が起動のたびにビルドします。負荷はk6で掛けるので、`mise install k6@2.3.0`で入れておきます。

## 応答の一致を確かめる

`bench/parity.sh`はRails版の応答を正として、Go版とsnails版の応答を比べます。比べる対象はステータス、`X-Total-Count`、`WWW-Authenticate`、JSON本文です。

```bash
PG_URL=postgres://bench:bench@127.0.0.1:55432/big_tree_table_parity bench/parity.sh
```

このスクリプトは指定したデータベースの`orders`表を作り直します。3万行のseedに`db/parity_extra.sql`の行を加え、端数のある金額と、LIKEの特殊文字や引用符や`?`を含む名前を検査します。ベンチ用のデータベースは指定しないでください。

## ベンチを実行する

`bench/run.sh`は実装を1つずつ起動して計測します。3つを同時には動かしません。1つの実装について起動、シナリオごとのウォームアップと計測、停止を行い、クールダウンを置いてから次の実装へ進みます。

```bash
PG_URL=postgres://bench:bench@127.0.0.1:55432/big_tree_table_bench bench/run.sh
```

シナリオは`k6/api.js`に定義しています。

| シナリオ | 内容 |
|---|---|
| list | OFFSETが10,000未満の一覧 |
| deep | OFFSETが100,000以上の一覧で、deferred joinを通ります |
| tree | 500件をツリーにまとめる応答 |
| filter | LIKE、等値、日付範囲を組み合わせた絞り込み |

条件は環境変数で変えられます。既定値は`run.sh`の冒頭に書いてあります。RailsはGVLのため1プロセスで1コアしか使えないので、`RAILS_WORKERS`の既定値をCPUのコア数にしています。Go版とsnails版は1プロセスで全コアを使うため、1プロセスのまま動かします。DBの接続数は、3者とも`RAILS_WORKERS × THREADS`にそろえてあります。

結果は`bench/results/<日時>/`に保存され、`summary.md`に表がまとまります。サーバーのRSSは`ps`で測るので、`ps`を使えない環境では空欄になります。

## 測定上の制約

負荷を掛けるk6、測られるサーバー、PostgreSQLは同じマシンで動きます。macOSにはCPUコアをプロセスごとに割り当てる仕組みがないため、k6とサーバーはコアを取り合います。PostgreSQLはDockerのVM内で動くので、VMに割り当てたCPU数が、データベースに重いシナリオの上限を決めます。結果を読むときは、`env.txt`に残した測定環境と合わせて確認してください。
