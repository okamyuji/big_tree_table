import { describe, it, expect, vi, beforeEach } from "vitest";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { TreeTable } from "../../src/components/TreeTable";

const mockTree = [
  {
    id: "customer:C001",
    kind: "customer",
    depth: 0,
    label: "テスト顧客A",
    summary: { order_count: 1, quantity: 10, total_amount: 10000, statuses: ["受注確認"] },
    children: [
      {
        id: "customer:C001:product:P001",
        kind: "product",
        depth: 1,
        label: "テスト商品A",
        summary: { order_count: 1, quantity: 10, total_amount: 10000, statuses: ["受注確認"] },
        children: [
          {
            id: "order:1",
            kind: "order",
            depth: 2,
            label: "ORD-001",
            order: {
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
            summary: { order_count: 1, quantity: 10, total_amount: 10000, statuses: ["受注確認"] },
            children: [],
          },
        ],
      },
    ],
  },
];

beforeEach(() => {
  vi.restoreAllMocks();
  globalThis.fetch = vi.fn().mockResolvedValue({
    ok: true,
    json: () =>
      Promise.resolve({
        data: mockTree,
        total: 1,
        page: 1,
        per_page: 25,
        total_pages: 1,
      }),
  });
});

describe("TreeTable", () => {
  it("fetches tree endpoint and renders root nodes", async () => {
    render(<TreeTable />);

    await waitFor(() => {
      expect(screen.getAllByText("テスト顧客A").length).toBeGreaterThan(0);
    });

    expect(globalThis.fetch).toHaveBeenCalledWith(
      expect.stringContaining("/api/order-tree?"),
      expect.any(Object),
    );
  });

  it("expands customer and product rows", async () => {
    render(<TreeTable />);

    const customerButton = await screen.findByRole("button", { name: "テスト顧客Aを展開" });
    fireEvent.click(customerButton);

    const productButton = await screen.findByRole("button", { name: "テスト商品Aを展開" });
    fireEvent.click(productButton);

    expect(screen.getByText("ORD-001")).toBeInTheDocument();
  });
});
