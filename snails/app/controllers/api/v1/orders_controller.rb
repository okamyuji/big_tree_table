module Api
  module V1
    # GET /api/v1/orders and GET /api/v1/orders/tree, the same contract as
    # rails/app/controllers/api/v1/orders_controller.rb.
    class OrdersController < ActionController::API
      # Snails does not inherit before_action, so the guard is declared here.
      before_action :guard_api

      def index
        rows, meta = paged_search
        render(json: { "orders" => rows.map { |row| Order.payload(row) }, "meta" => meta })
      end

      def tree
        rows, meta = paged_search
        render(json: { "tree" => Order.build_tree(rows), "meta" => meta })
      end

      private

      def paged_search
        rows  = Order.search(params)
        total = Order.search_count(params)
        headers["X-Total-Count"] = total.to_s

        per_page = Order.normalize_per_page(params["per_page"])
        meta = {
          "total" => total,
          "page" => Order.normalize_page(params["page"]),
          "per_page" => per_page,
          "total_pages" => total == 0 ? 0 : (total + per_page - 1) / per_page
        }
        [rows, meta]
      end

      # Api::BaseController#require_api_basic_auth plus its rate_limit.
      def guard_api
        return head(:too_many_requests) if ApiGuard.over_rate_limit?

        user = ENV["API_BASIC_AUTH_USER"].to_s
        pass = ENV["API_BASIC_AUTH_PASSWORD"].to_s
        if user.empty? || pass.empty?
          head(:unauthorized) unless Snails.env.development? || Snails.env.test?
          return
        end

        # The browser's first request carries no credentials, so it is neither counted nor blocked.
        return request_http_basic_authentication(ApiGuard::REALM) if request.headers["Authorization"].to_s.empty?

        # Count before judging so concurrent guesses cannot all pass a separate read.
        ip = request.remote_ip
        return head(:too_many_requests) if ApiGuard.count_failure(ip, 1) > ApiGuard::AUTH_FAILURE_LIMIT

        matched = authenticate_with_http_basic do |given_user, given_pass|
          user_ok = ApiGuard.secure_compare(given_user, user)
          pass_ok = ApiGuard.secure_compare(given_pass, pass)
          user_ok && pass_ok
        end
        if matched == true
          ApiGuard.count_failure(ip, -1)
          return
        end

        request_http_basic_authentication(ApiGuard::REALM)
      end
    end
  end
end
