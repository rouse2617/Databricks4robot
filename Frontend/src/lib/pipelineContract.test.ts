import { describe, expect, it } from "vitest";
import {
  formatEdgeEndpoint,
  toTranspilerPipeline,
  fromTranspilerPipeline,
} from "./pipelineContract";
import type { Pipeline, PipelineNodeData } from "../components/pipeline/types";
import type { Node, Edge } from "@xyflow/react";

// ── formatEdgeEndpoint ────────────────────────────────────────────────

describe("formatEdgeEndpoint", () => {
  it("formats with explicit handle", () => {
    expect(formatEdgeEndpoint("step-1", "output", "output")).toBe(
      "step-1.output",
    );
  });

  it("falls back to default port when handle is empty", () => {
    expect(formatEdgeEndpoint("step-1", "", "output")).toBe("step-1.output");
  });

  it("falls back to default port when handle is null", () => {
    expect(formatEdgeEndpoint("step-1", null, "output")).toBe("step-1.output");
  });

  it("falls back to default port when handle is undefined", () => {
    expect(formatEdgeEndpoint("step-1", undefined, "output")).toBe(
      "step-1.output",
    );
  });

  it("strips leading dot from handle", () => {
    expect(formatEdgeEndpoint("step-1", ".output", "output")).toBe(
      "step-1.output",
    );
  });

  it("returns nodeId as-is when it already contains a dot", () => {
    expect(formatEdgeEndpoint("parent.child", "output", "output")).toBe(
      "parent.child",
    );
  });

  it("returns nodeId as-is even with null handle when nodeId has dot", () => {
    expect(formatEdgeEndpoint("parent.child", null, "output")).toBe(
      "parent.child",
    );
  });

  it("uses custom default port when no handle provided", () => {
    expect(formatEdgeEndpoint("step-1", null, "custom")).toBe("step-1.custom");
  });
});

// ── toTranspilerPipeline ─────────────────────────────────────────────

describe("toTranspilerPipeline", () => {
  it("converts empty nodes/edges to pipeline with defaults", () => {
    const result = toTranspilerPipeline([], [], { name: "test" });
    expect(result.name).toBe("test");
    expect(result.version).toBe("1");
    expect(result.nodes).toEqual([]);
    expect(result.edges).toEqual([]);
  });

  it("converts a single node correctly", () => {
    const nodes: Node<PipelineNodeData>[] = [
      {
        id: "step-1",
        type: "pipelineStep",
        position: { x: 0, y: 0 },
        data: {
          label: "echo",
          image: "busybox:latest",
          command: ["sh", "-c"],
          args: [{ name: "args", value: "hello" }],
          cpu: "",
          memory: "",
          disk: "",
        },
      },
    ];
    const result = toTranspilerPipeline(nodes, [], { name: "test" });
    expect(result.nodes).toHaveLength(1);
    expect(result.nodes[0].id).toBe("step-1");
    expect(result.nodes[0].component.name).toBe("echo");
    expect(result.nodes[0].component.image).toBe("busybox:latest");
    expect(result.nodes[0].component.command).toEqual(["sh", "-c"]);
    expect(result.nodes[0].component.args).toEqual([
      { name: "args", value: "hello" },
    ]);
    expect(result.nodes[0].component.resources).toBeUndefined();
    expect(result.nodes[0].inputs).toHaveLength(1);
    expect(result.nodes[0].inputs![0].name).toBe("input");
    expect(result.nodes[0].outputs).toHaveLength(1);
    expect(result.nodes[0].outputs![0].name).toBe("output");
  });

  it("includes resources when CPU/memory/disk are set", () => {
    const nodes: Node<PipelineNodeData>[] = [
      {
        id: "step-1",
        type: "pipelineStep",
        position: { x: 0, y: 0 },
        data: {
          label: "compute",
          image: "python:3.11",
          command: ["python", "main.py"],
          args: [],
          cpu: "1000m",
          memory: "512Mi",
          disk: "10Gi",
        },
      },
    ];
    const result = toTranspilerPipeline(nodes, [], { name: "test" });
    expect(result.nodes[0].component.resources).toEqual({
      cpu: "1000m",
      memory: "512Mi",
      disk: "10Gi",
    });
  });

  it("omits resources when all are empty", () => {
    const nodes: Node<PipelineNodeData>[] = [
      {
        id: "step-1",
        type: "pipelineStep",
        position: { x: 0, y: 0 },
        data: {
          label: "basic",
          image: "busybox",
          command: [],
          args: [],
          cpu: "",
          memory: "",
          disk: "",
        },
      },
    ];
    const result = toTranspilerPipeline(nodes, [], { name: "test" });
    expect(result.nodes[0].component.resources).toBeUndefined();
  });

  it("converts edges with source/target handles", () => {
    const edges: Edge[] = [
      {
        id: "e-1",
        source: "step-1",
        sourceHandle: "output",
        target: "step-2",
        targetHandle: "input",
      },
    ];
    const result = toTranspilerPipeline([], edges, { name: "test" });
    expect(result.edges).toHaveLength(1);
    expect(result.edges[0].source).toBe("step-1.output");
    expect(result.edges[0].target).toBe("step-2.input");
  });

  it("converts edges without handles using defaults", () => {
    const edges: Edge[] = [
      {
        id: "e-1",
        source: "step-1",
        sourceHandle: null,
        target: "step-2",
        targetHandle: null,
      },
    ];
    const result = toTranspilerPipeline([], edges, { name: "test" });
    expect(result.edges[0].source).toBe("step-1.output");
    expect(result.edges[0].target).toBe("step-2.input");
  });

  it("uses custom version when provided", () => {
    const result = toTranspilerPipeline([], [], {
      name: "test",
      version: "2",
    });
    expect(result.version).toBe("2");
  });
});

