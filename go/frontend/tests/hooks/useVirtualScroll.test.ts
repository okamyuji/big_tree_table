import { describe, it, expect } from "vitest";
import { renderHook, act } from "@testing-library/react";
import { useVirtualScroll } from "../../src/hooks/useVirtualScroll";

describe("useVirtualScroll", () => {
  it("calculates visibleRange correctly with 100 items, 40px height, 400px container", () => {
    const { result } = renderHook(() =>
      useVirtualScroll({
        itemCount: 100,
        itemHeight: 40,
        containerHeight: 400,
        overscan: 5,
      }),
    );

    // At scrollTop=0: startIndex=0, visibleCount=10
    // start = max(0, 0-5) = 0
    // end = min(100, 0+10+5) = 15
    expect(result.current.visibleRange.start).toBe(0);
    expect(result.current.visibleRange.end).toBe(15);
    expect(result.current.totalHeight).toBe(4000);
  });

  it("updates visibleRange when scrolled", () => {
    const { result } = renderHook(() =>
      useVirtualScroll({
        itemCount: 100,
        itemHeight: 40,
        containerHeight: 400,
        overscan: 5,
      }),
    );

    act(() => {
      result.current.onScroll({
        currentTarget: { scrollTop: 800 },
      } as unknown as React.UIEvent<HTMLDivElement>);
    });

    // scrollTop=800: startIndex=20, visibleCount=10
    // start = max(0, 20-5) = 15
    // end = min(100, 20+10+5) = 35
    expect(result.current.visibleRange.start).toBe(15);
    expect(result.current.visibleRange.end).toBe(35);
  });

  it("handles itemCount=0 safely", () => {
    const { result } = renderHook(() =>
      useVirtualScroll({
        itemCount: 0,
        itemHeight: 40,
        containerHeight: 400,
      }),
    );

    expect(result.current.visibleRange.start).toBe(0);
    expect(result.current.visibleRange.end).toBe(0);
    expect(result.current.totalHeight).toBe(0);
  });
});
