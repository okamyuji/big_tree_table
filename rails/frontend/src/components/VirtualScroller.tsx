import type { ReactNode } from "react";
import { useVirtualScroll } from "../hooks/useVirtualScroll";

interface VirtualScrollerProps {
  itemCount: number;
  itemHeight: number;
  containerHeight: number;
  overscan?: number;
  renderItem: (index: number) => ReactNode;
  header?: ReactNode;
}

export function VirtualScroller({
  itemCount,
  itemHeight,
  containerHeight,
  overscan,
  renderItem,
  header,
}: VirtualScrollerProps) {
  const { visibleRange, totalHeight, offsetY, onScroll } = useVirtualScroll({
    itemCount,
    itemHeight,
    containerHeight,
    overscan,
  });

  const items: ReactNode[] = [];
  for (let i = visibleRange.start; i < visibleRange.end; i++) {
    items.push(renderItem(i));
  }

  return (
    <div
      role="grid"
      aria-rowcount={itemCount}
      style={{ height: containerHeight, overflow: "auto" }}
      onScroll={onScroll}
    >
      {header}
      <div style={{ height: totalHeight, position: "relative" }}>
        <div style={{ transform: `translateY(${offsetY}px)` }}>{items}</div>
      </div>
    </div>
  );
}
