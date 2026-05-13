import type { CSSProperties } from "react";
import type { OrderTreeNode } from "../types/tree";
import { TREE_COLUMNS } from "./treeColumns";

interface TreeTableRowProps {
  node: OrderTreeNode;
  index: number;
  expanded: boolean;
  onToggle: (id: string) => void;
  style?: CSSProperties;
}

const ROW_HEIGHT = 40;

const STATUS_COLORS: Record<string, string> = {
  受注確認: "bg-blue-100 text-blue-800",
  出荷準備中: "bg-yellow-100 text-yellow-800",
  出荷済み: "bg-green-100 text-green-800",
  納品完了: "bg-gray-100 text-gray-800",
  キャンセル: "bg-red-100 text-red-800",
};

const KIND_LABELS: Record<OrderTreeNode["kind"], string> = {
  customer: "顧客",
  product: "商品",
  order: "注文",
};

function formatCurrency(value: number): string {
  return `¥${value.toLocaleString()}`;
}

function formatQuantity(value: number): string {
  return value.toLocaleString();
}

function renderStatus(statuses: string[]) {
  if (statuses.length === 0) {
    return "";
  }

  if (statuses.length === 1) {
    const status = statuses[0];
    const colorClass = STATUS_COLORS[status] ?? "bg-gray-100 text-gray-800";
    return (
      <span className={`inline-block px-2 py-0.5 rounded text-xs font-medium ${colorClass}`}>
        {status}
      </span>
    );
  }

  return `${statuses.length}種類`;
}

function getCellContent(node: OrderTreeNode, key: string) {
  const order = node.order;

  switch (key) {
    case "kind":
      return KIND_LABELS[node.kind];
    case "order_date":
      return order?.order_date ?? `${node.summary.order_count.toLocaleString()}件`;
    case "customer_name":
      return order?.customer_name ?? (node.kind === "customer" ? node.label : "");
    case "product_name":
      return order?.product_name ?? (node.kind === "product" ? node.label : "");
    case "quantity":
      return formatQuantity(node.summary.quantity);
    case "unit_price":
      return order ? formatCurrency(order.unit_price) : "";
    case "total_amount":
      return formatCurrency(node.summary.total_amount);
    case "status":
      return renderStatus(node.summary.statuses);
    case "delivery_date":
      return order?.delivery_date ?? "";
    default:
      return "";
  }
}

export function TreeTableRow({ node, index, expanded, onToggle, style }: TreeTableRowProps) {
  const bgClass = index % 2 === 0 ? "bg-white" : "bg-gray-50";
  const canExpand = node.children.length > 0;
  const indent = Math.min(node.depth, 6) * 18;
  const rowTone =
    node.kind === "customer"
      ? "font-semibold text-gray-950"
      : node.kind === "product"
        ? "font-medium text-gray-800"
        : "text-gray-700";

  return (
    <div
      role="row"
      aria-rowindex={index + 2}
      aria-level={node.depth + 1}
      aria-expanded={canExpand ? expanded : undefined}
      className={`flex items-center text-sm border-b border-gray-200 min-w-max ${bgClass} ${rowTone} hover:bg-blue-50`}
      style={{ height: ROW_HEIGHT, ...style }}
    >
      {TREE_COLUMNS.map((col) => (
        <div
          key={col.key}
          role="gridcell"
          className="px-2 truncate"
          style={{ width: col.width, minWidth: col.width }}
        >
          {col.key === "label" ? (
            <div className="flex items-center min-w-0" style={{ paddingLeft: indent }}>
              {canExpand ? (
                <button
                  type="button"
                  aria-label={`${node.label}を${expanded ? "折りたたむ" : "展開"}`}
                  className="mr-2 flex h-6 w-6 shrink-0 items-center justify-center rounded border border-gray-300 bg-white text-xs text-gray-700 hover:bg-gray-100"
                  onClick={() => onToggle(node.id)}
                >
                  {expanded ? "▼" : "▶"}
                </button>
              ) : (
                <span className="mr-2 h-6 w-6 shrink-0" />
              )}
              <span className="truncate">{node.label}</span>
            </div>
          ) : (
            getCellContent(node, col.key)
          )}
        </div>
      ))}
    </div>
  );
}
