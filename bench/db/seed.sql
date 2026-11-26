-- rails/db/seeds.rb と同じ式で行を作る。now を固定し、何度流しても同じデータになるようにする。
-- 行数は psql 変数 seed_orders で渡す。
INSERT INTO orders (
  order_number, order_type, order_date, customer_name, customer_code,
  product_name, product_code, quantity, unit_price, total_amount,
  status, delivery_date, notes, created_at, updated_at
)
SELECT
  'ORD-' || lpad(seq::text, 10, '0'),
  (ARRAY['standard','rush','wholesale','subscription','return'])[seq % 5 + 1],
  d.order_date,
  'Customer ' || lpad((seq % 200)::text, 4, '0'),
  'CUST-' || lpad((seq % 200)::text, 5, '0'),
  'Product ' || lpad((seq % 80)::text, 3, '0'),
  'PROD-' || lpad((seq % 80)::text, 5, '0'),
  seq % 50 + 1,
  100 + seq % 9900,
  (seq % 50 + 1) * (100 + seq % 9900),
  (ARRAY['pending','confirmed','shipped','delivered','cancelled'])[seq % 5 + 1],
  d.order_date + (seq % 14 + 1),
  CASE WHEN seq % 2 = 0 THEN '' ELSE 'auto-seed #' || seq END,
  timestamp '2026-11-01 00:00:00',
  timestamp '2026-11-01 00:00:00'
FROM generate_series(1, :seed_orders) AS seq
CROSS JOIN LATERAL (SELECT date '2026-11-01' - (seq % 365) AS order_date) AS d
ORDER BY seq;

ANALYZE orders;
