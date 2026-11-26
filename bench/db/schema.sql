-- rails/db/migrate と同じ列と index を PostgreSQL で定義する。3者のベンチが共有する。
DROP TABLE IF EXISTS orders;

CREATE TABLE orders (
  id            bigserial PRIMARY KEY,
  order_number  varchar(64)    NOT NULL,
  order_type    varchar(32)    NOT NULL,
  order_date    date           NOT NULL,
  customer_name varchar(128)   NOT NULL,
  customer_code varchar(64)    NOT NULL,
  product_name  varchar(128)   NOT NULL,
  product_code  varchar(64)    NOT NULL,
  quantity      integer        NOT NULL,
  unit_price    numeric(12, 2) NOT NULL,
  total_amount  numeric(14, 2) NOT NULL,
  status        varchar(32)    NOT NULL,
  delivery_date date,
  notes         varchar(255)   NOT NULL DEFAULT '',
  created_at    timestamp(6)   NOT NULL,
  updated_at    timestamp(6)   NOT NULL
);

CREATE INDEX idx_order_number  ON orders (order_number);
CREATE INDEX idx_order_type    ON orders (order_type);
CREATE INDEX idx_order_date    ON orders (order_date);
CREATE INDEX idx_customer_name ON orders (customer_name);
CREATE INDEX idx_product_name  ON orders (product_name);
CREATE INDEX idx_status        ON orders (status);
