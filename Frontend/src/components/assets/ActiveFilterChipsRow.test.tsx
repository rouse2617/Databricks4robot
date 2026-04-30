// @vitest-environment jsdom
import { describe, it, expect, vi, afterEach } from "vitest";
import { render, screen, fireEvent, cleanup } from "@testing-library/react";
import ActiveFilterChipsRow, {
  formatOpLabel,
  formatChipValue,
  formatChipText,
} from "./ActiveFilterChipsRow";
import type { FilterChip } from "../../lib/assets/assetsDiscoveryTypes";

afterEach(cleanup);

// ─── Helper ───

function makeChip(overrides: Partial<FilterChip> = {}): FilterChip {
  return {
    id: "chip_1",
    field: "env",
    op: "eq",
    value: "warehouse",
    source: "facet",
    ...overrides,
  };
}

// ─── formatOpLabel ───

describe("formatOpLabel", () => {
  it("maps known operators to symbols", () => {
    expect(formatOpLabel("eq")).toBe("=");
    expect(formatOpLabel("ne")).toBe("≠");
    expect(formatOpLabel("gt")).toBe(">");
    expect(formatOpLabel("lt")).toBe("<");
    expect(formatOpLabel("gte")).toBe("≥");
    expect(formatOpLabel("lte")).toBe("≤");
    expect(formatOpLabel("ilike")).toBe("~");
    expect(formatOpLabel("between")).toBe("between");
    expect(formatOpLabel("in")).toBe("in");
  });

  it("returns raw op for unknown operators", () => {
    expect(formatOpLabel("custom_op")).toBe("custom_op");
  });
});

// ─── formatChipValue ───

describe("formatChipValue", () => {
  it("returns string values as-is", () => {
    expect(formatChipValue("warehouse")).toBe("warehouse");
  });

  it("joins array values with comma and space", () => {
    expect(formatChipValue(["a", "b", "c"])).toBe("a, b, c");
  });
});

// ─── formatChipText ───

describe("formatChipText", () => {
  it("formats field op value", () => {
    expect(formatChipText(makeChip())).toBe("env = warehouse");
  });

  it("formats array values", () => {
    expect(
      formatChipText(makeChip({ op: "in", value: ["a", "b"] })),
    ).toBe("env in a, b");
  });
});

// ─── Component Rendering ───

describe("ActiveFilterChipsRow", () => {
  it("renders nothing when chips array is empty", () => {
    const { container } = render(
      <ActiveFilterChipsRow chips={[]} onRemoveChip={() => {}} onClearAll={() => {}} />,
    );
    expect(container.innerHTML).toBe("");
  });

  it("renders chips with correct text", () => {
    const chips = [
      makeChip({ id: "1", field: "env", op: "eq", value: "warehouse" }),
      makeChip({ id: "2", field: "status", op: "ne", value: "archived" }),
    ];
    render(
      <ActiveFilterChipsRow chips={chips} onRemoveChip={() => {}} onClearAll={() => {}} />,
    );
    expect(screen.getByText("env = warehouse")).toBeTruthy();
    expect(screen.getByText("status ≠ archived")).toBeTruthy();
  });

  it("renders 清除全部 button when chips exist", () => {
    render(
      <ActiveFilterChipsRow
        chips={[makeChip()]}
        onRemoveChip={() => {}}
        onClearAll={() => {}}
      />,
    );
    expect(screen.getByText("清除全部")).toBeTruthy();
  });

  it("calls onRemoveChip when chip close button is clicked", () => {
    const onRemoveChip = vi.fn();
    render(
      <ActiveFilterChipsRow
        chips={[makeChip({ id: "chip_42" })]}
        onRemoveChip={onRemoveChip}
        onClearAll={() => {}}
      />,
    );
    const tag = screen.getByLabelText("移除筛选: env = warehouse");
    const closeBtn = tag.querySelector(".ant-tag-close-icon") as HTMLElement;
    if (closeBtn) {
      fireEvent.click(closeBtn);
    }
    expect(onRemoveChip).toHaveBeenCalledWith("chip_42");
  });

  it("calls onClearAll when 清除全部 is clicked", () => {
    const onClearAll = vi.fn();
    render(
      <ActiveFilterChipsRow
        chips={[makeChip()]}
        onRemoveChip={() => {}}
        onClearAll={onClearAll}
      />,
    );
    fireEvent.click(screen.getByText("清除全部"));
    expect(onClearAll).toHaveBeenCalledOnce();
  });

  it("calls onClearField when a field-level clear button is clicked", () => {
    const onClearField = vi.fn();
    render(
      <ActiveFilterChipsRow
        chips={[
          makeChip({ id: "1", field: "owner", value: "alice" }),
          makeChip({ id: "2", field: "owner", value: "bob" }),
        ]}
        onRemoveChip={() => {}}
        onClearAll={() => {}}
        onClearField={onClearField}
      />,
    );
    fireEvent.click(screen.getByText("清空 owner"));
    expect(onClearField).toHaveBeenCalledWith("owner");
  });

  it("applies red color for algo_status:failed chip", () => {
    const chip = makeChip({ field: "algo_status", op: "eq", value: "failed" });
    render(
      <ActiveFilterChipsRow chips={[chip]} onRemoveChip={() => {}} onClearAll={() => {}} />,
    );
    const tag = screen.getByText("algo_status = failed").closest(".ant-tag");
    expect(tag?.className).toContain("red");
  });

  it("applies green color for status:approved chip", () => {
    const chip = makeChip({ field: "status", op: "eq", value: "approved" });
    render(
      <ActiveFilterChipsRow chips={[chip]} onRemoveChip={() => {}} onClearAll={() => {}} />,
    );
    const tag = screen.getByText("status = approved").closest(".ant-tag");
    expect(tag?.className).toContain("green");
  });

  it("applies green color for lifecycle_state:ready chip", () => {
    const chip = makeChip({ field: "lifecycle_state", op: "eq", value: "ready" });
    render(
      <ActiveFilterChipsRow chips={[chip]} onRemoveChip={() => {}} onClearAll={() => {}} />,
    );
    const tag = screen.getByText("lifecycle_state = ready").closest(".ant-tag");
    expect(tag?.className).toContain("green");
  });

  it("uses default color for other chips", () => {
    const chip = makeChip({ field: "env", op: "eq", value: "warehouse" });
    render(
      <ActiveFilterChipsRow chips={[chip]} onRemoveChip={() => {}} onClearAll={() => {}} />,
    );
    const tag = screen.getByText("env = warehouse").closest(".ant-tag");
    expect(tag?.className).not.toContain("red");
    expect(tag?.className).not.toContain("green");
  });
});
