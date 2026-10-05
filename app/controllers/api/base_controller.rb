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

    before_action :require_api_basic_auth

    private

    def require_api_basic_auth
      user     = ENV["API_BASIC_AUTH_USER"].presence
      password = ENV["API_BASIC_AUTH_PASSWORD"].presence

      if user.nil? || password.nil?
        head :unauthorized unless Rails.env.local?
        return
      end

      http_basic_authenticate_or_request_with(name: user, password: password, realm: "big_tree_table API")
    end
  end
end
