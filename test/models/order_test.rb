# typed: false

require "test_helper"

class OrderTest < ActiveSupport::TestCase
  # ----- validations --------------------------------------------------------

  test "valid fixture is valid" do
    assert_predicate orders(:order_one), :valid?
  end

  test "requires presence of all NOT NULL columns" do
    o = Order.new

    assert_not o.valid?
    %i[order_number order_type order_date
       customer_name customer_code
       product_name product_code
       quantity unit_price total_amount status].each do |attr|
      assert_includes o.errors.attribute_names, attr, "missing presence error on #{attr}"
    end
  end

  # ----- normalize_* helpers (via .search behaviour) ------------------------

  test "search defaults: page 1, default per_page, asc by id" do
    rel = Order.search({})
    sql = rel.to_sql
    # When the sort column is id itself, the explicit (id, id) tiebreaker
    # collapses to one ORDER BY id ASC clause — that is the expected SQL.
    assert_match(/ORDER BY `orders`\.`id` ASC/i, sql)
    assert_no_match(/ORDER BY .*ORDER BY/i, sql)
    assert_match(/LIMIT #{Order::DEFAULT_PER_PAGE}/, sql)
  end

  test "search caps per_page at MAX_PER_PAGE" do
    sql = Order.search(per_page: 9999).to_sql

    assert_match(/LIMIT #{Order::MAX_PER_PAGE}/, sql)
  end

  test "search rejects unknown sort column and falls back to id" do
    sql = Order.search(sort: "; DROP TABLE orders --").to_sql

    assert_match(/ORDER BY .*`orders`\.`id`/i, sql)
    assert_no_match(/DROP TABLE/i, sql)
  end

  test "search accepts whitelisted sort columns" do
    Order::SORTABLE_COLUMNS.each do |col|
      sql = Order.search(sort: col, order: "desc").to_sql

      assert_match(/`orders`\.`#{col}` DESC/, sql, "expected ORDER BY on #{col}")
    end
  end

  test "search treats invalid order direction as asc" do
    sql = Order.search(sort: "order_date", order: "OR 1=1").to_sql

    assert_match(/`order_date` ASC/, sql)
  end

  test "search treats negative or zero page as page 1" do
    sql_neg  = Order.search(page: "-3").to_sql
    sql_zero = Order.search(page: "0").to_sql

    assert_match(/OFFSET 0/, sql_neg)
    assert_match(/OFFSET 0/, sql_zero)
  end

  # ----- deferred-join switch ----------------------------------------------

  test "search uses plain LIMIT/OFFSET below the offset threshold" do
    sql = Order.search(page: 2, per_page: 50).to_sql

    assert_match(/LIMIT 50 OFFSET 50/, sql)
    assert_no_match(/INNER JOIN \(SELECT/i, sql)
  end

  test "search switches to deferred join at or above the threshold" do
    # offset = (page-1) * per_page. With per_page=50, page=201 → offset=10_000.
    sql = Order.search(page: 201, per_page: 50).to_sql

    assert_match(/INNER JOIN \(SELECT/i, sql, "expected INNER JOIN (SELECT id …) deferred join")
    assert_match(/LIMIT 50 OFFSET 10000/, sql)
  end

  test "search threshold is exactly OFFSET_THRESHOLD inclusive" do
    # Just under (offset = 9_950): plain query.
    just_under = Order.search(page: 200, per_page: 50).to_sql

    assert_no_match(/INNER JOIN \(SELECT/i, just_under)

    # Exactly at threshold (offset = 10_000) → deferred join.
    at_thresh = Order.search(page: 21, per_page: 500).to_sql # offset = 10_000

    assert_match(/INNER JOIN \(SELECT/i, at_thresh)
  end

  # ----- filtering ---------------------------------------------------------

  test "exact-match filters: order_type, status" do
    rel = Order.search(order_type: "rush")

    assert_includes rel.to_a, orders(:order_two)
    assert_not_includes rel.to_a, orders(:order_one)

    rel2 = Order.search(status: "confirmed")

    assert_includes rel2.to_a, orders(:order_one)
    assert_not_includes rel2.to_a, orders(:order_two)
  end

  test "blank string filter values are ignored" do
    rel = Order.search(order_type: "", status: "")

    assert_equal Order.count, rel.count
  end

  test "LIKE filter escapes %, _, and \\" do
    # Should match the literal "ACME 100%" customer, not be tricked into wildcards.
    rel = Order.search(customer_name: "100%")

    assert_includes rel.to_a, orders(:order_three_like_match)
    assert_equal 1, rel.count

    rel2 = Order.search(product_name: "Widget_X")

    assert_includes rel2.to_a, orders(:order_three_like_match)
    assert_equal 1, rel2.count
  end

  test "LIKE filter matches substrings when escaping is not triggered" do
    rel = Order.search(customer_name: "Customer 0001")

    assert_includes rel.to_a, orders(:order_one)
  end

  test "date_from and date_to apply inclusive bounds on order_date" do
    rel = Order.search(date_from: "2026-02-01", date_to: "2026-02-28")
    ids = rel.pluck(:id)

    assert_includes ids, orders(:order_two).id
    assert_not_includes ids, orders(:order_one).id
  end

  test "malformed dates are silently dropped" do
    rel = Order.search(date_from: "not-a-date", date_to: "2026-99-99")

    assert_equal Order.count, rel.count
  end

  # ----- search_count + .count consistency ---------------------------------

  test "search_count matches the unpaginated row count for the same filter" do
    full = Order.search_count(status: "confirmed")
    paged = Order.where(status: "confirmed").count

    assert_equal paged, full
  end

  # ----- build_tree --------------------------------------------------------

  test "build_tree returns recursive customer→product→order nodes (Go BuildOrderTree shape)" do
    rows = Order.all.to_a
    tree = Order.build_tree(rows)

    assert_equal Order.distinct.pluck(:customer_code).size, tree.size

    customer = tree.find { |c| c[:id] == "customer:#{orders(:order_one).customer_code}" }

    assert customer
    assert_equal "customer", customer[:kind]
    assert_equal 0, customer[:depth]
    assert_equal orders(:order_one).customer_name, customer[:label]
    assert_kind_of Array, customer[:children]

    product = customer[:children].find do |p|
      p[:id] == "customer:#{orders(:order_one).customer_code}:product:#{orders(:order_one).product_code}"
    end

    assert product
    assert_equal "product", product[:kind]
    assert_equal 1, product[:depth]
    assert_kind_of Array, product[:children]

    order_node = product[:children].find { |n| n[:order][:id] == orders(:order_one).id }

    assert order_node
    assert_equal "order", order_node[:kind]
    assert_equal 2, order_node[:depth]

    summary = customer[:summary]

    assert_equal customer[:children].sum { |p| p[:summary][:order_count] }, summary[:order_count]
  end

  test "summary keys: order_count, quantity, total_amount, statuses" do
    rows = [ orders(:order_one), orders(:order_two) ]
    tree = Order.build_tree(rows)
    customer = tree.first

    assert_equal 1, customer[:summary][:order_count]
    assert_equal rows.first.quantity, customer[:summary][:quantity]
    assert_equal rows.first.total_amount, customer[:summary][:total_amount]
    assert_equal rows.first.status, customer[:summary][:statuses].first
  end

  test "order_payload returns Rails-default Date and BigDecimal types and includes notes + updated_at" do
    o       = orders(:order_one)
    payload = Order.order_payload(o)

    assert_equal o.id, payload[:id]
    assert_equal Date.new(2026, 1, 15), payload[:order_date]
    assert_equal Date.new(2026, 1, 20), payload[:delivery_date]
    assert_equal o.unit_price,   payload[:unit_price]
    assert_equal o.total_amount, payload[:total_amount]
    assert_equal "", payload[:notes]
    assert payload.key?(:updated_at)
  end

  test "order_payload returns nil delivery_date when the column is null (Rails default)" do
    payload = Order.order_payload(orders(:order_three_like_match))

    assert_nil payload[:delivery_date]
  end

  # ----- deferred join end-to-end (correctness, not perf) ------------------

  test "deferred-join path returns the same ids as the equivalent plain query" do
    # Capture a fixture row's attributes BEFORE delete_all so we can re-use the
    # column shape, then wipe the table for a clean ordered insert.
    template = orders(:order_one).attributes.except("id", "order_number", "created_at", "updated_at")
    Order.delete_all
    rows = Array.new(50) do |i|
      template.merge(
        "order_number" => format("ORD-DJ-%010d", i),
        "order_date"   => Date.new(2026, 1, 1) + i,
        "created_at"   => Time.current,
        "updated_at"   => Time.current
      )
    end
    Order.insert_all(rows)

    # Lower the threshold via stub-ish constant override pattern.
    original = Order::OFFSET_THRESHOLD
    Order.send(:remove_const, :OFFSET_THRESHOLD)
    Order.const_set(:OFFSET_THRESHOLD, 10)
    begin
      plain    = Order.where(nil).reorder(id: :asc).limit(5).offset(15).pluck(:id)
      deferred = Order.search(page: 4, per_page: 5, sort: "id", order: "asc").pluck(:id)

      assert_equal plain, deferred
    ensure
      Order.send(:remove_const, :OFFSET_THRESHOLD)
      Order.const_set(:OFFSET_THRESHOLD, original)
    end
  end
end
