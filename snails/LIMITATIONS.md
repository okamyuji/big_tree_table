# snailsの限界（big_tree_tableの移植で確かめた範囲）

big_tree_tableのRails版API（`GET /api/v1/orders`と`GET /api/v1/orders/tree`）を、snails v1.3.0（Spinel 2669f3fec）へ移植しました。この文書はその作業で突き当たった制約をまとめたもので、出典はsnailsの公式マニュアルと`snails-core`のソース、それに移植中に実際に起きた事象です。

## 認証とセキュリティ

snailsには秘密値を定数時間で比べる関数がありません。マニュアル08章にも「Nothing in the framework stores sessions, rotates tokens, or compares secrets in constant time.」と書かれています。Railsの`ActiveSupport::SecurityUtils.secure_compare`にあたる処理は、移植で`app/lib/api_guard.rb`に自作しました。

Basic認証の補助は`authenticate_with_http_basic`と`request_http_basic_authentication`の2つだけです。401の本文は`HTTP Basic: Access denied.`という固定の文字列なので、JSONで返したいときは自分で組み立てる必要があります。

Railsの`rate_limit`にあたる宣言もありません。回数制限は`Snails.cache.increment`でカウンタを作って実装しました。

そのキャッシュには次の制約があります。

- プロセスごとに独立しています。`WORKERS`で複数プロセスにすると、カウンタはプロセスの数だけ別々に数えられます。
- 上限は5,000件です（`snails/cache.rb`の`MAX_ENTRIES = 5000`）。上限に達した状態で新しいキーを書くと、`flush_if_full`がすべての項目を消します。異なる送信元IPアドレスからの失敗が大量に来ると、全体の回数制限のカウンタまで一緒に消えてしまいます。

`before_action`は継承されません。`action_controller/api.rb`は、登録したクラス名と実行中のクラス名が一致するフィルタだけを動かします（`next unless entry[0] == self.class.name`）。親コントローラに認証を書いても子では動かないため、コントローラごとに宣言し直すしかありません。宣言を1つ忘れたコントローラは認証なしで公開されてしまいます。

`rescue_from`もありません。例外を1か所で扱うには`process_action`を上書きし、`super`を`rescue`で囲みます。

## データベース

使えるデータベースはSQLiteとPostgreSQLだけです。`snails new --database=`は`sqlite`と`postgresql`しか受け付けないので、MySQLで動くアプリはそのままでは移せません。このリポジトリでは比較のために、Rails版とGo版にもPostgreSQLで動く設定を足しました。

接続先は環境変数`PG_URL`からだけ読まれます。Railsの慣習である`DATABASE_URL`は読みません。

ActiveRecordはなく、モデルは`Sqlx::Base`を継承して生のSQLを書きます。`where`や`order`をつなげて検索を組み立てる書き方はできません。プレースホルダは`?`だけで、値はクライアント側で文字列に埋め込まれます。`$1`の形は使えません。

この埋め込み方には安全上の前提があります。`database/postgresql.rb`の`escape_string`は、値の中の単一引用符を重ねるだけです。この方法が安全なのはPostgreSQLの`standard_conforming_strings`がon（9.1以降の既定）のときに限られます。offのサーバーではバックスラッシュが引用符を打ち消すので、検索語からSQLを書き換えられるおそれがあります。アダプタは接続時にこの設定をしません。移植では`config/environment.rb`で`PG_URL`にlibpqの`options=-c standard_conforming_strings=on`を足し、接続ごとにonにしました。

値にNUL文字（文字コード0）が入ると、埋め込んだSQLがlibpqに渡る時点でNULの位置で切れます。閉じていない引用符で終わるため構文エラーになり、サーバーは500を返しました。SQLの書き換えにはなりません。PostgreSQLはtext型にNUL文字を入れられないので、RailsとGoでも同じく500でした。移植では3つの実装とも認証の後で400を返すようにそろえています。

生のSQLの結果では型の変換がRailsと異なります。

| PostgreSQLの型 | snailsが返す値 | 移植での対処 |
|---|---|---|
| NUMERIC | Float（`101.00`が`101.0`になり、末尾の0が落ちる） | SQLで`::text`にし、Rubyで銭単位の整数に直す |
| BOOL | 1と0 | ordersの表にはない |
| DATE | `YYYY-MM-DD`の文字列 | そのまま使う |
| TIMESTAMP | PostgreSQLの表記（`2026-11-01 00:00:00`） | SQLの`to_char`でISO 8601に直す |

BigDecimalもないので、金額の合計は銭単位の整数で計算しました。

PostgreSQLではアプリ側でseedを流す`seed_from_file`が使えず、呼ぶと「unsupported in the PG adapter」という例外になります。データの投入はpsqlで行いました。`Sqlx::Database.exec`は1回に1文しか実行できません。

## 言語（SpinelによるRubyのサブセット）

snailsはRubyのコードをSpinelでネイティブのバイナリにコンパイルします。そのため、使えるRubyはサブセットに限られます。

- gem、Bundler、ネイティブ拡張は使えません。
- 文字列リテラルは凍結されています。`out = "%"`に`<<`で追記すると、ビルド時に警告が出て、実行時にFrozenErrorになります。`String.new("%")`のように可変の文字列を作る必要がありました。
- 整数は64ビットで、Bignumがありません。20桁の`page`パラメータに`to_i`を使うと、サーバーが500を返しました。移植では、桁数が多いときは上限に丸めています。
- 配列の要素は同じ型にそろえる必要があります。
- ブロックを変数に保存したり、`instance_exec`で実行したりはできません。
- `require`は静的に解決され、autoloadはありません。

