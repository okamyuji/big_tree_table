import type { SortConfig } from "../types/order";
import { TREE_COLUMNS } from "./treeColumns";

interface TreeTableHeaderProps {
  sort: SortConfig;
  onSort: (column: string) => void;
}

export function TreeTableHeader({ sort, onSort }: TreeTableHeaderProps) {
  const getSortIndicator = (column?: string) => {
    if (!column || sort.column !== column) return null;
    return sort.direction === "asc" ? " ▲" : " ▼";
  };

  const getAriaSort = (column?: string): "ascending" | "descending" | "none" => {
    if (!column || sort.column !== column) return "none";
    return sort.direction === "asc" ? "ascending" : "descending";
  };

  return (
    <div
      role="row"
      className="flex bg-gray-100 border-b border-gray-300 font-semibold text-sm sticky top-0 z-10 min-w-max"
    >
      {TREE_COLUMNS.map((col) => {
        const sortKey = col.sortKey ?? col.key;
        const sortable = col.sortable === true;

        return (
          <div
            key={col.key}
            role="columnheader"
            aria-sort={sortable ? getAriaSort(sortKey) : "none"}
            tabIndex={sortable ? 0 : undefined}
            className={`px-2 py-2 truncate ${
              sortable ? "cursor-pointer select-none hover:bg-gray-200" : ""
            }`}
            style={{ width: col.width, minWidth: col.width }}
            onClick={sortable ? () => onSort(sortKey) : undefined}
            onKeyDown={
              sortable
                ? (e) => {
                    if (e.key === "Enter" || e.key === " ") {
                      e.preventDefault();
                      onSort(sortKey);
                    }
                  }
                : undefined
            }
          >
            {col.label}
            {getSortIndicator(sortable ? sortKey : undefined)}
          </div>
        );
      })}
    </div>
  );
}
