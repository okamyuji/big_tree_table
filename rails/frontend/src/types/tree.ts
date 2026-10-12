import type { Order, OrdersMeta } from "./order";

export type TreeNodeKind = "customer" | "product" | "order";

export interface TreeSummary {
  order_count: number;
  quantity: number;
  // BigDecimal serialised as string by Rails (matches Order.unit_price/total_amount).
  total_amount: string;
  statuses: string[];
}

export interface OrderTreeNode {
  id: string;
  kind: TreeNodeKind;
  depth: number;
  label: string;
  order?: Order;
  summary: TreeSummary;
  children: OrderTreeNode[];
}

export interface OrderTreeResponse {
  tree: OrderTreeNode[];
  meta: OrdersMeta;
}
