import type { Order } from "./order";

export type TreeNodeKind = "customer" | "product" | "order";

export interface TreeSummary {
  order_count: number;
  quantity: number;
  total_amount: number;
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
  data: OrderTreeNode[];
  total: number;
  page: number;
  per_page: number;
  total_pages: number;
}
