// @vitest-environment jsdom

import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { MemoryRouter } from "react-router-dom";
import PipelinePage from "./PipelinePage";
import type { Deployment } from "../api/pipelineApi";

// ── Mock pipelineApi ──────────────────────────────────────────────
const mockSavePipeline = vi.fn();
const mockDeployTemplate = vi.fn();

vi.mock("../api/pipelineApi", () => ({
  savePipeline: (...args: unknown[]) => mockSavePipeline(...args),
  deployTemplate: (...args: unknown[]) => mockDeployTemplate(...args),
}));

// ── Mock pipelineComponentApi ─────────────────────────────────────
const mockListComponents = vi.fn().mockResolvedValue({ items: [] });

vi.mock("../api/pipelineComponentApi", () => ({
  listComponents: (...args: unknown[]) => mockListComponents(...args),
  createComponent: vi.fn(),
  updateComponent: vi.fn(),
  deleteComponent: vi.fn(),
}));

// ── Mock AssetPicker ──────────────────────────────────────────────
vi.mock("../components/pipeline/AssetPicker", () => ({
  default: ({ selectedIds, onSelectionChange }: any) => (
    <div data-testid="mock-asset-picker">
      <span>Selected: {selectedIds.join(",") || "(none)"}</span>
      <button data-testid="select-assets-btn" onClick={() => onSelectionChange(["ast-001", "ast-002"])}>
        MockSelect
      </button>
    </div>
  ),
}));

// ── Mock antd (message as spies inside factory) ───────────────────
vi.mock("antd", async (importOriginal) => {
  const actual = await importOriginal<Record<string, unknown>>();
  return {
    ...actual,
    message: { success: vi.fn(), error: vi.fn() },
  };
});

// ── Helpers ───────────────────────────────────────────────────────
function renderPage() {
  return render(
    <MemoryRouter>
      <PipelinePage />
    </MemoryRouter>,
  );
}

function mockDeployResult(overrides: Partial<Deployment> = {}): Deployment {
  return {
    id: "dep-001",
    pipelineName: "test-pipeline",
    workflowName: "wf-test-001",
    status: "Succeeded",
    nodeCount: 3,
    createdAt: "2026-05-28T12:00:00Z",
    ...overrides,
  };
}

/** Import a pipeline with one node so canvas is non-empty for deploy tests. */
function importOneNodePipeline(customName = "test-pipeline") {
  const pipeline = {
    name: customName,
    version: "1",
    nodes: [
      {
        id: "step-1",
        component: { name: "test", image: "busybox:latest", command: [], args: [] },
        inputs: [{ name: "input", type: "string" }],
        outputs: [{ name: "output", type: "string" }],
      },
    ],
    edges: [],
  };
  vi.spyOn(window, "prompt").mockReturnValue(JSON.stringify(pipeline));
  fireEvent.click(screen.getByText("导入"));
}

async function getMockMessage() {
  const antd: any = await import("antd");
  return antd.message;
}

/** Find the modal's primary deploy button (not the canvas toolbar one). */
function getModalDeployBtn(): HTMLButtonElement {
  // Ant Design 5 adds -sm for small buttons (canvas toolbar) and spacing in text ("部 署")
  const btn = document.querySelector<HTMLButtonElement>('.ant-btn-primary:not(.ant-btn-sm)');
  expect(btn).not.toBeNull();
  expect(btn!.disabled).toBe(false);
  return btn!;
}

