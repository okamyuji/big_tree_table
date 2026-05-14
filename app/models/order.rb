# typed: true

# Order is the single table that backs the BigTreeTable virtual-scroll grid.
#
# `search` ports BigTreeTable's BuildQuery: above OFFSET_THRESHOLD it switches
# from a plain LIMIT/OFFSET query to a deferred join (PK-only inner query
# joined back to the row), which is what keeps deep paging from collapsing on
# a 1M+ row table.
class Order < ApplicationRecord
  OFFSET_THRESHOLD = 10_000

  SORTABLE_COLUMNS = %w[
    id
    order_number
    order_type
    order_date
    customer_name
    customer_code
    product_name
    product_code
    quantity
    unit_price
    total_amount
    status
    delivery_date
    created_at
  ].freeze

  ALLOWED_SORT_DIRECTIONS = %w[asc desc].freeze

  DEFAULT_PER_PAGE = 50
  MAX_PER_PAGE     = 500

  validates :order_number, :order_type, :order_date,
            :customer_name, :customer_code,
            :product_name,  :product_code,
            :quantity, :unit_price, :total_amount, :status,
            presence: true

  # Build the filtered, sorted, paginated relation.
  #
  # params keys (all optional, all strings from the controller):
  #   :order_type, :status               — exact match
  #   :customer_name, :product_name      — case-sensitive LIKE %term%, escaped
  #   :date_from, :date_to               — inclusive YYYY-MM-DD bounds on order_date
  #   :sort, :order                      — column whitelist + asc/desc
  #   :page, :per_page                   — 1-indexed paging
  def self.search(params)
    page     = normalize_page(params[:page])
    per_page = normalize_per_page(params[:per_page])
    sort_col = normalize_sort_column(params[:sort])
    sort_dir = normalize_sort_direction(params[:order])
    offset   = (page - 1) * per_page

    base = filtered(params)

    if offset >= OFFSET_THRESHOLD
      deferred_join(base, sort_col, sort_dir, per_page, offset)
    else
      base
        .reorder(sort_col => sort_dir, :id => sort_dir)
        .limit(per_page)
        .offset(offset)
    end
  end

  # SQL form (matches BigTreeTable's BuildQuery output 1:1):
  #   SELECT o.* FROM orders o
  #   INNER JOIN (SELECT id FROM orders WHERE … ORDER BY … LIMIT … OFFSET …) sub
  #     ON o.id = sub.id
  #   ORDER BY …
  #
  # Uses the Arel JOIN form because MySQL still rejects the simpler
  # `WHERE id IN (SELECT id FROM … LIMIT … OFFSET …)` shape with
  # "This version of MySQL doesn't yet support 'LIMIT & IN/ALL/ANY/SOME subquery'".
  def self.deferred_join(base, sort_col, sort_dir, per_page, offset)
    sub_rel = base
      .reorder(sort_col => sort_dir, :id => sort_dir)
      .limit(per_page)
      .offset(offset)
      .select(:id)

    sub_alias = Arel::Nodes::TableAlias.new(Arel.sql("(#{sub_rel.to_sql})"), :deferred_ids)
    o         = arel_table
    join_node = o.create_join(sub_alias, o.create_on(o[:id].eq(sub_alias[:id])))

    joins(join_node).reorder(sort_col => sort_dir, :id => sort_dir)
  end
  private_class_method :deferred_join

  # Same WHERE clause as `search`, no ORDER BY / LIMIT — used by the controller
  # to populate the `meta.total` field in the JSON envelope.
  def self.search_count(params)
    filtered(params).count
  end

  # Group a flat list of Order rows into the recursive tree node array used
  # by the React TreeTable. Each node has a uniform `{id, kind, depth, label,
  # summary, children}` shape; `kind: "order"` leaves additionally carry the
  # full order payload under :order.
  def self.build_tree(orders)
    customers     = []
    customer_idx  = {}
    product_idx   = {}

    orders.each do |o|
      cid = "customer:#{o.customer_code}"
      ci  = customer_idx[cid]
      if ci.nil?
        customers << build_node(cid, "customer", 0, o.customer_name)
        ci = customers.size - 1
        customer_idx[cid] = ci
        product_idx[cid]  = {}
      end
      apply_summary(customers[ci][:summary], o)

      pid = "#{cid}:product:#{o.product_code}"
      pi  = product_idx[cid][pid]
      if pi.nil?
        customers[ci][:children] << build_node(pid, "product", 1, o.product_name)
        pi = customers[ci][:children].size - 1
        product_idx[cid][pid] = pi
      end
      product_node = customers[ci][:children][pi]
      apply_summary(product_node[:summary], o)

      product_node[:children] << {
        id:       "order:#{o.id}",
        kind:     "order",
        depth:    2,
        label:    o.order_number,
        order:    order_payload(o),
        summary:  {
          order_count:  1,
          quantity:     o.quantity,
          total_amount: o.total_amount,
          statuses:     [ o.status ]
        },
        children: []
      }
    end

    customers
  end

  # Plain hash payload — Rails serialises BigDecimal as a string and Date as
  # ISO8601 by default. We rely on those defaults instead of forcing Float /
  # empty-string nulls.
  def self.order_payload(order)
    {
      id:            order.id,
      order_number:  order.order_number,
      order_type:    order.order_type,
      order_date:    order.order_date,
      customer_name: order.customer_name,
      customer_code: order.customer_code,
      product_name:  order.product_name,
      product_code:  order.product_code,
      quantity:      order.quantity,
      unit_price:    order.unit_price,
      total_amount:  order.total_amount,
      status:        order.status,
      delivery_date: order.delivery_date,
      notes:         order.notes,
      created_at:    order.created_at,
      updated_at:    order.updated_at
    }
  end

  # ----- private helpers ---------------------------------------------------

  def self.build_node(id, kind, depth, label)
    {
      id:       id,
      kind:     kind,
      depth:    depth,
      label:    label,
      summary:  { order_count: 0, quantity: 0, total_amount: BigDecimal("0"), statuses: [] },
      children: []
    }
  end
  private_class_method :build_node

  def self.apply_summary(summary, order)
    summary[:order_count] += 1
    summary[:quantity]    += order.quantity
    summary[:total_amount] += order.total_amount
    summary[:statuses] << order.status unless summary[:statuses].include?(order.status)
  end
  private_class_method :apply_summary

  def self.filtered(params)
    rel = all
    rel = rel.where(order_type: params[:order_type]) if present?(params[:order_type])
    rel = rel.where(status: params[:status])         if present?(params[:status])

    if present?(params[:customer_name])
      rel = rel.where("customer_name LIKE ?", like_pattern(params[:customer_name]))
    end
    if present?(params[:product_name])
      rel = rel.where("product_name LIKE ?", like_pattern(params[:product_name]))
    end

    from_d = parse_iso_date(params[:date_from])
    to_d   = parse_iso_date(params[:date_to])
    rel = rel.where("order_date >= ?", from_d) if from_d
    rel = rel.where("order_date <= ?", to_d)   if to_d

    rel
  end
  private_class_method :filtered

  def self.present?(value)
    value.is_a?(String) && !value.empty?
  end
  private_class_method :present?

  def self.like_pattern(term)
    "%#{ActiveRecord::Base.sanitize_sql_like(term.to_s)}%"
  end
  private_class_method :like_pattern

  def self.parse_iso_date(value)
    return nil unless present?(value)
    return nil unless value.match?(/\A\d{4}-\d{2}-\d{2}\z/)

    Date.iso8601(value)
  rescue ArgumentError
    nil
  end
  private_class_method :parse_iso_date

  def self.normalize_page(value)
    n = value.to_i
    n < 1 ? 1 : n
  end
  private_class_method :normalize_page

  def self.normalize_per_page(value)
    n = value.to_i
    return DEFAULT_PER_PAGE if n <= 0

    [ n, MAX_PER_PAGE ].min
  end
  private_class_method :normalize_per_page

  def self.normalize_sort_column(value)
    s = value.to_s
    SORTABLE_COLUMNS.include?(s) ? s.to_sym : :id
  end
  private_class_method :normalize_sort_column

  def self.normalize_sort_direction(value)
    s = value.to_s.downcase
    ALLOWED_SORT_DIRECTIONS.include?(s) ? s.to_sym : :asc
  end
  private_class_method :normalize_sort_direction
end
