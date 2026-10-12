// Mirrors the Rails JSON shape:
//   - BigDecimal columns serialise as strings (preserves precision).
//   - Date / DateTime serialise as ISO8601 strings.
//   - delivery_date is `null` when the column is NULL.
//   - The list endpoint wraps results in `{ orders, meta }`.
export interface Order {
  id: number;
  order_number: string;
  order_type: string;
  order_date: string;
  customer_name: string;
  customer_code: string;
  product_name: string;
  product_code: string;
  quantity: number;
  unit_price: string;
  total_amount: string;
  status: string;
  delivery_date: string | null;
  notes: string;
  created_at: string;
  updated_at: string;
}

export interface OrdersMeta {
  total: number;
  page: number;
  per_page: number;
  total_pages: number;
}

export interface OrdersResponse {
  orders: Order[];
  meta: OrdersMeta;
}

export interface SortConfig {
  column: string;
  direction: "asc" | "desc";
}

export interface FilterConfig {
  order_type?: string;
  status?: string;
  customer_name?: string;
  product_name?: string;
  date_from?: string;
  date_to?: string;
}

export interface TableParams {
  page: number;
  perPage: number;
  sort: SortConfig;
  filters: FilterConfig;
}
