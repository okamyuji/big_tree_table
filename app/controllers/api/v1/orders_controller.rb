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
        scope, page_meta = paged_search

        render json: {
          orders: scope.map { |o| Order.order_payload(o) },
          meta:   page_meta
        }
      end

      # GET /api/v1/orders/tree
      def tree
        scope, page_meta = paged_search

        render json: {
          tree: Order.build_tree(scope.to_a),
          meta: page_meta
        }
      end

      private

      def paged_search
        params_hash = list_params.to_h.symbolize_keys
        scope = Order.search(params_hash)
        total = Order.search_count(params_hash)

        response.set_header("X-Total-Count", total.to_s)

        page     = Order.normalize_page(params_hash[:page])
        per_page = Order.normalize_per_page(params_hash[:per_page])

        [ scope, meta(total, page, per_page) ]
      end

      def list_params
        params.permit(
          :order_type, :status,
          :customer_name, :product_name,
          :date_from, :date_to,
          :sort, :order,
          :page, :per_page
        )
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
