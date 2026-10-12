# big_tree_table

MySQL上の100万件の注文データを、ReactのTreeTableと仮想スクロールで表示するデモです。同じフロントエンドの設計を、GoとRuby on Railsの2つのバックエンドで実装しています。

## 構成

- [`go/`](go/) Go + MySQLのバックエンドと、React + TypeScriptのフロントエンドがあります。Rails版の移植元はこの実装です。
- [`rails/`](rails/) Rails 8.1 + ActiveRecordのバックエンドと、Go版から流用したReactのフロントエンドがあります。

起動方法、API、テストの手順は、各ディレクトリのREADMEで説明しています。どちらの手順も、そのディレクトリを起点にコマンドを実行します。

## CI

GitHub Actionsの[ci.yml](.github/workflows/ci.yml)は、pull request、mainへのpush、手動実行のたびに両方の実装を検証するワークフローです。Rails版ではrubocop、Sorbet、Brakeman、bundler-audit、Minitest、アセットのビルドが動き、Go版ではgo vet、go build、go testが動きます。型チェック、lint、Vitestは、2つのフロントエンドに共通の項目です。

Gitleaksはリポジトリ全体を走査し、設定は[.gitleaks.toml](.gitleaks.toml)にまとめてあります。pre-commit hookを使うときは、リポジトリのルートで`pre-commit install`を実行してください。

## 解説記事

2つのバックエンドの違いは、Zennの記事「[Reactに100万件の親子データを渡すと何が起きるのか - GoとRailsでバックエンドはこう変わる](https://zenn.dev/okamyuji/articles/react-tree-table-go-rails-virtual-scroll)」で解説しています。
