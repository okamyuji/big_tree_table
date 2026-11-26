# The orders table shared with the Rails and Go versions (schema in
# ../bench/db/schema.sql). `search` ports Order.search from
# rails/app/models/order.rb with raw SQL: above OFFSET_THRESHOLD it switches to
# a deferred join so deep pages do not scan full rows.
#
# Rows come back as name-keyed hashes of Strings and Integers. The SELECT list
# renders dates, decimals and timestamps as text in the forms Rails emits, so
# the JSON matches the Rails API byte for byte.
class Order < Sqlx::Base
  def self.table_name = "orders"

  OFFSET_THRESHOLD = 10_000

  SORTABLE_COLUMNS = %w[
    id order_number order_type order_date customer_name customer_code
    product_name product_code quantity unit_price total_amount status
    delivery_date created_at
  ].freeze

  DEFAULT_PER_PAGE = 50
  MAX_PER_PAGE     = 500
  MAX_PAGE         = 1_000_000

  PAYLOAD_COLUMNS = "o.id, o.order_number, o.order_type, o.order_date::text AS order_date, " \
    "o.customer_name, o.customer_code, o.product_name, o.product_code, o.quantity, " \
    "o.unit_price::text AS unit_price, o.total_amount::text AS total_amount, o.status, " \
    "o.delivery_date::text AS delivery_date, o.notes, " \
    "to_char(o.created_at, 'YYYY-MM-DD\"T\"HH24:MI:SS.MS\"Z\"') AS created_at, " \
    "to_char(o.updated_at, 'YYYY-MM-DD\"T\"HH24:MI:SS.MS\"Z\"') AS updated_at"

  # The rows for one page. params is the controller's string-keyed hash.
  def self.search(params)
    page     = normalize_page(params["page"])
    per_page = normalize_per_page(params["per_page"])
    order_by = order_clause(params["sort"], params["order"])
    offset   = (page - 1) * per_page
    where, binds = filtered(params)

    sql = if offset >= OFFSET_THRESHOLD
      "SELECT #{PAYLOAD_COLUMNS} FROM orders o INNER JOIN " \
        "(SELECT id FROM orders o#{where}#{order_by} LIMIT ? OFFSET ?) deferred_ids " \
        "ON o.id = deferred_ids.id#{order_by}"
    else
      "SELECT #{PAYLOAD_COLUMNS} FROM orders o#{where}#{order_by} LIMIT ? OFFSET ?"
    end
    binds << per_page
    binds << offset
    Order.rows_by_sql(sql, binds)
  end

  def self.search_count(params)
    where, binds = filtered(params)
    row = Order.row_by_sql("SELECT COUNT(*) AS total FROM orders o#{where}", binds)
    row.nil? ? 0 : row["total"].to_i
  end

  # Order.build_tree: customer → product → order nodes in first-seen order.
  # Amounts are summed as integer cents and rendered like BigDecimal#to_s.
  def self.build_tree(rows)
    customers    = []
    customer_idx = {}
    product_idx  = {}

    rows.each do |row|
      cid = "customer:#{row["customer_code"]}"
      ci  = customer_idx[cid]
      if ci.nil?
        customers << branch_node(cid, "customer", 0, row["customer_name"])
        ci = customers.size - 1
        customer_idx[cid] = ci
      end
      customer = customers[ci]
      cents = Money.cents(row["total_amount"])
      add_to_summary(customer["summary"], row, cents)

      pid = "#{cid}:product:#{row["product_code"]}"
      pi  = product_idx[pid]
      if pi.nil?
        customer["children"] << branch_node(pid, "product", 1, row["product_name"])
        pi = customer["children"].size - 1
        product_idx[pid] = pi
      end
      product = customer["children"][pi]
      add_to_summary(product["summary"], row, cents)

      payload = payload(row)
      product["children"] << {
        "id" => "order:#{row["id"]}",
        "kind" => "order",
        "depth" => 2,
        "label" => row["order_number"],
        "order" => payload,
        "summary" => {
          "order_count" => 1,
          "quantity" => row["quantity"],
          "total_amount" => payload["total_amount"],
          "statuses" => [row["status"]]
        },
        "children" => []
      }
    end

    customers.each { |c| finish_summaries(c) }
    customers
  end

  # One row in the shape of Order.order_payload.
  def self.payload(row)
    {
      "id" => row["id"],
      "order_number" => row["order_number"],
      "order_type" => row["order_type"],
      "order_date" => row["order_date"],
      "customer_name" => row["customer_name"],
      "customer_code" => row["customer_code"],
      "product_name" => row["product_name"],
      "product_code" => row["product_code"],
      "quantity" => row["quantity"],
      "unit_price" => Money.decimal_text(row["unit_price"]),
      "total_amount" => Money.decimal_text(row["total_amount"]),
      "status" => row["status"],
      "delivery_date" => row["delivery_date"],
      "notes" => row["notes"],
      "created_at" => row["created_at"],
      "updated_at" => row["updated_at"]
    }
  end

  def self.normalize_page(value)
    int_param(value).clamp(1, MAX_PAGE)
  end

  def self.normalize_per_page(value)
    n = int_param(value)
    return DEFAULT_PER_PAGE if n <= 0

    n < MAX_PER_PAGE ? n : MAX_PER_PAGE
  end

  INT_PREFIX = /\A[ \t\n\v\f\r]*[+-]?\d+(?:_\d+)*/
  SATURATED  = 1_000_000_000_000

  # String#to_i as Rails applies it to page/per_page, but saturating: Spinel
  # integers are 64-bit, so a 20-digit page must not reach to_i.
  def self.int_param(value)
    prefix = value.to_s[INT_PREFIX]
    return 0 if prefix.nil?

    digits = prefix.delete("_ \t\n\v\f\r+-")
    return prefix.to_i if digits.size <= 12

    prefix.include?("-") ? -SATURATED : SATURATED
  end

  def self.order_clause(sort, direction)
    col = sort.to_s
    col = "id" unless SORTABLE_COLUMNS.include?(col)
    dir = direction.to_s.downcase == "desc" ? "DESC" : "ASC"
    return " ORDER BY o.id #{dir}" if col == "id"

    " ORDER BY o.#{col} #{dir}, o.id #{dir}"
  end

  # The WHERE clause and its binds; mirrors Order.filtered.
  def self.filtered(params)
    conds = []
    binds = []
    if present?(params["order_type"])
      conds << "o.order_type = ?"
      binds << params["order_type"]
    end
    if present?(params["status"])
      conds << "o.status = ?"
      binds << params["status"]
    end
    if present?(params["customer_name"])
      conds << "o.customer_name LIKE ?"
      binds << like_pattern(params["customer_name"])
    end
    if present?(params["product_name"])
      conds << "o.product_name LIKE ?"
      binds << like_pattern(params["product_name"])
    end
    from_d = iso_date(params["date_from"])
    unless from_d.nil?
      conds << "o.order_date >= ?"
      binds << from_d
    end
    to_d = iso_date(params["date_to"])
    unless to_d.nil?
      conds << "o.order_date <= ?"
      binds << to_d
    end
    where = conds.empty? ? "" : " WHERE #{conds.join(" AND ")}"
    [where, binds]
  end

  def self.present?(value)
    value.is_a?(String) && !value.empty?
  end

  # ActiveRecord's sanitize_sql_like: backslash escapes \, % and _.
  def self.like_pattern(term)
    out = String.new("%")
    term.each_char do |c|
      out << "\\" if c == "\\" || c == "%" || c == "_"
      out << c
    end
    out << "%"
    out
  end

  # The value when it is a real YYYY-MM-DD calendar date, as Date.iso8601
  # accepts it in Rails; nil otherwise.
  def self.iso_date(value)
    return nil unless present?(value)
    return nil unless value.match?(/\A\d{4}-\d{2}-\d{2}\z/)

    year  = value[0, 4].to_i
    month = value[5, 2].to_i
    day   = value[8, 2].to_i
    return nil if month < 1 || month > 12 || day < 1
    return nil if day > days_in_month(year, month)

    value
  end

  def self.days_in_month(year, month)
    return 29 if month == 2 && leap_year?(year)
    return 28 if month == 2
    return 30 if [4, 6, 9, 11].include?(month)

    31
  end

  def self.leap_year?(year)
    (year % 4 == 0 && year % 100 != 0) || year % 400 == 0
  end

  def self.branch_node(id, kind, depth, label)
    {
      "id" => id,
      "kind" => kind,
      "depth" => depth,
      "label" => label,
      "summary" => { "order_count" => 0, "quantity" => 0, "total_amount" => "0.0", "cents" => 0, "statuses" => [] },
      "children" => []
    }
  end

  def self.add_to_summary(summary, row, cents)
    summary["order_count"] = summary["order_count"] + 1
    summary["quantity"] = summary["quantity"] + row["quantity"]
    summary["cents"] = summary["cents"] + cents
    statuses = summary["statuses"]
    statuses << row["status"] unless statuses.include?(row["status"])
  end

  # Turns the running cents of each branch into the rendered total and drops
  # the working field, so the JSON has only the Rails keys.
  def self.finish_summaries(node)
    summary = node["summary"]
    cents = summary.delete("cents")
    return if cents.nil?

    summary["total_amount"] = Money.format_cents(cents)
    node["children"].each { |child| finish_summaries(child) }
  end
end
