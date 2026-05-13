import type { OrderTreeNode } from "../types/tree";

export function flattenVisibleTree(
  nodes: OrderTreeNode[],
  expandedNodeIds: ReadonlySet<string>,
): OrderTreeNode[] {
  const visibleNodes: OrderTreeNode[] = [];

  const visit = (node: OrderTreeNode) => {
    visibleNodes.push(node);
    if (!expandedNodeIds.has(node.id)) {
      return;
    }
    for (const child of node.children) {
      visit(child);
    }
  };

  for (const node of nodes) {
    visit(node);
  }

  return visibleNodes;
}

export function collectExpandableNodeIds(nodes: OrderTreeNode[]): Set<string> {
  const ids = new Set<string>();

  const visit = (node: OrderTreeNode) => {
    if (node.children.length > 0) {
      ids.add(node.id);
    }
    for (const child of node.children) {
      visit(child);
    }
  };

  for (const node of nodes) {
    visit(node);
  }

  return ids;
}
