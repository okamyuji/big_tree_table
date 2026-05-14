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

    const startIndex = Math.floor(scrollTop / itemHeight);
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
