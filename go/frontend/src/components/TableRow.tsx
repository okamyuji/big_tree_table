import type { Order } from "../types/order";
import { COLUMNS } from "./TableHeader";

interface TableRowProps {
  order: Order;
  index: number;
  style?: React.CSSProperties;
}

const STATUS_COLORS: Record<string, string> = {
  受注確認: "bg-blue-100 text-blue-800",
  出荷準備中: "bg-yellow-100 text-yellow-800",
  出荷済み: "bg-green-100 text-green-800",
  納品完了: "bg-gray-100 text-gray-800",
  キャンセル: "bg-red-100 text-red-800",
};

function formatCurrency(value: number): string {
  return `¥${value.toLocaleString()}`;
}

function formatQuantity(value: number): string {
  return value.toLocaleString();
}

function getCellContent(order: Order, key: string): React.ReactNode {
  switch (key) {
    case "quantity":
      return formatQuantity(order.quantity);
    case "unit_price":
      return formatCurrency(order.unit_price);
    case "total_amount":
      return formatCurrency(order.total_amount);
    case "status": {
      const colorClass = STATUS_COLORS[order.status] ?? "bg-gray-100 text-gray-800";
      return (
        <span className={`inline-block px-2 py-0.5 rounded text-xs font-medium ${colorClass}`}>
          {order.status}
        </span>
      );
    }
    default:
      return String(order[key as keyof Order] ?? "");
  }
}

export function TableRow({ order, index, style }: TableRowProps) {
  const bgClass = index % 2 === 0 ? "bg-white" : "bg-gray-50";

  return (
    <div
      role="row"
      aria-rowindex={index + 2}
      className={`flex items-center text-sm border-b border-gray-200 ${bgClass} hover:bg-blue-50`}
      style={{ height: 40, ...style }}
    >
      {COLUMNS.map((col) => (
        <div
          key={col.key}
          role="gridcell"
          className="px-2 truncate"
          style={{ width: col.width, minWidth: col.width }}
        >
          {getCellContent(order, col.key)}
        </div>
      ))}
    </div>
  );
}