// ── fromTranspilerPipeline ───────────────────────────────────────────

describe("fromTranspilerPipeline", () => {
  it("restores nodes and edges from pipeline JSON", () => {
    const pipeline: Pipeline = {
      name: "my-pipeline",
      version: "1",
      nodes: [
        {
          id: "step-1",
          component: {
            name: "echo",
            image: "busybox:latest",
            command: ["sh", "-c"],
            args: [{ name: "args", value: "hello" }],
          },
        },
      ],
      edges: [
        { source: "step-1.output", target: "step-2.input" },
      ],
    };
    const { nodes, edges } = fromTranspilerPipeline(pipeline);
    expect(nodes).toHaveLength(1);
    expect(nodes[0].id).toBe("step-1");
    expect(nodes[0].type).toBe("pipelineStep");
    expect(nodes[0].data.label).toBe("echo");
    expect(nodes[0].data.image).toBe("busybox:latest");
    expect(nodes[0].data.command).toEqual(["sh", "-c"]);
    expect(nodes[0].data.args).toEqual([{ name: "args", value: "hello" }]);
    expect(edges).toHaveLength(1);
    expect(edges[0].source).toBe("step-1");
    expect(edges[0].target).toBe("step-2");
    expect(edges[0].sourceHandle).toBe("output");
    expect(edges[0].targetHandle).toBe("input");
  });

  it("restores resources from pipeline JSON", () => {
    const pipeline: Pipeline = {
      name: "compute",
      version: "1",
      nodes: [
        {
          id: "step-1",
          component: {
            name: "ai-model",
            image: "python:3.11",
            command: ["python", "train.py"],
            args: [],
            resources: { cpu: "2000m", memory: "2Gi", disk: "20Gi" },
          },
        },
      ],
      edges: [],
    };
    const { nodes } = fromTranspilerPipeline(pipeline);
    expect(nodes[0].data.cpu).toBe("2000m");
    expect(nodes[0].data.memory).toBe("2Gi");
    expect(nodes[0].data.disk).toBe("20Gi");
  });

  it("defaults empty resources to empty strings", () => {
    const pipeline: Pipeline = {
      name: "basic",
      version: "1",
      nodes: [
        {
          id: "step-1",
          component: {
            name: "echo",
            image: "busybox",
            command: [],
            args: [],
          },
        },
      ],
      edges: [],
    };
    const { nodes } = fromTranspilerPipeline(pipeline);
    expect(nodes[0].data.cpu).toBe("");
    expect(nodes[0].data.memory).toBe("");
    expect(nodes[0].data.disk).toBe("");
  });

  it("handles edges without port handles", () => {
    const pipeline: Pipeline = {
      name: "simple",
      version: "1",
      nodes: [
        {
          id: "step-1",
          component: { name: "a", image: "img" },
        },
        {
          id: "step-2",
          component: { name: "b", image: "img" },
        },
      ],
      edges: [
        { source: "step-1", target: "step-2" },
      ],
    };
    const { edges } = fromTranspilerPipeline(pipeline);
    expect(edges[0].source).toBe("step-1");
    expect(edges[0].target).toBe("step-2");
    expect(edges[0].sourceHandle).toBeUndefined();
    expect(edges[0].targetHandle).toBeUndefined();
  });

  it("assigns sequential edge IDs", () => {
    const pipeline: Pipeline = {
      name: "multi-edge",
      version: "1",
      nodes: [],
      edges: [
        { source: "a.output", target: "b.input" },
        { source: "b.output", target: "c.input" },
      ],
    };
    const { edges } = fromTranspilerPipeline(pipeline);
    expect(edges[0].id).toBe("e-0");
    expect(edges[1].id).toBe("e-1");
  });
});

