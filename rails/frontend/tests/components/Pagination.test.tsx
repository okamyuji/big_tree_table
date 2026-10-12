import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { Pagination } from "../../src/components/Pagination";

describe("Pagination", () => {
  const defaultProps = {
    page: 3,
    totalPages: 10,
    total: 250,
    perPage: 25,
    onPageChange: vi.fn(),
    onPerPageChange: vi.fn(),
  };

  it("shows current page", () => {
    render(<Pagination {...defaultProps} />);

    const currentButton = screen.getByRole("button", { current: "page" });
    expect(currentButton).toHaveTextContent("3");
  });

  it("calls onPageChange with page-1 when previous is clicked", () => {
    const onPageChange = vi.fn();
    render(<Pagination {...defaultProps} onPageChange={onPageChange} />);

    const prevButton = screen.getByRole("button", { name: "前のページ" });
    fireEvent.click(prevButton);

    expect(onPageChange).toHaveBeenCalledWith(2);
  });

  it("disables previous button on page 1", () => {
    render(<Pagination {...defaultProps} page={1} />);

    const prevButton = screen.getByRole("button", { name: "前のページ" });
    expect(prevButton).toBeDisabled();
  });

  it("disables next button on last page", () => {
    render(<Pagination {...defaultProps} page={10} />);

    const nextButton = screen.getByRole("button", { name: "次のページ" });
    expect(nextButton).toBeDisabled();
  });
});
