# GET /api/v1/orders and /api/v1/orders/tree against a few hand-made rows.
# Cross-implementation equality with Rails runs in ../bench/parity.sh; these
# cases pin the snails-side rules that the seed data alone would not reach.
require_relative "../boot"

if Snails::Harness.file_selected?
  class OrdersApiTest < Snails::Harness::IntegrationTest
    AUTH = { "Authorization" => "Basic #{Base64.strict_encode64("harness:not-a-real-password")}" }.freeze

    def tables
      ["orders"]
    end

    def setup
      insert("ORD-1", "C1", "Odd 100% _x\\y O'Brien ?", "P1", 1, "0.10", "shipped", "2026-02-28")
      insert("ORD-2", "C1", "Odd 100% _x\\y O'Brien ?", "P1", 2, "0.20", "pending", "2026-02-28")
      insert("ORD-3", "C1", "Odd 100% _x\\y O'Brien ?", "P2", 3, "12.45", "shipped", "2024-02-29")
      insert("ORD-4", "C2", "Plain", "P1", 4, "-0.50", "cancelled", "2026-01-15")
    end

    def insert(number, customer_code, customer_name, product_code, qty, total, status, date)
      Order.write(
        "INSERT INTO orders (order_number, order_type, order_date, customer_name, customer_code, " \
        "product_name, product_code, quantity, unit_price, total_amount, status, delivery_date, notes, " \
        "created_at, updated_at) VALUES (?, 'rush', ?, ?, ?, ?, ?, ?, 0.1, ?, ?, NULL, '', " \
        "'2026-11-01 12:34:56.789', '2026-11-01 12:34:56.789') RETURNING id",
        [number, date, customer_name, customer_code, "Product #{product_code}", product_code, qty, total, status]
      )
    end

    test "a request without credentials gets the Basic challenge" do
      get("/api/v1/orders")
      assert_response(:unauthorized)
      assert_equal("Basic realm=\"big_tree_table API\"", @response.headers["WWW-Authenticate"])
    end

    test "a wrong password gets 401" do
      bad = { "Authorization" => "Basic #{Base64.strict_encode64("harness:wrong")}" }
      get("/api/v1/orders", headers: bad)
      assert_response(:unauthorized)
    end

    test "the list renders values the way Rails does" do
      get("/api/v1/orders", params: { "sort" => "order_number" }, headers: AUTH)
      assert_response(:ok)
      assert_equal("4", @response.headers["X-Total-Count"])
      first = @response.parsed_body["orders"][0]
      assert_equal("ORD-1", first["order_number"])
      assert_equal("0.1", first["total_amount"])
      assert_equal("0.1", first["unit_price"])
      assert_equal("2026-02-28", first["order_date"])
      assert_nil(first["delivery_date"])
      assert_equal("2026-11-01T12:34:56.789Z", first["created_at"])
      meta = @response.parsed_body["meta"]
      assert_equal(4, meta["total"])
      assert_equal(1, meta["total_pages"])
    end

    test "LIKE terms match %, _, backslash, quote and ? literally" do
      get("/api/v1/orders", params: { "customer_name" => "100% _x\\y O'Brien ?" }, headers: AUTH)
      assert_equal(3, @response.parsed_body["meta"]["total"])
      get("/api/v1/orders", params: { "customer_name" => "%" }, headers: AUTH)
      assert_equal(3, @response.parsed_body["meta"]["total"])
      get("/api/v1/orders", params: { "customer_name" => "Pl_in" }, headers: AUTH)
      assert_equal(0, @response.parsed_body["meta"]["total"])
    end

    test "invalid paging, sort and dates fall back to the defaults" do
      get("/api/v1/orders", params: { "page" => "99999999999999999999", "per_page" => "-5",
                                      "sort" => "bogus", "order" => "sideways",
                                      "date_from" => "2026-02-30", "date_to" => "2026-13-01" }, headers: AUTH)
      assert_response(:ok)
      meta = @response.parsed_body["meta"]
      assert_equal(1_000_000, meta["page"])
      assert_equal(50, meta["per_page"])
      assert_equal(4, meta["total"])
    end

    test "dates filter inclusively, leap day included" do
      get("/api/v1/orders", params: { "date_from" => "2024-02-29", "date_to" => "2024-02-29" }, headers: AUTH)
      assert_equal(1, @response.parsed_body["meta"]["total"])
    end

    test "the tree sums fractional amounts exactly" do
      get("/api/v1/orders/tree", params: { "sort" => "order_number" }, headers: AUTH)
      assert_response(:ok)
      tree = @response.parsed_body["tree"]
      assert_equal(2, tree.size)
      c1 = tree[0]
      assert_equal("customer:C1", c1["id"])
      assert_equal("12.75", c1["summary"]["total_amount"])
      assert_equal(["shipped", "pending"], c1["summary"]["statuses"])
      assert_equal(6, c1["summary"]["quantity"])
      p1 = c1["children"][0]
      assert_equal("customer:C1:product:P1", p1["id"])
      assert_equal("0.3", p1["summary"]["total_amount"])
      assert_equal(2, p1["children"].size)
      assert_equal("order:", p1["children"][1]["id"][0, 6])
      assert_equal("0.2", p1["children"][1]["summary"]["total_amount"])
      assert_equal("-0.5", tree[1]["summary"]["total_amount"])
      assert_nil(c1["summary"]["cents"])
    end
  end
end
