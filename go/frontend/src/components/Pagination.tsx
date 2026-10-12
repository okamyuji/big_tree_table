interface PaginationProps {
  page: number;
  totalPages: number;
  total: number;
  perPage: number;
  onPageChange: (page: number) => void;
  onPerPageChange: (perPage: number) => void;
}

function getPageNumbers(current: number, totalPages: number): (number | "...")[] {
  if (totalPages <= 7) {
    return Array.from({ length: totalPages }, (_, i) => i + 1);
  }

  const pages: (number | "...")[] = [1];

  if (current > 3) {
    pages.push("...");
  }

  const start = Math.max(2, current - 1);
  const end = Math.min(totalPages - 1, current + 1);

  for (let i = start; i <= end; i++) {
    pages.push(i);
  }

  if (current < totalPages - 2) {
    pages.push("...");
  }

  pages.push(totalPages);

  return pages;
}

export function Pagination({
  page,
  totalPages,
  total,
  perPage,
  onPageChange,
  onPerPageChange,
}: PaginationProps) {
  const pageNumbers = getPageNumbers(page, totalPages);

  return (
    <div className="flex items-center justify-between py-3 px-4">
      <div className="text-sm text-gray-700">
        全 <span data-testid="total-count">{total.toLocaleString()}</span> 件
      </div>

      <div className="flex items-center gap-2">
        <label htmlFor="per-page" className="text-sm text-gray-700">
          表示件数:
        </label>
        <select
          id="per-page"
          value={perPage}
          onChange={(e) => onPerPageChange(Number(e.target.value))}
          className="border border-gray-300 rounded px-2 py-1 text-sm"
        >
          {[10, 25, 50, 100].map((n) => (
            <option key={n} value={n}>
              {n}
            </option>
          ))}
        </select>
      </div>

      <nav className="flex items-center gap-1" aria-label="ページナビゲーション">
        <button
          aria-label="前のページ"
          disabled={page <= 1}
          onClick={() => onPageChange(page - 1)}
          className="px-3 py-1 text-sm border border-gray-300 rounded disabled:opacity-50 disabled:cursor-not-allowed hover:bg-gray-100"
        >
          前へ
        </button>

        {pageNumbers.map((p, i) =>
          p === "..." ? (
            <span key={`ellipsis-${i}`} className="px-2 text-gray-500">
              ...
            </span>
          ) : (
            <button
              key={p}
              aria-current={p === page ? "page" : undefined}
              onClick={() => onPageChange(p)}
              className={`px-3 py-1 text-sm border rounded ${
                p === page
                  ? "bg-blue-600 text-white border-blue-600"
                  : "border-gray-300 hover:bg-gray-100"
              }`}
            >
              {p}
            </button>
          ),
        )}

        <button
          aria-label="次のページ"
          disabled={page >= totalPages}
          onClick={() => onPageChange(page + 1)}
          className="px-3 py-1 text-sm border border-gray-300 rounded disabled:opacity-50 disabled:cursor-not-allowed hover:bg-gray-100"
        >
          次へ
        </button>
      </nav>
    </div>
  );
}
