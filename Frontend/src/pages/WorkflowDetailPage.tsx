import { useCallback, useEffect, useState } from "react";
import { useParams, useNavigate } from "react-router-dom";
import { Button, Tag, Spin, Descriptions } from "antd";
import { ArrowLeftOutlined } from "@ant-design/icons";
import {
  ReactFlow,
  ReactFlowProvider,
  Background,
  BackgroundVariant,
  Controls,
  type Node as RFNode,
  type Edge as RFEdge,
  useNodesState,
  useEdgesState,
} from "@xyflow/react";
import "@xyflow/react/dist/style.css";
import {
  getWorkflow,
  type WorkflowDetail,
  type WorkflowNodeStatus,
} from "../api/workflowApi";

const PHASE_COLORS: Record<string, string> = {
  Succeeded: "#16a34a",
  Running: "#2563eb",
  Pending: "#d97706",
  Failed: "#dc2626",
  Error: "#dc2626",
  Skipped: "#6b7280",
};

const STATUS_COLORS: Record<string, string> = {
  Succeeded: "success",
  Running: "processing",
  Pending: "warning",
  Failed: "error",
  Error: "error",
};

function buildFlowNodes(
  nodes: WorkflowNodeStatus[],
  onNodeClick: (node: WorkflowNodeStatus) => void,
): RFNode[] {
  return nodes.map((n, i) => ({
    id: n.id,
    type: "default",
    position: { x: (i % 4) * 220, y: Math.floor(i / 4) * 120 },
    data: {
      label: (
        <div style={{ fontSize: 12, textAlign: "center" }}>
          <div style={{ fontWeight: 600 }}>{n.displayName || n.name}</div>
          <Tag
            color={STATUS_COLORS[n.phase] || "default"}
            style={{ fontSize: 10, marginTop: 4 }}
          >
            {n.phase}
          </Tag>
        </div>
      ),
      nodeStatus: n,
      onClick: () => onNodeClick(n),
    },
    style: {
      background: PHASE_COLORS[n.phase] || "#f3f4f6",
      color: "#fff",
      border: "none",
      borderRadius: 8,
      padding: 10,
      minWidth: 140,
    },
  }));
}

function buildFlowEdges(nodes: WorkflowNodeStatus[]): RFEdge[] {
  const edges: RFEdge[] = [];
  // Try to infer edges from node IDs — Argo uses parent/child naming
  for (const n of nodes) {
    const parts = n.id.split(".");
    if (parts.length > 1) {
      const parentId = parts.slice(0, -1).join(".");
      if (nodes.find((x) => x.id === parentId)) {
        edges.push({
          id: `e-${parentId}-${n.id}`,
          source: parentId,
          target: n.id,
          style: { stroke: "#94a3b8" },
        });
      }
    }
  }
  return edges;
}

function Flow({
  nodes: rawNodes,
  onNodeSelect,
}: {
  nodes: WorkflowNodeStatus[];
  onNodeSelect: (node: WorkflowNodeStatus | null) => void;
}) {
  const [nodes, setNodes, onNodesChange] = useNodesState(
    buildFlowNodes(rawNodes, onNodeSelect),
  );
  const [edges, setEdges, onEdgesChange] = useEdgesState(
    buildFlowEdges(rawNodes),
  );

  useEffect(() => {
    setNodes(buildFlowNodes(rawNodes, onNodeSelect));
    setEdges(buildFlowEdges(rawNodes));
  }, [rawNodes, onNodeSelect, setNodes, setEdges]);

  const onNodeClick = useCallback(
    (_: React.MouseEvent, node: RFNode) => {
      const ns = node.data?.nodeStatus as WorkflowNodeStatus | undefined;
      if (ns) {
        onNodeSelect(ns);
      }
    },
    [onNodeSelect],
  );

  const onPaneClick = useCallback(() => {
    onNodeSelect(null);
  }, [onNodeSelect]);

  return (
    <ReactFlow
      nodes={nodes}
      edges={edges}
      onNodesChange={onNodesChange}
      onEdgesChange={onEdgesChange}
      onNodeClick={onNodeClick}
      onPaneClick={onPaneClick}
      fitView
    >
      <Background variant={BackgroundVariant.Dots} gap={24} color="#cbd5e1" />
      <Controls />
    </ReactFlow>
  );
}

export default function WorkflowDetailPage() {
  const { name } = useParams<{ name: string }>();
  const navigate = useNavigate();
  const [wf, setWf] = useState<WorkflowDetail | null>(null);
  const [loading, setLoading] = useState(true);
  const [selectedNode, setSelectedNode] = useState<WorkflowNodeStatus | null>(
    null,
  );

  useEffect(() => {
    if (!name) return;
    setLoading(true);
    getWorkflow(name)
      .then(setWf)
      .catch(console.error)
      .finally(() => setLoading(false));
  }, [name]);

  if (loading) {
    return (
      <div
        style={{
          display: "flex",
          justifyContent: "center",
          padding: 80,
        }}
      >
        <Spin size="large" />
      </div>
    );
  }

  if (!wf) {
    return <div style={{ padding: 24 }}>未找到工作流</div>;
  }

  return (
    <div
      style={{
        height: "calc(100vh - 64px)",
        display: "flex",
        flexDirection: "column",
      }}
    >
      <div
        style={{
          padding: "12px 24px",
          borderBottom: "1px solid #e5e7eb",
          display: "flex",
          alignItems: "center",
          gap: 12,
        }}
      >
        <Button
          icon={<ArrowLeftOutlined />}
          onClick={() => navigate("/workflows")}
        >
          返回
        </Button>
        <h3 style={{ margin: 0 }}>{wf.name}</h3>
        <Tag color={STATUS_COLORS[wf.status] || "default"}>
          {wf.status}
        </Tag>
        <span style={{ color: "#6b7280", fontSize: 12 }}>
          创建: {new Date(wf.createdAt).toLocaleString()}
          {wf.finishedAt &&
            ` | 完成: ${new Date(wf.finishedAt).toLocaleString()}`}
        </span>
      </div>
      <div style={{ flex: 1, display: "flex" }}>
        <div style={{ flex: 1 }}>
          <ReactFlowProvider>
            <Flow
              nodes={wf.nodes}
              onNodeSelect={setSelectedNode}
            />
          </ReactFlowProvider>
        </div>
        {selectedNode && (
          <div
            style={{
              width: 300,
              borderLeft: "1px solid #e5e7eb",
              padding: 16,
              overflowY: "auto",
            }}
          >
            {selectedNode.displayName && (
              <h4>{selectedNode.displayName}</h4>
            )}
            <Descriptions column={1} size="small">
              <Descriptions.Item label="名称">
                {selectedNode.name}
              </Descriptions.Item>
              <Descriptions.Item label="状态">
                <Tag
                  color={STATUS_COLORS[selectedNode.phase] || "default"}
                >
                  {selectedNode.phase}
                </Tag>
              </Descriptions.Item>
              {selectedNode.message && (
                <Descriptions.Item label="消息">
                  {selectedNode.message}
                </Descriptions.Item>
              )}
              {selectedNode.startedAt && (
                <Descriptions.Item label="开始时间">
                  {new Date(selectedNode.startedAt).toLocaleString()}
                </Descriptions.Item>
              )}
              {selectedNode.finishedAt && (
                <Descriptions.Item label="完成时间">
                  {new Date(selectedNode.finishedAt).toLocaleString()}
                </Descriptions.Item>
              )}
            </Descriptions>
          </div>
        )}
      </div>
    </div>
  );
}
