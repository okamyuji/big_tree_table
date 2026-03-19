import { useState, useCallback, useMemo } from "react";
import type { FilterConfig, SortConfig } from "../types/order";
import { useTableData } from "../hooks/useTableData";
import { VirtualScroller } from "./VirtualScroller";
import { TableHeader } from "./TableHeader";
import { TableRow } from "./TableRow";
import { ColumnFilter } from "./ColumnFilter";
import { DateRangeFilter } from "./DateRangeFilter";
import { Pagination } from "./Pagination";

const ROW_HEIGHT = 40;
const TABLE_HEIGHT = 600;

const ORDER_TYPE_OPTIONS = ["通常注文", "急ぎ注文", "定期注文", "特別注文"];
const STATUS_OPTIONS = ["受注確認", "出荷準備中", "出荷済み", "納品完了", "キャンセル"];

export function BigTable() {
  const {
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
  } = useTableData();

  const [localFilters, setLocalFilters] = useState<FilterConfig>({});

  const handleSort = useCallback(
    (column: string) => {
      setSort((prev: SortConfig) => ({
        column,
        direction: prev.column === column && prev.direction === "asc" ? "desc" : "asc",
      }));
    },
    [setSort],
  );

  const handleApplyFilters = useCallback(() => {
    setFilters(localFilters);
  }, [setFilters, localFilters]);

  const handleClearFilters = useCallback(() => {
    setLocalFilters({});
    setFilters({});
  }, [setFilters]);

  const updateLocalFilter = useCallback((key: keyof FilterConfig, value: string) => {
    setLocalFilters((prev) => ({ ...prev, [key]: value }));
  }, []);

  const header = useMemo(() => <TableHeader sort={sort} onSort={handleSort} />, [sort, handleSort]);

  const renderItem = useCallback(
    (index: number) => {
      const order = data[index];
      if (!order) return null;
      return <TableRow key={order.id} order={order} index={index} />;
    },
    [data],
  );

  return (
    <div className="max-w-full mx-auto p-4">
      <h1 className="text-2xl font-bold mb-4">受注管理テーブル</h1>

      {/* Filters */}
      <div className="bg-white rounded-lg shadow p-4 mb-4">
        <div className="flex flex-wrap gap-4 items-end">
          <ColumnFilter
            label="種別"
            value={localFilters.order_type ?? ""}
            onChange={(v) => updateLocalFilter("order_type", v)}
            options={ORDER_TYPE_OPTIONS}
          />
          <ColumnFilter
            label="ステータス"
            value={localFilters.status ?? ""}
            onChange={(v) => updateLocalFilter("status", v)}
            options={STATUS_OPTIONS}
          />
          <ColumnFilter
            label="顧客名"
            value={localFilters.customer_name ?? ""}
            onChange={(v) => updateLocalFilter("customer_name", v)}
            placeholder="顧客名で検索"
          />
          <ColumnFilter
            label="商品名"
            value={localFilters.product_name ?? ""}
            onChange={(v) => updateLocalFilter("product_name", v)}
            placeholder="商品名で検索"
          />
          <DateRangeFilter
            dateFrom={localFilters.date_from ?? ""}
            dateTo={localFilters.date_to ?? ""}
            onDateFromChange={(v) => updateLocalFilter("date_from", v)}
            onDateToChange={(v) => updateLocalFilter("date_to", v)}
          />
          <div className="flex gap-2">
            <button
              onClick={handleApplyFilters}
              className="px-4 py-1.5 bg-blue-600 text-white rounded text-sm hover:bg-blue-700"
            >
              適用
            </button>
            <button
              onClick={handleClearFilters}
              className="px-4 py-1.5 bg-gray-200 text-gray-700 rounded text-sm hover:bg-gray-300"
            >
              クリア
            </button>
          </div>
        </div>
      </div>

      {/* Error */}
      {error && (
        <div role="alert" className="bg-red-100 text-red-700 p-3 rounded mb-4">
          {error}
        </div>
      )}

      {/* Table */}
      <div className="bg-white rounded-lg shadow overflow-hidden">
        {loading && (
          <div className="flex justify-center items-center py-8">
            <div
              className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600"
              role="status"
            >
              <span className="sr-only">読み込み中...</span>
            </div>
          </div>
        )}

        {!loading && (
          <VirtualScroller
            itemCount={data.length}
            itemHeight={ROW_HEIGHT}
            containerHeight={TABLE_HEIGHT}
            header={header}
            renderItem={renderItem}
          />
        )}

        <Pagination
          page={page}
          totalPages={totalPages}
          total={total}
          perPage={perPage}
          onPageChange={setPage}
          onPerPageChange={setPerPage}
        />
      </div>
    </div>
  );
}
