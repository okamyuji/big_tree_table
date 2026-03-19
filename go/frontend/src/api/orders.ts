import type { OrdersResponse, TableParams } from "../types/order";

export async function fetchOrders(
  params: TableParams,
  signal?: AbortSignal,
): Promise<OrdersResponse> {
  const searchParams = new URLSearchParams();

  searchParams.set("page", String(params.page));
  searchParams.set("per_page", String(params.perPage));
  searchParams.set("sort_column", params.sort.column);
  searchParams.set("sort_direction", params.sort.direction);

  const { filters } = params;
  if (filters.order_type) searchParams.set("order_type", filters.order_type);
  if (filters.status) searchParams.set("status", filters.status);
  if (filters.customer_name) searchParams.set("customer_name", filters.customer_name);
  if (filters.product_name) searchParams.set("product_name", filters.product_name);
  if (filters.date_from) searchParams.set("date_from", filters.date_from);
  if (filters.date_to) searchParams.set("date_to", filters.date_to);

  const response = await fetch(`/api/orders?${searchParams.toString()}`, {
    signal,
  });

  if (!response.ok) {
    throw new Error(`Failed to fetch orders: ${response.status}`);
  }

  return response.json() as Promise<OrdersResponse>;
}
