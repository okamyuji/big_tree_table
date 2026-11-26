-- parity 用の追加行。seed の単価は整数だけなので、端数の金額と、LIKE の特殊文字や
-- 引用符と ? を含む名前、NULL の配送日をここで足す。同じ顧客・商品にまとめ、ツリーの合計で端数を足し合わせる。
INSERT INTO orders (
  order_number, order_type, order_date, customer_name, customer_code,
  product_name, product_code, quantity, unit_price, total_amount,
  status, delivery_date, notes, created_at, updated_at
) VALUES
  ('ORD-X1', 'rush', '2026-02-28', 'Odd 100% _x\y O''Brien ?', 'CUST-ODD', 'Widget ?', 'PROD-ODD', 1, 0.10, 0.10, 'shipped', '2026-03-01', 'odd ?', '2026-11-01 12:34:56.789', '2026-11-01 12:34:56.789'),
  ('ORD-X2', 'rush', '2026-02-28', 'Odd 100% _x\y O''Brien ?', 'CUST-ODD', 'Widget ?', 'PROD-ODD', 2, 0.10, 0.20, 'pending', NULL, '', '2026-11-01 12:34:56.000', '2026-11-01 12:34:56.001'),
  ('ORD-X3', 'return', '2024-02-29', 'Odd 100% _x\y O''Brien ?', 'CUST-ODD', 'Gadget', 'PROD-ODD2', 3, 4.15, 12.45, 'cancelled', '2024-03-01', '', '2026-11-01 00:00:00', '2026-11-01 00:00:00'),
  ('ORD-X4', 'rush', '2026-01-15', 'Neg Cust', 'CUST-NEG', 'Refund', 'PROD-NEG', 1, -0.50, -0.50, 'cancelled', '2026-01-16', '', '2026-11-01 00:00:00', '2026-11-01 00:00:00'),
  ('ORD-X5', 'rush', '2026-01-15', 'Big Cust', 'CUST-BIG', 'Big', 'PROD-BIG', 1, 9999999999.99, 999999999999.99, 'shipped', '2026-01-16', '', '2026-11-01 00:00:00', '2026-11-01 00:00:00');
ANALYZE orders;