// ── Suite ─────────────────────────────────────────────────────────
describe("PipelinePage", () => {
  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  // ── Render & structure ──────────────────────────────────────────
  it("renders the page with three tab buttons", () => {
    renderPage();
    expect(screen.getAllByText("组件").length).toBeGreaterThanOrEqual(1);
    expect(screen.getByText("画布")).toBeInTheDocument();
    expect(screen.getAllByText("部署").length).toBeGreaterThanOrEqual(1);
  });

  it("shows canvas toolbar buttons by default", () => {
    renderPage();
    expect(screen.getAllByText("部署").length).toBeGreaterThanOrEqual(1);
    expect(screen.getByText("保存")).toBeInTheDocument();
    expect(screen.getByText("导出")).toBeInTheDocument();
    expect(screen.getByText("导入")).toBeInTheDocument();
  });

  it("shows the pipeline name input with default value", () => {
    renderPage();
    expect(screen.getByDisplayValue("my-pipeline")).toBeInTheDocument();
  });

  // ── Tab switching ───────────────────────────────────────────────
  it("switches to components tab", async () => {
    renderPage();
    fireEvent.click(screen.getAllByText("组件")[0]);
    await waitFor(() => {
      expect(screen.getByText("组件注册表")).toBeInTheDocument();
    });
  });

  it("switches to deploy tab", async () => {
    renderPage();
    fireEvent.click(screen.getAllByText("部署")[0]);
    await waitFor(() => {
      expect(screen.getByText("部署记录")).toBeInTheDocument();
    });
  });

  it("switches back to canvas tab from deploy", () => {
    renderPage();
    fireEvent.click(screen.getAllByText("部署")[0]);
    fireEvent.click(screen.getByText("画布"));
    expect(screen.getByDisplayValue("my-pipeline")).toBeInTheDocument();
  });

  // ── Export ──────────────────────────────────────────────────────
  it("shows JSON output on export", () => {
    renderPage();
    fireEvent.click(screen.getByText("导出"));
    const pre = document.querySelector(".json-output");
    expect(pre).toBeInTheDocument();
    expect(pre!.textContent).toContain('"version"');
  });

  // ── Import ──────────────────────────────────────────────────────
  it("loads pipeline from JSON import", () => {
    renderPage();
    importOneNodePipeline("imported-pipeline");
    expect(screen.getByDisplayValue("imported-pipeline")).toBeInTheDocument();
    expect(document.querySelector(".json-output")).not.toBeInTheDocument();
  });

  it("shows error on invalid JSON import", async () => {
    vi.spyOn(window, "prompt").mockReturnValue("invalid json{{}");
    renderPage();
    fireEvent.click(screen.getByText("导入"));
    const msg = await getMockMessage();
    expect(msg.error).toHaveBeenCalledWith("无效的 JSON");
  });

  it("does nothing on cancelled import", () => {
    vi.spyOn(window, "prompt").mockReturnValue(null);
    renderPage();
    fireEvent.click(screen.getByText("导入"));
    expect(screen.getByDisplayValue("my-pipeline")).toBeInTheDocument();
  });

  // ── Save ────────────────────────────────────────────────────────
  it("saves pipeline on save button click", async () => {
    mockSavePipeline.mockResolvedValueOnce({ id: "tmpl-001", name: "my-pipeline" });
    renderPage();
    fireEvent.click(screen.getByText("保存"));
    await waitFor(() => expect(mockSavePipeline).toHaveBeenCalledTimes(1));
    expect(mockSavePipeline).toHaveBeenCalledWith("my-pipeline", expect.objectContaining({ name: "my-pipeline" }));
    const msg = await getMockMessage();
    expect(msg.success).toHaveBeenCalledWith(expect.stringContaining("已保存"));
  });

  it("shows error on save failure", async () => {
    mockSavePipeline.mockRejectedValueOnce(new Error("Network error"));
    renderPage();
    fireEvent.click(screen.getByText("保存"));
    const msg = await getMockMessage();
    await waitFor(() => expect(msg.error).toHaveBeenCalledWith(expect.stringContaining("保存失败")));
  });

  // ── Deploy dialog ──────────────────────────────────────────────
  it("opens deploy modal with title", () => {
    renderPage();
    fireEvent.click(screen.getByRole("button", { name: /play-circle/i }));
    expect(screen.getByText("部署流水线")).toBeInTheDocument();
  });

  it("shows node count in deploy modal (empty canvas)", () => {
    renderPage();
    fireEvent.click(screen.getByRole("button", { name: /play-circle/i }));
    expect(screen.getByText("0 个节点")).toBeInTheDocument();
  });

  it("shows node count in deploy modal (with nodes)", () => {
    renderPage();
    importOneNodePipeline("with-nodes");
    fireEvent.click(screen.getByRole("button", { name: /play-circle/i }));
    expect(screen.getByText("1 个节点")).toBeInTheDocument();
  });

  it("deploys pipeline with nodes", async () => {
    mockSavePipeline.mockResolvedValueOnce({ id: "tmpl-001", name: "with-nodes" });
    mockDeployTemplate.mockResolvedValueOnce(mockDeployResult());

    renderPage();
    importOneNodePipeline("with-nodes");

    // Open deploy modal
    fireEvent.click(screen.getByRole("button", { name: /play-circle/i }));

    // Change workflow name
    const nameInput = screen.getByPlaceholderText("with-nodes");
    fireEvent.change(nameInput, { target: { value: "my-workflow" } });

    // Click the modal's primary deploy button
    fireEvent.click(getModalDeployBtn());

    await waitFor(() => {
      expect(mockSavePipeline).toHaveBeenCalled();
      expect(mockDeployTemplate).toHaveBeenCalledWith("tmpl-001", undefined);
    });

    await waitFor(() => {
      // Ant Design 5 adds spacing in Chinese chars ("关 闭"), use regex
      expect(screen.getByText(/查看 Workflow/)).toBeInTheDocument();
      expect(screen.getByText(/关.*闭/)).toBeInTheDocument();
    });
  });

  it("deploys with selected assets", async () => {
    mockSavePipeline.mockResolvedValueOnce({ id: "tmpl-001", name: "with-nodes" });
    mockDeployTemplate.mockResolvedValueOnce(mockDeployResult());

    renderPage();
    importOneNodePipeline("with-nodes");
    fireEvent.click(screen.getByRole("button", { name: /play-circle/i }));

    // Expand asset picker
    fireEvent.click(screen.getByText("高级：绑定资产（可选）"));
    await waitFor(() => expect(screen.getByTestId("mock-asset-picker")).toBeInTheDocument());
    fireEvent.click(screen.getByTestId("select-assets-btn"));

    fireEvent.click(getModalDeployBtn());

    await waitFor(() => {
      expect(mockDeployTemplate).toHaveBeenCalledWith("tmpl-001", ["ast-001", "ast-002"]);
    });
  });

  it("shows error on deploy failure", async () => {
    mockSavePipeline.mockResolvedValueOnce({ id: "tmpl-001", name: "with-nodes" });
    mockDeployTemplate.mockRejectedValueOnce(new Error("Cluster unavailable"));

    renderPage();
    importOneNodePipeline("with-nodes");
    fireEvent.click(screen.getByRole("button", { name: /play-circle/i }));
    fireEvent.click(getModalDeployBtn());

    await waitFor(() => {
      expect(screen.getByText(/部署失败/)).toBeInTheDocument();
      expect(screen.getByText(/Cluster unavailable/)).toBeInTheDocument();
    });
  });

  it("navigates to workflow on '查看 Workflow'", async () => {
    mockSavePipeline.mockResolvedValueOnce({ id: "tmpl-001", name: "with-nodes" });
    mockDeployTemplate.mockResolvedValueOnce(mockDeployResult());

    renderPage();
    importOneNodePipeline("with-nodes");
    fireEvent.click(screen.getByRole("button", { name: /play-circle/i }));
    fireEvent.click(getModalDeployBtn());

    await waitFor(() => expect(screen.getByText("查看 Workflow")).toBeInTheDocument());
    fireEvent.click(screen.getByText("查看 Workflow"));
  });

  it("switches to deploy tab on '查看部署'", async () => {
    mockSavePipeline.mockResolvedValueOnce({ id: "tmpl-001", name: "with-nodes" });
    mockDeployTemplate.mockResolvedValueOnce(mockDeployResult());

    renderPage();
    importOneNodePipeline("with-nodes");
    fireEvent.click(screen.getByRole("button", { name: /play-circle/i }));
    fireEvent.click(getModalDeployBtn());

    await waitFor(() => expect(screen.getByText(/查看部署/)).toBeInTheDocument());
    fireEvent.click(screen.getByText(/查看部署/));

    await waitFor(() => {
      expect(screen.getByText("部署记录")).toBeInTheDocument();
    });
  });

  // ── Component registry ──────────────────────────────────────────
  it("loads components from API on mount", async () => {
    mockListComponents.mockResolvedValueOnce({
      items: [{
        id: "comp-1", name: "processor", image: "alpine", tag: "latest",
        source: "manual", inputPorts: [{ name: "input", type: "string" }],
        outputPorts: [{ name: "output", type: "string" }],
        resources: { command: ["sh", "-c"], args: [] }, envVars: [],
      }],
    });
    renderPage();
    await waitFor(() => expect(mockListComponents).toHaveBeenCalledTimes(1));
  });

  it("falls back to localStorage when API fails", async () => {
    mockListComponents.mockRejectedValueOnce(new Error("API down"));
    renderPage();
    await waitFor(() => expect(mockListComponents).toHaveBeenCalledTimes(1));
  });

  // ── sessionStorage ──────────────────────────────────────────────
  it("loads pipeline from sessionStorage on mount", async () => {
    sessionStorage.setItem("pipeline-edit", JSON.stringify({ name: "from-storage", version: "1", nodes: [], edges: [] }));
    renderPage();
    await waitFor(() => expect(screen.getByDisplayValue("from-storage")).toBeInTheDocument(), { timeout: 3000 });
  });

  it("handles empty sessionStorage gracefully", () => {
    renderPage();
    expect(screen.getByDisplayValue("my-pipeline")).toBeInTheDocument();
  });

  it("handles invalid JSON in sessionStorage gracefully", () => {
    sessionStorage.setItem("pipeline-edit", "{bad json");
    renderPage();
    expect(screen.getByDisplayValue("my-pipeline")).toBeInTheDocument();
  });
});
