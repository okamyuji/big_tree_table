import type { SortConfig } from "../types/order";

export interface Column {
  key: string;
  label: string;
  width: string;
}

export const COLUMNS: Column[] = [
  { key: "order_number", label: "注文番号", width: "110px" },
  { key: "order_type", label: "種別", width: "80px" },
  { key: "order_date", label: "注文日", width: "110px" },
  { key: "customer_name", label: "顧客名", width: "150px" },
  { key: "product_name", label: "商品名", width: "180px" },
  { key: "quantity", label: "数量", width: "80px" },
  { key: "unit_price", label: "単価", width: "100px" },
  { key: "total_amount", label: "金額", width: "120px" },
  { key: "status", label: "ステータス", width: "110px" },
  { key: "delivery_date", label: "納期", width: "110px" },
];

interface TableHeaderProps {
  sort: SortConfig;
  onSort: (column: string) => void;
}

export function TableHeader({ sort, onSort }: TableHeaderProps) {
  const getSortIndicator = (column: string) => {
    if (sort.column !== column) return null;
    return sort.direction === "asc" ? " ▲" : " ▼";
  };

  const getAriaSort = (column: string): "ascending" | "descending" | "none" => {
    if (sort.column !== column) return "none";
    return sort.direction === "asc" ? "ascending" : "descending";
  };

  return (
    <div
      role="row"
      className="flex bg-gray-100 border-b border-gray-300 font-semibold text-sm sticky top-0 z-10"
    >
      {COLUMNS.map((col) => (
        <div
          key={col.key}
          role="columnheader"
          aria-sort={getAriaSort(col.key)}
          tabIndex={0}
          className="px-2 py-2 cursor-pointer select-none hover:bg-gray-200 truncate"
          style={{ width: col.width, minWidth: col.width }}
          onClick={() => onSort(col.key)}
          onKeyDown={(e) => {
            if (e.key === "Enter" || e.key === " ") {
              e.preventDefault();
              onSort(col.key);
            }
          }}
        >
          {col.label}
          {getSortIndicator(col.key)}
        </div>
      ))}
    </div>
  );
}
