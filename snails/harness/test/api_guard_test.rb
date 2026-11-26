# ApiGuard runs in this test process too, so its rules are checked directly:
# the comparison, the API_RATE_LIMIT parsing and the fixed-window counter.
require_relative "../boot"

if Snails::Harness.file_selected?
  class ApiGuardTest < Snails::Harness::IntegrationTest
    test "secure_compare matches only identical strings" do
      assert_equal(true, ApiGuard.secure_compare("harness", "harness"))
      assert_equal(false, ApiGuard.secure_compare("harnest", "harness"))
      assert_equal(false, ApiGuard.secure_compare("harnes", "harness"))
      assert_equal(false, ApiGuard.secure_compare("harnesss", "harness"))
      assert_equal(false, ApiGuard.secure_compare("", "harness"))
      assert_equal(true, ApiGuard.secure_compare("", ""))
    end

    test "API_RATE_LIMIT accepts only a positive integer" do
      cases = [["", 300], ["1000000", 1_000_000], ["+5", 5], ["0", 300], ["-1", 300], ["abc", 300], ["12abc", 300]]
      cases.each do |pair|
        ENV["API_RATE_LIMIT"] = pair[0]
        assert_equal(pair[1], ApiGuard.rate_limit)
      end
    end

    test "the global counter trips once the limit is passed" do
      ENV["API_RATE_LIMIT"] = "3"
      assert_equal(false, ApiGuard.over_rate_limit?)
      assert_equal(false, ApiGuard.over_rate_limit?)
      assert_equal(false, ApiGuard.over_rate_limit?)
      assert_equal(true, ApiGuard.over_rate_limit?)
    end

    test "failure counts add up per address and give back on success" do
      assert_equal(1, ApiGuard.count_failure("198.51.100.7", 1))
      assert_equal(2, ApiGuard.count_failure("198.51.100.7", 1))
      assert_equal(1, ApiGuard.count_failure("198.51.100.7", -1))
      assert_equal(1, ApiGuard.count_failure("198.51.100.8", 1))
    end
  end
end
