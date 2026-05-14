import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import { BigTable } from "../../src/components/BigTable";

const mockOrders = [
  {
    id: 1,
    order_number: "ORD-001",
    order_type: "受注",
    order_date: "2024-01-01",
    customer_name: "テスト顧客A",
    customer_code: "C001",
    product_name: "テスト商品A",
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
  {
    id: 2,
    order_number: "ORD-002",
    order_type: "発注",
    order_date: "2024-01-02",
    customer_name: "テスト顧客B",
    customer_code: "C002",
    product_name: "テスト商品B",
    product_code: "P002",
    quantity: 5,
    unit_price: 2000,
    total_amount: 10000,
    status: "出荷済み",
    delivery_date: "2024-01-16",
    notes: "",
    created_at: "2024-01-02T00:00:00Z",
    updated_at: "2024-01-02T00:00:00Z",
  },
];

beforeEach(() => {
  vi.restoreAllMocks();
  globalThis.fetch = vi.fn().mockResolvedValue({
    ok: true,
    json: () =>
      Promise.resolve({
        data: mockOrders,
        total: 2,
        page: 1,
        per_page: 25,
        total_pages: 1,
      }),
  });
});

describe("BigTable", () => {
  it("renders table headers", async () => {
    render(<BigTable />);

    await waitFor(() => {
      expect(screen.getByText("注文番号")).toBeInTheDocument();
    });

    expect(screen.getAllByText("種別").length).toBeGreaterThanOrEqual(1);
    expect(screen.getAllByText("注文日").length).toBeGreaterThanOrEqual(1);
    expect(screen.getAllByText("顧客名").length).toBeGreaterThanOrEqual(1);
    expect(screen.getAllByText("商品名").length).toBeGreaterThanOrEqual(1);
    expect(screen.getAllByText("数量").length).toBeGreaterThanOrEqual(1);
    expect(screen.getAllByText("単価").length).toBeGreaterThanOrEqual(1);
    expect(screen.getAllByText("金額").length).toBeGreaterThanOrEqual(1);
    expect(screen.getAllByText("ステータス").length).toBeGreaterThanOrEqual(1);
    expect(screen.getAllByText("納期").length).toBeGreaterThanOrEqual(1);
  });

  it("renders data rows after loading", async () => {
    render(<BigTable />);

    await waitFor(() => {
      expect(screen.getByText("ORD-001")).toBeInTheDocument();
    });

    expect(screen.getByText("ORD-002")).toBeInTheDocument();
    expect(screen.getByText("テスト顧客A")).toBeInTheDocument();
    expect(screen.getByText("テスト顧客B")).toBeInTheDocument();
  });

  it("displays total count", async () => {
    render(<BigTable />);

    await waitFor(() => {
      expect(screen.getByTestId("total-count")).toHaveTextContent("2");
    });
  });
});
