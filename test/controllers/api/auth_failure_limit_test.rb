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

    # Holds every auth-failure counter access until all racers have arrived, so
    # concurrent requests touch the counter at the same moment. A timeout keeps
    # a racer that is never joined (e.g. one already answered) from hanging.
    class RacingStore < SimpleDelegator
      def initialize(store, barrier)
        super(store)
        @barrier = barrier
      end

      %i[read increment decrement].each do |name|
        define_method(name) do |key, *args, **opts|
          @barrier.wait(1) if key.to_s.start_with?("auth-failure")
          __getobj__.public_send(name, key, *args, **opts)
        end
      end
    end

    test "concurrent wrong guesses cannot exceed the failure limit" do
      racers = 5
      fail_times(client(37), LIMIT - 1)
      original = Api::BaseController.cache_store
      Api::BaseController.config.cache_store = RacingStore.new(original, Concurrent::CyclicBarrier.new(racers))

      statuses = Array.new(racers) do
        Thread.new do
          session = open_session
          credentials = ActionController::HttpAuthentication::Basic.encode_credentials(@user, @wrong)
          session.get "/api/v1/orders", headers: { "HTTP_AUTHORIZATION" => credentials }, env: { "REMOTE_ADDR" => client(37) }
          session.response.status
        end
      end.map(&:value)

      assert_equal [ 401, 429, 429, 429, 429 ], statuses.sort
    ensure
      Api::BaseController.config.cache_store = original
    end

    # Solid Cache's failsafe turns a transient DB error (deadlock, lost
    # connection) inside increment into a nil return instead of raising.
    class FailingIncrementStore < SimpleDelegator
      attr_reader :increments

      def initialize(store, failures)
        super(store)
        @failures = failures
        @increments = 0
      end

      def increment(key, *args, **opts)
        @increments += 1
        return nil if @increments <= @failures

        __getobj__.increment(key, *args, **opts)
      end
    end

    def with_store(store)
      original = Api::BaseController.cache_store
      Api::BaseController.config.cache_store = store
      yield
    ensure
      Api::BaseController.config.cache_store = original
    end

    test "when the counter cannot be updated, the request is refused even with the right password" do
      store = FailingIncrementStore.new(Api::BaseController.cache_store, 2)

      with_store(store) { call_api(client(38), @secret) }

      assert_response :too_many_requests
      assert_equal 2, store.increments
    end

    test "one transient counter failure is retried, so the right password still gets through" do
      store = FailingIncrementStore.new(Api::BaseController.cache_store, 1)

      with_store(store) { call_api(client(39), @secret) }

      assert_response :success
      assert_equal 2, store.increments
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
