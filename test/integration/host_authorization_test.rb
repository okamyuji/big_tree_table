# typed: false

require "test_helper"

# config/initializers/cors.rb が "backend" を config.hosts へ足すのは、Viteの
# プロキシ経由でリクエストが来る development 環境のためだけ。環境ガードなしに
# 足すと production で問題になる。production では Rails が config.hosts を
# 空のまま起動するので、初期化子が1件足すと許可リストが ["backend"] だけになり、
# 本番ドメインを含む他のすべてのHostが HostAuthorization に 403 で弾かれる。
class HostAuthorizationTest < ActionDispatch::IntegrationTest
  test "the development-only backend host is not injected outside development" do
    refute_predicate Rails.env, :development?, "この検査は development 以外の環境で走らせる前提"
    refute_includes Rails.application.config.hosts, "backend",
      "config.hosts に development 専用の 'backend' が入っている。" \
      "production では許可リストがこれだけになり、他のHostが全部403になる"
  end

  test "requests to an allowed host still succeed" do
    get api_v1_orders_url(per_page: 1), headers: { "Host" => "www.example.com" }

    assert_response :success
  end
end
