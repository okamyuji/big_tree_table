# Locks this client's address out for the rest of the run, so the file name
# sorts it after every other case that calls the API.
require_relative "../boot"

if Snails::Harness.file_selected?
  class AuthFailureLimitTest < Snails::Harness::IntegrationTest
    RIGHT = { "Authorization" => "Basic #{Base64.strict_encode64("harness:not-a-real-password")}" }.freeze
    WRONG = { "Authorization" => "Basic #{Base64.strict_encode64("harness:wrong")}" }.freeze

    # Earlier files may already have counted a failure from this address, so
    # the case only requires 401s up to the first 429, within the budget.
    test "after 10 wrong passwords even the right one gets 429" do
      unauthorized = 0
      status = 401
      while status == 401 && unauthorized <= 10
        get("/api/v1/orders", headers: WRONG)
        status = @response.status
        unauthorized = unauthorized + 1 if status == 401
      end
      assert_equal(429, status)
      assert_equal(true, unauthorized >= 1 && unauthorized <= 10)
      get("/api/v1/orders", headers: RIGHT)
      assert_response(429)
    end
  end
end
