# typed: false

# Seeds the orders table with synthetic data so the deferred-join code path
# can be exercised end-to-end against a realistic row count.
#
# Knobs:
#   SEED_ORDERS=10000   # how many rows to insert (default 10_000)
#   SEED_RESET=true     # truncate before inserting (default false; idempotent skip otherwise)
#
# At default page_size=50, 10_000 rows is enough to push past the 10_000-OFFSET
# threshold via deep paging only via per_page=10. For load-style verification,
# bump SEED_ORDERS to 1_000_000.

require "securerandom"

target_count = Integer(ENV.fetch("SEED_ORDERS", "10000"))
reset        = ENV["SEED_RESET"] == "true"

if reset
  Order.delete_all
  ActiveRecord::Base.connection.execute("ALTER TABLE orders AUTO_INCREMENT = 1")
end

current = Order.count
needed  = target_count - current

if needed <= 0
  puts "[seed] orders already at #{current}, skipping (target=#{target_count})"
else
  puts "[seed] inserting #{needed} orders (current=#{current}, target=#{target_count})"
end

ORDER_TYPES   = %w[standard rush wholesale subscription return].freeze
STATUSES      = %w[pending confirmed shipped delivered cancelled].freeze
CUSTOMER_BASE = 200    # ~200 distinct customers
PRODUCT_BASE  = 80     # ~80 distinct products
BATCH_SIZE    = 1_000

now = Time.current

(0...[ needed, 0 ].max).each_slice(BATCH_SIZE) do |slice|
  rows = slice.map do |i|
    seq           = current + i + 1
    customer_idx  = seq % CUSTOMER_BASE
    product_idx   = seq % PRODUCT_BASE
    qty           = (seq % 50) + 1
    unit_price    = 100 + (seq % 9_900)
    total         = qty * unit_price
    order_date    = (now - (seq % 365).days).to_date
    delivery_date = order_date + ((seq % 14) + 1)

    {
      order_number:  format("ORD-%010d", seq),
      order_type:    ORDER_TYPES[seq % ORDER_TYPES.size],
      order_date:    order_date,
      customer_name: "Customer #{format('%04d', customer_idx)}",
      customer_code: format("CUST-%05d", customer_idx),
      product_name:  "Product #{format('%03d', product_idx)}",
      product_code:  format("PROD-%05d", product_idx),
      quantity:      qty,
      unit_price:    unit_price,
      total_amount:  total,
      status:        STATUSES[seq % STATUSES.size],
      delivery_date: delivery_date,
      notes:         seq.even? ? "" : "auto-seed ##{seq}",
      created_at:    now,
      updated_at:    now
    }
  end

  Order.insert_all(rows)
end

puts "[seed] done; orders=#{Order.count}"
