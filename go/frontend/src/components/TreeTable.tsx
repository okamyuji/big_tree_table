import { useState, useCallback, useMemo } from "react";
import type { FilterConfig, SortConfig } from "../types/order";
import { useTreeTableData } from "../hooks/useTreeTableData";
import { VirtualScroller } from "./VirtualScroller";
import { TreeTableHeader } from "./TreeTableHeader";
import { TreeTableRow } from "./TreeTableRow";
import { ColumnFilter } from "./ColumnFilter";
import { DateRangeFilter } from "./DateRangeFilter";
import { Pagination } from "./Pagination";
import { collectExpandableNodeIds, flattenVisibleTree } from "../utils/treeData";

const ROW_HEIGHT = 40;
const TABLE_HEIGHT = 600;

const ORDER_TYPE_OPTIONS = ["受注", "発注"];
const STATUS_OPTIONS = ["受注確認", "出荷準備中", "出荷済み", "納品完了", "キャンセル"];

export function TreeTable() {
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
  } = useTreeTableData();

  const [localFilters, setLocalFilters] = useState<FilterConfig>({});
  const [expandedNodeIds, setExpandedNodeIds] = useState<Set<string>>(() => new Set());

  const tree = data;
  const expandableNodeIds = useMemo(() => collectExpandableNodeIds(tree), [tree]);
  const activeExpandedNodeIds = useMemo(
    () => new Set([...expandedNodeIds].filter((id) => expandableNodeIds.has(id))),
    [expandedNodeIds, expandableNodeIds],
  );
  const visibleNodes = useMemo(
    () => flattenVisibleTree(tree, activeExpandedNodeIds),
    [tree, activeExpandedNodeIds],
  );

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

  const toggleNode = useCallback((id: string) => {
    setExpandedNodeIds((prev) => {
      const next = new Set(prev);
      if (next.has(id)) {
        next.delete(id);
      } else {
        next.add(id);
      }
      return next;
    });
  }, []);

  const expandAll = useCallback(() => {
    setExpandedNodeIds(new Set(expandableNodeIds));
  }, [expandableNodeIds]);

  const collapseAll = useCallback(() => {
    setExpandedNodeIds(new Set());
  }, []);

  const header = useMemo(
    () => <TreeTableHeader sort={sort} onSort={handleSort} />,
    [sort, handleSort],
  );

  const renderItem = useCallback(
    (index: number) => {
      const node = visibleNodes[index];
      if (!node) return null;
      return (
        <TreeTableRow
          key={node.id}
          node={node}
          index={index}
          expanded={activeExpandedNodeIds.has(node.id)}
          onToggle={toggleNode}
        />
      );
    },
    [visibleNodes, activeExpandedNodeIds, toggleNode],
  );

  return (
    <div className="max-w-full mx-auto p-4">
      <div className="mb-4 flex flex-wrap items-end justify-between gap-3">
        <div>
          <h1 className="text-2xl font-bold">受注管理ツリーテーブル</h1>
          <p className="mt-1 text-sm text-gray-600">
            現在ページの受注を顧客、商品、注文の順に階層表示
          </p>
        </div>
        <div className="flex gap-2">
          <button
            type="button"
            onClick={expandAll}
            className="px-3 py-1.5 bg-blue-600 text-white rounded text-sm hover:bg-blue-700"
          >
            すべて展開
          </button>
          <button
            type="button"
            onClick={collapseAll}
            className="px-3 py-1.5 bg-gray-200 text-gray-700 rounded text-sm hover:bg-gray-300"
          >
            すべて折りたたみ
          </button>
        </div>
      </div>

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
              type="button"
              onClick={handleApplyFilters}
              className="px-4 py-1.5 bg-blue-600 text-white rounded text-sm hover:bg-blue-700"
            >
              適用
            </button>
            <button
              type="button"
              onClick={handleClearFilters}
              className="px-4 py-1.5 bg-gray-200 text-gray-700 rounded text-sm hover:bg-gray-300"
            >
              クリア
            </button>
          </div>
        </div>
      </div>

      {error && (
        <div role="alert" className="bg-red-100 text-red-700 p-3 rounded mb-4">
          {error}
        </div>
      )}

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
            itemCount={visibleNodes.length}
            itemHeight={ROW_HEIGHT}
            containerHeight={TABLE_HEIGHT}
            header={header}
            renderItem={renderItem}
          />
        )}

        <div className="border-t border-gray-200 px-4 py-2 text-xs text-gray-600">
          現在ページ: {tree.length.toLocaleString()} 顧客 / {visibleNodes.length.toLocaleString()}{" "}
          表示行
        </div>

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
