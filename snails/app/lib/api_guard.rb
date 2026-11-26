# The API's request budget and Basic-auth check, matching
# rails/app/controllers/api/base_controller.rb: a global fixed-window rate
# limit, then a per-IP budget for wrong credentials. Counters live in
# Snails.cache, which is per process like Rails' memory_store in the bench env.
module ApiGuard
  REALM               = "big_tree_table API"
  DEFAULT_RATE_LIMIT  = 300
  RATE_LIMIT_WINDOW   = 60
  AUTH_FAILURE_LIMIT  = 10
  AUTH_FAILURE_WINDOW = 15 * 60

  def self.rate_limit
    value = ENV["API_RATE_LIMIT"].to_s
    return DEFAULT_RATE_LIMIT unless value.match?(/\A\+?\d{1,12}\z/)

    n = value.to_i
    n > 0 ? n : DEFAULT_RATE_LIMIT
  end

  # True once this request pushes the global count past the limit.
  def self.over_rate_limit?
    Snails.cache.increment("rate-limit:api", 1, expires_in: RATE_LIMIT_WINDOW) > rate_limit
  end

  def self.count_failure(ip, amount)
    Snails.cache.increment("auth-failure:api:#{ip}", amount, expires_in: AUTH_FAILURE_WINDOW)
  end

  # Compares every byte of equal-length strings so the time taken does not
  # depend on where they differ, like ActiveSupport::SecurityUtils.secure_compare.
  def self.secure_compare(given, expected)
    return false unless given.bytesize == expected.bytesize

    diff = 0
    i = 0
    while i < given.bytesize
      diff = diff | (given.getbyte(i) ^ expected.getbyte(i))
      i = i + 1
    end
    diff == 0
  end
end
