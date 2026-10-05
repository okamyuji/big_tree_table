# typed: true

# Base controller for the JSON API.
#
# - Inherits from ActionController::API so we skip the HTML middleware stack
#   (cookies, flash, CSRF, etc.) entirely — the React frontend is the only
#   intended client and it talks to us purely over JSON.
# - allow_browser is intentionally NOT applied here because API clients aren't
#   browsers and the user-agent gate would only block legitimate calls.
# - HTTP Basic auth: the SPA has no login and any token it held would ship in
#   the JS bundle, so the browser's own Basic dialog is the gate. Credentials
#   are read per request (not via the class-level http_basic_authenticate_with)
#   so a missing variable fails closed outside development/test.
module Api
  class BaseController < ActionController::API
    include ActionController::HttpAuthentication::Basic::ControllerMethods

    AUTH_REALM = "big_tree_table API"

    RATE_LIMIT        = 300
    RATE_LIMIT_WINDOW = 1.minute

    # A stricter budget for wrong credentials only. rate_limit cannot express
    # this: it increments on every request, and an `if:` that skips successes
    # would let the right password through while wrong ones get 429, which
    # tells a guesser when it has hit. So once the limit is reached, every
    # request from that IP gets 429, right password included.
    AUTH_FAILURE_LIMIT  = 10
    AUTH_FAILURE_WINDOW = 15.minutes

    # Declared before the auth check so rejected (401) attempts are counted too;
    # otherwise the halted chain would let password guessing bypass the limit.
    # A fixed scope keeps one budget for the whole API; the default is per controller.
    rate_limit to: RATE_LIMIT, within: RATE_LIMIT_WINDOW, scope: "api"
    before_action :require_api_basic_auth

    private

    def require_api_basic_auth
      user     = ENV["API_BASIC_AUTH_USER"].presence
      password = ENV["API_BASIC_AUTH_PASSWORD"].presence

      if user.nil? || password.nil?
        head :unauthorized unless Rails.env.local?
        return
      end

      # The browser's first request carries no credentials and cannot guess, so
      # it is neither counted nor blocked.
      return request_http_basic_authentication(AUTH_REALM) if request.authorization.blank?

      # Count first and judge on the value increment returns: a separate read
      # would let concurrent guesses all pass the check before any is counted.
      # Solid Cache returns nil instead of raising on a transient DB error such as
      # a deadlock on a new key. Retry once, then fail closed: an uncounted guess
      # would be a free one.
      failures = count_auth_attempt || count_auth_attempt
      raise ActionController::TooManyRequests if failures.nil? || failures > AUTH_FAILURE_LIMIT

      if authenticate_with_http_basic { |given_user, given_password|
        ActiveSupport::SecurityUtils.secure_compare(given_user.to_s, user) &
          ActiveSupport::SecurityUtils.secure_compare(given_password.to_s, password)
      }
        cache_store.decrement(auth_failure_key, 1, expires_in: AUTH_FAILURE_WINDOW)
        return
      end

      request_http_basic_authentication(AUTH_REALM)
    end

    def count_auth_attempt
      cache_store.increment(auth_failure_key, 1, expires_in: AUTH_FAILURE_WINDOW)
    end

    def auth_failure_key
      "auth-failure:api:#{request.remote_ip}"
    end
  end
end