## ルーティング、テスト、開発環境

ルートには`namespace`や`scope`がないので、`/api/v1/orders/tree`のようにパスを丸ごと書きます。

開発サーバーはファイルの変更を監視しません。コードを変えるたびにビルドし直す必要があり、このアプリでは1回に20秒ほどかかりました。

macOSではビルドし直した`build/snails`が何も出力せずに`Killed: 9`で止まったことがありました。同じパスのバイナリを上書きしたため、macOSが古い署名の情報で実行を止めたものです。`codesign -f -s - build/snails`で署名をかけ直すと起動しました。

テストは実際に起動したサーバーへHTTPで送る結合テストだけで、モデル単体のテストを書く層はありません。1つのテストファイルの中では全ケースが同じサーバーを使うため、送信元IPアドレスを締め出す回数制限のテストを入れると後のケースがすべて429になります。この確認は3実装の応答を比べる`bench/parity.sh`で行います。

ログレベルを環境変数で変える仕組みもありません。`config/environment.rb`に`Snails.logger.level`を書く必要があり、ベンチ用の`SNAILS_LOG_LEVEL`は移植で足したものです。

## 導入と文書

導入にはインストールスクリプトを`curl | bash`で実行する方法と、GitHub Releasesからコンパイラを取得する方法を使います。

マニュアルとコードには食い違いがあります。マニュアル16章は`WORKERS`を「互換性のために受け付けるだけで、サーバーは1プロセスで動く」と説明しています。ところが`snails/main/server.rb`の実装は、`WORKERS`の値の数だけプロセスをpreforkするものでした。ベンチでは1プロセスで動かし、OSのworkerスレッドで全コアを使う既定の動作のまま測りました。

## 性能（Rails版、Go版との比較）

2026-11-27 14:50 JSTに、Rails版、Go版、snails版を同じ条件で測りました。3者は同じPostgreSQLの100万行を読み、同じBasic認証と回数制限を通って同じJSONを返します。詳しい条件と全数値は[`bench/RESULTS.md`](../bench/RESULTS.md)にあります。

- マシン Apple Silicon Mac（12コア）です。PostgreSQL 16はcolimaのDocker VM（8コア）で動かしました。
- 負荷 k6 2.3.0で同時接続32、計測30秒です。3者は1つずつ動かしました。
- 起動条件 Railsは12プロセス×5スレッド、Goとsnailsはどちらも1プロセスです。DB接続は3者とも60本です。

200以外の応答は全シナリオで0件でした。RSSは計測直後の使用メモリで、Railsは12プロセスの合計です。

| シナリオ | 実装 | req/s | p50 ms | p99 ms | RSS MiB |
|---|---|---:|---:|---:|---:|
| light | Go | 3175 | 9.5 | 20.9 | 35 |
| light | Rails | 2034 | 15.1 | 29.3 | 1448 |
| light | snails | 1778 | 18.6 | 26.8 | 87 |
| list | Go | 234 | 151.2 | 204.1 | 29 |
| list | Rails | 228 | 154.7 | 208.9 | 1425 |
| list | snails | 193 | 163.1 | 300.5 | 96 |
| filter | Go | 175 | 181.1 | 250.5 | 28 |
| filter | Rails | 170 | 186.6 | 249.2 | 1312 |
| filter | snails | 133 | 261.1 | 360.2 | 101 |
| deep | Go | 24 | 1225.9 | 3219.0 | 27 |
| deep | Rails | 23 | 1370.2 | 2905.6 | 1423 |
| deep | snails | 23 | 1374.8 | 2404.9 | 96 |
| tree | Go | 14 | 1672.8 | 4380.4 | 29 |
| tree | Rails | 13 | 1747.0 | 4370.6 | 1316 |
| tree | snails | 10 | 2685.1 | 8785.7 | 361 |

アプリ側の速さはlightで読みます。lightは注文日を1日に絞った一覧で、DBの処理は約2.6msで済みます。snailsはGoの56%、Railsの87%の件数でした。このAPIではネイティブバイナリのsnailsがRailsより速くなることはありませんでした。p99は26.8msで、Railsの29.3msより短くなっています。

残りの4つはDBが上限です。全件の`COUNT(*)`は1回14.3msかかり、本体と並列ワーカー2つの3プロセスで数えます。それでもsnailsはlistでGoより18%、filterで24%少ない件数でした。treeではp99が8,785.7msで、GoとRailsの約2.0倍です。500行を顧客、商品、注文の3段に集計する処理で、snailsは遅れが大きく出ました。

snailsの強みはメモリです。使用量は87〜361MiBでした。同じシナリオのRails（12プロセスの合計）と比べると、lightで6%、treeで27%に収まります。Goの27〜35MiBよりは大きくなります。

## まとめ

snailsを使えばRailsに近い書き方のアプリを、gemに頼らない小さなネイティブバイナリにできます。代わりに、認証の部品、回数制限、型の変換、テストの層は、Railsより自分で書く部分が多くなります。移植で最も手間がかかったのは値の表記をRailsと完全に一致させる作業で、最も危険なのは`before_action`が継承されないことによる認証の漏れです。性能ではRailsよりメモリを大きく減らせましたが、処理件数はRailsを下回りました。
