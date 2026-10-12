import { describe, expect, it } from "vitest";
import { collectExpandableNodeIds, flattenVisibleTree } from "../../src/utils/treeData";
import type { OrderTreeNode } from "../../src/types/tree";

const tree: OrderTreeNode[] = [
  {
    id: "customer:C001",
    kind: "customer",
    depth: 0,
    label: "顧客A",
    summary: { order_count: 2, quantity: 5, total_amount: 7500, statuses: ["受注確認"] },
    children: [
      {
        id: "customer:C001:product:P001",
        kind: "product",
        depth: 1,
        label: "商品A",
        summary: { order_count: 2, quantity: 5, total_amount: 7500, statuses: ["受注確認"] },
        children: [
          {
            id: "order:1",
            kind: "order",
            depth: 2,
            label: "ORD-001",
            summary: { order_count: 1, quantity: 2, total_amount: 3000, statuses: ["受注確認"] },
            children: [],
          },
        ],
      },
    ],
  },
];

describe("treeData", () => {
  it("flattens only root nodes when collapsed", () => {
    const visible = flattenVisibleTree(tree, new Set());
    expect(visible.map((node) => node.id)).toEqual(["customer:C001"]);
  });

  it("flattens expanded descendants in depth-first order", () => {
    const visible = flattenVisibleTree(
      tree,
      new Set(["customer:C001", "customer:C001:product:P001"]),
    );
    expect(visible.map((node) => node.id)).toEqual([
      "customer:C001",
      "customer:C001:product:P001",
      "order:1",
    ]);
  });

  it("collects expandable node ids", () => {
    expect([...collectExpandableNodeIds(tree)]).toEqual([
      "customer:C001",
      "customer:C001:product:P001",
    ]);
  });
});
