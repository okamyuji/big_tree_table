import { describe, it, expect, vi, beforeEach } from "vitest";
import { renderHook, act, waitFor } from "@testing-library/react";
import { useTableData } from "../../src/hooks/useTableData";

const mockResponse = {
  data: [
    {
      id: 1,
      order_number: "ORD-001",
      order_type: "受注",
      order_date: "2024-01-01",
      customer_name: "テスト顧客",
      customer_code: "C001",
      product_name: "テスト商品",
      product_code: "P001",
      quantity: 10,
      unit_price: 1000,
      total_amount: 10000,
      status: "受注確認",
      delivery_date: "2024-01-15",
      notes: "",
      created_at: "2024-01-01T00:00:00Z",
      updated_at: "2024-01-01T00:00:00Z",
    },
  ],
  total: 100,
  page: 1,
  per_page: 25,
  total_pages: 4,
};

beforeEach(() => {
  vi.restoreAllMocks();
});

function mockFetch() {
  const fetchMock = vi.fn().mockResolvedValue({
    ok: true,
    json: () => Promise.resolve(mockResponse),
  });
  globalThis.fetch = fetchMock;
  return fetchMock;
}

describe("useTableData", () => {
  it("fetches data on initial load", async () => {
    const fetchMock = mockFetch();

    const { result } = renderHook(() => useTableData());

    expect(result.current.loading).toBe(true);

    await waitFor(() => {
      expect(result.current.loading).toBe(false);
    });

    expect(result.current.data).toEqual(mockResponse.data);
    expect(result.current.total).toBe(100);
    expect(fetchMock).toHaveBeenCalledTimes(1);
  });

  it("sends correct params on sort change", async () => {
    const fetchMock = mockFetch();

    const { result } = renderHook(() => useTableData());

    await waitFor(() => {
      expect(result.current.loading).toBe(false);
    });

    act(() => {
      result.current.setSort({ column: "customer_name", direction: "asc" });
    });

    await waitFor(() => {
      expect(result.current.loading).toBe(false);
    });

    const lastCall = fetchMock.mock.calls[fetchMock.mock.calls.length - 1];
    const url = lastCall[0] as string;
    expect(url).toContain("sort=customer_name");
    expect(url).toContain("order=asc");
  });

  it("sends correct params on page change", async () => {
    const fetchMock = mockFetch();

    const { result } = renderHook(() => useTableData());

    await waitFor(() => {
      expect(result.current.loading).toBe(false);
    });

    act(() => {
      result.current.setPage(3);
    });

    await waitFor(() => {
      expect(result.current.loading).toBe(false);
    });

    const lastCall = fetchMock.mock.calls[fetchMock.mock.calls.length - 1];
    const url = lastCall[0] as string;
    expect(url).toContain("page=3");
  });

  it("resets to page 1 on filter change", async () => {
    const fetchMock = mockFetch();

    const { result } = renderHook(() => useTableData());

    await waitFor(() => {
      expect(result.current.loading).toBe(false);
    });

    // First go to page 3
    act(() => {
      result.current.setPage(3);
    });

    await waitFor(() => {
      expect(result.current.loading).toBe(false);
    });

    // Now set filters - should reset to page 1
    act(() => {
      result.current.setFilters({ status: "受注確認" });
    });

    expect(result.current.page).toBe(1);

    await waitFor(() => {
      expect(result.current.loading).toBe(false);
    });

    const lastCall = fetchMock.mock.calls[fetchMock.mock.calls.length - 1];
    const url = lastCall[0] as string;
    expect(url).toContain("page=1");
    expect(url).toContain("status=%E5%8F%97%E6%B3%A8%E7%A2%BA%E8%AA%8D");
  });
});
