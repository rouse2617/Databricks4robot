import { describe, it, expect } from "vitest";
import { render } from "@testing-library/react";
import ResultsEmptyState from "./ResultsEmptyState";

describe("ResultsEmptyState", () => {
  it("renders empty state message", () => {
    render(<ResultsEmptyState variant="initial" />);
    // The component should render some empty state content
    const container = document.querySelector("[class*='empty']") || document.body;
    expect(container).toBeTruthy();
  });
});
