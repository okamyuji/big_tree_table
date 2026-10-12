import { useState, useMemo, useCallback } from "react";

interface UseVirtualScrollProps {
  itemCount: number;
  itemHeight: number;
  containerHeight: number;
  overscan?: number;
}

interface VisibleRange {
  start: number;
  end: number;
}

export function useVirtualScroll({
  itemCount,
  itemHeight,
  containerHeight,
  overscan = 5,
}: UseVirtualScrollProps) {
  const [scrollTop, setScrollTop] = useState(0);

  const visibleRange: VisibleRange = useMemo(() => {
    if (itemCount === 0) {
      return { start: 0, end: 0 };
    }

    // scrollTop can outlive a shrinking list (e.g. collapse all) until the next scroll event.
    const startIndex = Math.min(Math.floor(scrollTop / itemHeight), itemCount - 1);
    const visibleCount = Math.ceil(containerHeight / itemHeight);

    const start = Math.max(0, startIndex - overscan);
    const end = Math.min(itemCount, startIndex + visibleCount + overscan);

    return { start, end };
  }, [scrollTop, itemCount, itemHeight, containerHeight, overscan]);

  const totalHeight = itemCount * itemHeight;
  const offsetY = visibleRange.start * itemHeight;

  const onScroll = useCallback((e: React.UIEvent<HTMLDivElement>) => {
    setScrollTop(e.currentTarget.scrollTop);
  }, []);

  return { visibleRange, totalHeight, offsetY, onScroll };
}
