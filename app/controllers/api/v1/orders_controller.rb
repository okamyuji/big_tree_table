# typed: true

module Api
  module V1
    # Versioned, resource-scoped JSON endpoints used by the React frontend:
    #
    #   GET /api/v1/orders        — flat paged list with `orders` + `meta` envelope
    #   GET /api/v1/orders/tree   — recursive customer→product→order tree
    #
    # Both share Order.search/Order.search_count, so the deferred-join switch
    # past OFFSET_THRESHOLD applies uniformly.
    class OrdersController < Api::BaseController
      # GET /api/v1/orders
      def index
        params_hash = list_params.to_h.symbolize_keys
        scope = Order.search(params_hash)
        total = Order.search_count(params_hash)
        per   = per_page_for(params_hash)
        page  = page_for(params_hash)

        response.set_header("X-Total-Count", total.to_s)

        render json: {
          orders: scope.map { |o| Order.order_payload(o) },
          meta:   meta(total, page, per)
        }
      end

      # GET /api/v1/orders/tree
      def tree
        params_hash = list_params.to_h.symbolize_keys
        scope = Order.search(params_hash)
        total = Order.search_count(params_hash)
        per   = per_page_for(params_hash)
        page  = page_for(params_hash)

        response.set_header("X-Total-Count", total.to_s)

        render json: {
          tree: Order.build_tree(scope.to_a),
          meta: meta(total, page, per)
        }
      end

      private

      def list_params
        params.permit(
          :order_type, :status,
          :customer_name, :product_name,
          :date_from, :date_to,
          :sort, :order,
          :page, :per_page
        )
      end

      def page_for(params_hash)
        n = params_hash[:page].to_i
        n < 1 ? 1 : n
      end

      def per_page_for(params_hash)
        n = params_hash[:per_page].to_i
        return Order::DEFAULT_PER_PAGE if n <= 0

        [ n, Order::MAX_PER_PAGE ].min
      end

      def meta(total, page, per_page)
        {
          total:       total,
          page:        page,
          per_page:    per_page,
          total_pages: total.to_i.zero? ? 0 : (total.to_f / per_page).ceil
        }
      end
    end
  end
end
