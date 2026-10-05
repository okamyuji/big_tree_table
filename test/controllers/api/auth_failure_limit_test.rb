require "test_helper"

module Api
  class AuthFailureLimitTest < ActionDispatch::IntegrationTest
    LIMIT  = Api::BaseController::AUTH_FAILURE_LIMIT
    WINDOW = Api::BaseController::AUTH_FAILURE_WINDOW

    setup do
      @saved_env = ENV.to_h.slice("API_BASIC_AUTH_USER", "API_BASIC_AUTH_PASSWORD")
      @user = "viewer"
      @secret = SecureRandom.hex(16)
      @wrong = SecureRandom.hex(16)
      ENV["API_BASIC_AUTH_USER"] = @user
      ENV["API_BASIC_AUTH_PASSWORD"] = @secret
    end

    teardown do
      ENV.delete("API_BASIC_AUTH_USER")
      ENV.delete("API_BASIC_AUTH_PASSWORD")
      ENV.update(@saved_env)
    end

    def client(host)
      "203.0.113.#{host}"
    end

    def call_api(ip, secret)
      credentials = ActionController::HttpAuthentication::Basic.encode_credentials(@user, secret)
      get "/api/v1/orders", headers: { "HTTP_AUTHORIZATION" => credentials }, env: { "REMOTE_ADDR" => ip }
    end

    def fail_times(ip, count)
      count.times do
        call_api(ip, @wrong)

        assert_response :unauthorized
      end
    end

    test "the failure limit is far below the general limit" do
      assert_operator LIMIT, :<, Api::BaseController::RATE_LIMIT
    end

    test "after the failure limit, even the right password gets 429" do
      fail_times(client(30), LIMIT)

      call_api(client(30), @secret)

      assert_response :too_many_requests
    end

    test "one failure below the limit still lets the right password through" do
      fail_times(client(31), LIMIT - 1)

      call_api(client(31), @secret)

      assert_response :success
    end

    test "requests without credentials are challenged but not counted" do
      (LIMIT + 1).times do
        get "/api/v1/orders", env: { "REMOTE_ADDR" => client(32) }

        assert_response :unauthorized
      end

      call_api(client(32), @secret)

      assert_response :success
    end

    test "successful requests are not counted as failures" do
      (LIMIT + 1).times do
        call_api(client(33), @secret)

        assert_response :success
      end
    end

    test "the failure count is per client IP" do
      fail_times(client(34), LIMIT)

      call_api(client(35), @secret)

      assert_response :success
    end

    test "the failure count resets after the window" do
      fail_times(client(36), LIMIT)

      travel WINDOW + 1.second do
        call_api(client(36), @secret)

        assert_response :success
      end
    end
  end
end