// ── Round-trip ───────────────────────────────────────────────────────

describe("round-trip: toTranspilerPipeline → fromTranspilerPipeline", () => {
  it("preserves node data through a full round-trip", () => {
    const inputNodes: Node<PipelineNodeData>[] = [
      {
        id: "step-1",
        type: "pipelineStep",
        position: { x: 100, y: 200 },
        data: {
          label: "echo",
          image: "busybox:latest",
          command: ["sh", "-c"],
          args: [{ name: "args", value: "hello" }],
          cpu: "500m",
          memory: "256Mi",
          disk: "1Gi",
        },
      },
      {
        id: "step-2",
        type: "pipelineStep",
        position: { x: 300, y: 400 },
        data: {
          label: "python",
          image: "python:3.11",
          command: ["python", "script.py"],
          args: [],
          cpu: "",
          memory: "",
          disk: "",
        },
      },
    ];
    const inputEdges: Edge[] = [
      {
        id: "e-react-1",
        source: "step-1",
        sourceHandle: "output",
        target: "step-2",
        targetHandle: "input",
      },
    ];

    // to → from
    const pipeline = toTranspilerPipeline(inputNodes, inputEdges, {
      name: "roundtrip",
    });
    const { nodes, edges } = fromTranspilerPipeline(pipeline);

    // Nodes
    expect(nodes).toHaveLength(2);
    expect(nodes[0].id).toBe("step-1");
    expect(nodes[0].data.label).toBe("echo");
    expect(nodes[0].data.image).toBe("busybox:latest");
    expect(nodes[0].data.command).toEqual(["sh", "-c"]);
    expect(nodes[0].data.args).toEqual([{ name: "args", value: "hello" }]);
    expect(nodes[0].data.cpu).toBe("500m");
    expect(nodes[0].data.memory).toBe("256Mi");
    expect(nodes[0].data.disk).toBe("1Gi");
    // Node without resources
    expect(nodes[1].data.cpu).toBe("");
    expect(nodes[1].data.memory).toBe("");
    expect(nodes[1].data.disk).toBe("");

    // Edges
    expect(edges).toHaveLength(1);
    expect(edges[0].source).toBe("step-1");
    expect(edges[0].target).toBe("step-2");
    expect(edges[0].sourceHandle).toBe("output");
    expect(edges[0].targetHandle).toBe("input");
  });

  it("preserves pipeline name through round-trip", () => {
    const pipeline = toTranspilerPipeline([], [], { name: "my-pipeline" });
    expect(pipeline.name).toBe("my-pipeline");
    // fromTranspilerPipeline doesn't carry name — that's stored as page state
    // Verify the conversion produces a valid output
    const { nodes, edges } = fromTranspilerPipeline(pipeline);
    expect(nodes).toHaveLength(0);
    expect(edges).toHaveLength(0);
  });

  it("handles empty pipeline round-trip", () => {
    const pipeline = toTranspilerPipeline([], [], { name: "empty" });
    const { nodes, edges } = fromTranspilerPipeline(pipeline);
    expect(nodes).toHaveLength(0);
    expect(edges).toHaveLength(0);
  });
});
