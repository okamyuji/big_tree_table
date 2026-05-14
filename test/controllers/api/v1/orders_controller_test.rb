# typed: false

require "test_helper"

module Api
  module V1
    class OrdersControllerTest < ActionDispatch::IntegrationTest
      test "GET /api/v1/orders returns Rails-style { orders, meta } envelope" do
        get "/api/v1/orders", params: { per_page: 2, page: 1, sort: "id", order: "asc" }

        assert_response :success
        body = JSON.parse(response.body)

        assert_kind_of Array, body["orders"]
        assert_operator body["orders"].size, :<=, 2

        meta = body["meta"]

        assert_equal 1, meta["page"]
        assert_equal 2, meta["per_page"]
        assert_equal Order.count, meta["total"]
        assert_equal (Order.count.to_f / 2).ceil, meta["total_pages"]
        assert_equal Order.count.to_s, response.headers["X-Total-Count"]
      end

      test "GET /api/v1/orders applies the order_type filter" do
        get "/api/v1/orders", params: { order_type: "rush" }

        assert_response :success
        body = JSON.parse(response.body)

        assert(body["orders"].all? { |o| o["order_type"] == "rush" })
      end

      test "GET /api/v1/orders applies the LIKE-escaped customer_name filter" do
        get "/api/v1/orders", params: { customer_name: "100%" }

        assert_response :success
        body = JSON.parse(response.body)

        assert_equal 1, body["orders"].size
        assert_equal "ACME 100% Corp", body["orders"].first["customer_name"]
      end

      test "GET /api/v1/orders date_from / date_to map to order_date inclusive bounds" do
        get "/api/v1/orders", params: { date_from: "2026-02-01", date_to: "2026-02-28" }
        body = JSON.parse(response.body)

        ids = body["orders"].map { |o| o["id"] }

        assert_includes ids, orders(:order_two).id
        assert_not_includes ids, orders(:order_one).id
      end

      test "GET /api/v1/orders payload uses Rails default Decimal serialization (string) and includes notes + updated_at" do
        get "/api/v1/orders", params: { per_page: 1, sort: "id", order: "asc" }
        body  = JSON.parse(response.body)
        first = body["orders"].first

        %w[id order_number order_type order_date customer_code customer_name
           product_code product_name quantity unit_price total_amount status
           delivery_date notes created_at updated_at].each do |k|
          assert first.key?(k), "missing key #{k} in payload"
        end
        # ActiveRecord serialises BigDecimal as String to keep precision.
        assert_kind_of String, first["unit_price"]
        assert_kind_of String, first["total_amount"]
        assert_kind_of Numeric, first["quantity"]
      end

      test "GET /api/v1/orders/tree returns recursive customer→product→order nodes" do
        get "/api/v1/orders/tree", params: { sort: "id", order: "asc" }

        assert_response :success
        body = JSON.parse(response.body)

        assert_kind_of Array, body["tree"]
        customer = body["tree"].first

        assert_equal "customer", customer["kind"]
        assert_equal 0, customer["depth"]
        assert customer.key?("summary")
        assert_kind_of Array, customer["children"]

        product = customer["children"].first

        assert_equal "product", product["kind"]
        assert_equal 1, product["depth"]
        assert_kind_of Array, product["children"]

        order = product["children"].first

        assert_equal "order", order["kind"]
        assert_equal 2, order["depth"]
        assert order.key?("order")
        assert_equal order["order"]["order_number"], order["label"]
      end

      test "GET /api/v1/orders defaults paging to page=1, per_page=DEFAULT_PER_PAGE" do
        get "/api/v1/orders"
        body = JSON.parse(response.body)

        assert_equal 1, body["meta"]["page"]
        assert_equal Order::DEFAULT_PER_PAGE, body["meta"]["per_page"]
      end

      test "GET /api/v1/orders caps per_page at MAX_PER_PAGE" do
        get "/api/v1/orders", params: { per_page: 9999 }
        body = JSON.parse(response.body)

        assert_equal Order::MAX_PER_PAGE, body["meta"]["per_page"]
      end

      test "GET /api/v1/orders normalises non-positive page to 1" do
        get "/api/v1/orders", params: { page: -5 }
        body = JSON.parse(response.body)

        assert_equal 1, body["meta"]["page"]
      end
    end
  end
end
