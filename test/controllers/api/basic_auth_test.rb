require "test_helper"

module Api
  class BasicAuthTest < ActionDispatch::IntegrationTest
    USER = "viewer".freeze
    PASSWORD = SecureRandom.hex(16)

    setup do
      @saved_env = ENV.to_h.slice("API_BASIC_AUTH_USER", "API_BASIC_AUTH_PASSWORD")
      @saved_rails_env = Rails.env
    end

    teardown do
      ENV.delete("API_BASIC_AUTH_USER")
      ENV.delete("API_BASIC_AUTH_PASSWORD")
      ENV.update(@saved_env)
      Rails.env = @saved_rails_env
    end

    def configure_credentials
      ENV["API_BASIC_AUTH_USER"] = USER
      ENV["API_BASIC_AUTH_PASSWORD"] = PASSWORD
    end

    def auth_header(user, password)
      { "HTTP_AUTHORIZATION" => ActionController::HttpAuthentication::Basic.encode_credentials(user, password) }
    end

    test "credentials configured: correct credentials are accepted" do
      configure_credentials

      %w[/api/v1/orders /api/v1/orders/tree].each do |path|
        get path, headers: auth_header(USER, PASSWORD)

        assert_response :success
      end
    end

    test "credentials configured: missing Authorization gets 401 with a Basic challenge" do
      configure_credentials

      get "/api/v1/orders"

      assert_response :unauthorized
      assert_match(/\ABasic realm=/, response.headers["WWW-Authenticate"])
    end

    test "credentials configured: wrong user or password gets 401" do
      configure_credentials

      [ [ USER, "wrong" ], [ "wrong", PASSWORD ] ].each do |user, password|
        get "/api/v1/orders/tree", headers: auth_header(user, password)

        assert_response :unauthorized
      end
    end

    test "credentials unset in development/test: API is open" do
      ENV.delete("API_BASIC_AUTH_USER")
      ENV.delete("API_BASIC_AUTH_PASSWORD")

      get "/api/v1/orders"

      assert_response :success
    end

    test "credentials unset in production: every request is refused, even with a header" do
      ENV.delete("API_BASIC_AUTH_USER")
      ENV.delete("API_BASIC_AUTH_PASSWORD")
      Rails.env = "production"

      get "/api/v1/orders"

      assert_response :unauthorized

      get "/api/v1/orders", headers: auth_header("", "")

      assert_response :unauthorized
    end

    test "only one of the two variables set counts as unset" do
      ENV["API_BASIC_AUTH_USER"] = USER
      ENV.delete("API_BASIC_AUTH_PASSWORD")
      Rails.env = "production"

      get "/api/v1/orders", headers: auth_header(USER, "")

      assert_response :unauthorized
    end
  end
end
