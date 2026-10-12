import { useState, useEffect, useCallback } from "react";
import type { SortConfig, FilterConfig } from "../types/order";
import type { OrderTreeNode } from "../types/tree";
import { fetchOrderTree } from "../api/orderTree";

export function useTreeTableData() {
  const [data, setData] = useState<OrderTreeNode[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [total, setTotal] = useState(0);
  const [totalPages, setTotalPages] = useState(0);
  const [page, setPage] = useState(1);
  const [perPage, setPerPageState] = useState(25);
  const [sort, setSort] = useState<SortConfig>({
    column: "order_date",
    direction: "desc",
  });
  const [filters, setFiltersState] = useState<FilterConfig>({});

  const setFilters = useCallback((newFilters: FilterConfig) => {
    setFiltersState(newFilters);
    setPage(1);
  }, []);

  const setPerPage = useCallback((newPerPage: number) => {
    setPerPageState(newPerPage);
    setPage(1);
  }, []);

  useEffect(() => {
    const abortController = new AbortController();

    const load = async () => {
      setLoading(true);
      setError(null);
      try {
        const result = await fetchOrderTree(
          { page, perPage, sort, filters },
          abortController.signal,
        );
        setData(result.data);
        setTotal(result.total);
        setTotalPages(result.total_pages);
      } catch (err) {
        if (err instanceof DOMException && err.name === "AbortError") {
          return;
        }
        setError(err instanceof Error ? err.message : "An unknown error occurred");
      } finally {
        if (!abortController.signal.aborted) {
          setLoading(false);
        }
      }
    };

    void load();

    return () => {
      abortController.abort();
    };
  }, [page, perPage, sort, filters]);

  return {
    data,
    loading,
    error,
    total,
    totalPages,
    page,
    perPage,
    sort,
    setPage,
    setPerPage,
    setSort,
    setFilters,
  };
}
