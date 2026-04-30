// @vitest-environment jsdom
import { describe, it, expect, vi, afterEach } from "vitest";
import { render, screen, fireEvent, cleanup } from "@testing-library/react";
import ExpiringAssetsChip from "./ExpiringAssetsChip";
import type { FilterChip } from "../../lib/assets/assetsDiscoveryTypes";

afterEach(cleanup);

function makeExpiringChip(): FilterChip {
  return {
    id: "expire_chip_1",
    field: "expire_at",
    op: "between",
    value: "2025-01-01T00:00:00.000Z,2025-01-31T00:00:00.000Z",
    source: "add_filter",
  };
}

describe("ExpiringAssetsChip", () => {
  it("renders the chip with correct label", () => {
    render(
      <ExpiringAssetsChip
        activeFilters={[]}
        onAddFilter={() => {}}
        onRemoveFilter={() => {}}
      />,
    );
    expect(screen.getByText("30 天内将过期")).toBeTruthy();
  });

  it("calls onAddFilter when clicked and no existing filter", () => {
    const onAddFilter = vi.fn();
    render(
      <ExpiringAssetsChip
        activeFilters={[]}
        onAddFilter={onAddFilter}
        onRemoveFilter={() => {}}
      />,
    );
    fireEvent.click(screen.getByText("30 天内将过期"));
    expect(onAddFilter).toHaveBeenCalledOnce();
    const chip = onAddFilter.mock.calls[0][0] as FilterChip;
    expect(chip.field).toBe("expire_at");
    expect(chip.op).toBe("between");
    expect(chip.source).toBe("add_filter");
  });

  it("calls onRemoveFilter when clicked and filter already active", () => {
    const existingChip = makeExpiringChip();
    const onRemoveFilter = vi.fn();
    render(
      <ExpiringAssetsChip
        activeFilters={[existingChip]}
        onAddFilter={() => {}}
        onRemoveFilter={onRemoveFilter}
      />,
    );
    fireEvent.click(screen.getByText("30 天内将过期"));
    expect(onRemoveFilter).toHaveBeenCalledWith("expire_chip_1");
  });

  it("has aria-pressed=true when active", () => {
    const existingChip = makeExpiringChip();
    render(
      <ExpiringAssetsChip
        activeFilters={[existingChip]}
        onAddFilter={() => {}}
        onRemoveFilter={() => {}}
      />,
    );
    const tag = screen.getByTestId("expiring-assets-chip");
    expect(tag.getAttribute("aria-pressed")).toBe("true");
  });

  it("has aria-pressed=false when inactive", () => {
    render(
      <ExpiringAssetsChip
        activeFilters={[]}
        onAddFilter={() => {}}
        onRemoveFilter={() => {}}
      />,
    );
    const tag = screen.getByTestId("expiring-assets-chip");
    expect(tag.getAttribute("aria-pressed")).toBe("false");
  });
});
