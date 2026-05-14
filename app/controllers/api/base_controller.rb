# typed: true

# Base controller for the JSON API.
#
# - Inherits from ActionController::API so we skip the HTML middleware stack
#   (cookies, flash, CSRF, etc.) entirely — the React frontend is the only
#   intended client and it talks to us purely over JSON.
# - allow_browser is intentionally NOT applied here because API clients aren't
#   browsers and the user-agent gate would only block legitimate calls.
module Api
  class BaseController < ActionController::API
  end
end
