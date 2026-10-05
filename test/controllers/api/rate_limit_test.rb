require "test_helper"

module Api
  class RateLimitTest < ActionDispatch::IntegrationTest
    LIMIT  = Api::BaseController::RATE_LIMIT
    WINDOW = Api::BaseController::RATE_LIMIT_WINDOW

    setup do
      @saved_env = ENV.to_h.slice("API_BASIC_AUTH_USER", "API_BASIC_AUTH_PASSWORD")
      @user = "viewer"
      @secret = SecureRandom.hex(16)
      ENV["API_BASIC_AUTH_USER"] = @user
      ENV["API_BASIC_AUTH_PASSWORD"] = @secret
    end

    teardown do
      ENV.delete("API_BASIC_AUTH_USER")
      ENV.delete("API_BASIC_AUTH_PASSWORD")
      ENV.update(@saved_env)
    end

    def call_api(ip, secret)
      credentials = ActionController::HttpAuthentication::Basic.encode_credentials(@user, secret)
      get "/api/v1/orders", headers: { "HTTP_AUTHORIZATION" => credentials }, env: { "REMOTE_ADDR" => ip }
    end

    def exhaust_with_wrong_password(ip)
      LIMIT.times do
        call_api(ip, "wrong")

        assert_response :unauthorized
      end
    end

    test "failed attempts count toward the limit, so the next request gets 429 even with the right password" do
      exhaust_with_wrong_password("203.0.113.10")

      call_api("203.0.113.10", @secret)

      assert_response :too_many_requests
    end

    test "the limit is per client IP" do
      exhaust_with_wrong_password("203.0.113.11")

      call_api("203.0.113.12", @secret)

      assert_response :success
    end

    test "behind a private-network proxy, clients are keyed by X-Forwarded-For, not by the proxy address" do
      proxied = lambda do |client, secret|
        credentials = ActionController::HttpAuthentication::Basic.encode_credentials(@user, secret)
        get "/api/v1/orders",
            headers: { "HTTP_AUTHORIZATION" => credentials, "HTTP_X_FORWARDED_FOR" => client },
            env: { "REMOTE_ADDR" => "172.18.0.4" }
      end

      LIMIT.times { proxied.call("198.51.100.1", "wrong") }
      proxied.call("198.51.100.1", @secret)

      assert_response :too_many_requests

      proxied.call("198.51.100.2", @secret)

      assert_response :success
    end

    test "the counter key is one API-wide scope, not per controller" do
      call_api("203.0.113.20", @secret)

      assert Rails.cache.exist?("rate-limit:api:203.0.113.20")
      assert_not Rails.cache.exist?("rate-limit:api/v1/orders:203.0.113.20")
    end

    test "the count resets after the window" do
      exhaust_with_wrong_password("203.0.113.13")

      travel WINDOW + 1.second do
        call_api("203.0.113.13", @secret)

        assert_response :success
      end
    end
  end
end
