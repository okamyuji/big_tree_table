import type { SortConfig } from "../types/order";
import { COLUMNS } from "./columns";

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
