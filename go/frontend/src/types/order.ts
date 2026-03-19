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
  unit_price: number;
  total_amount: number;
  status: string;
  delivery_date: string;
  notes: string;
  created_at: string;
  updated_at: string;
}

export interface OrdersResponse {
  data: Order[];
  total: number;
  page: number;
  per_page: number;
  total_pages: number;
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
