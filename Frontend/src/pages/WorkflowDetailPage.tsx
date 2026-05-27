import { useCallback, useEffect, useState } from "react";
import { useParams, useNavigate } from "react-router-dom";
import { Button, Tag, Spin, Descriptions, Segmented, Tabs, message } from "antd";
import { ArrowLeftOutlined, ApartmentOutlined, BarsOutlined } from "@ant-design/icons";
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
  getWorkflowLogs,
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

function TimelineView({ nodes }: { nodes: WorkflowNodeStatus[] }) {
  const withTime = nodes.filter((n) => n.startedAt);
  if (withTime.length === 0) {
    return (
      <div style={{ padding: 40, textAlign: "center", color: "#6b7280" }}>
        暂无节点时间数据
      </div>
    );
  }

  const times = withTime.map((n) => ({
    start: new Date(n.startedAt!).getTime(),
    end: n.finishedAt ? new Date(n.finishedAt).getTime() : Date.now(),
  }));
  const globalStart = Math.min(...times.map((t) => t.start));
  const globalEnd = Math.max(...times.map((t) => t.end));
  const range = globalEnd - globalStart || 1;

  const rowH = 36;
  const headerH = 30;
  const labelW = 180;
  const padL = 16;

  return (
    <div style={{ overflow: "auto", height: "100%", padding: 16 }}>
      <div style={{ minWidth: 600 }}>
        {/* Time axis header */}
        <div style={{ position: "relative", height: headerH, marginLeft: labelW + padL }}>
          {[0, 0.25, 0.5, 0.75, 1].map((pct) => (
            <div
              key={pct}
              style={{
                position: "absolute",
                left: `${pct * 100}%`,
                top: 0,
                fontSize: 11,
                color: "#6b7280",
                transform: "translateX(-50%)",
              }}
            >
              {new Date(globalStart + range * pct).toLocaleTimeString()}
            </div>
          ))}
          <div
            style={{
              position: "absolute",
              top: 16,
              left: 0,
              right: 0,
              height: 1,
              background: "#e5e7eb",
            }}
          />
        </div>
        {/* Node rows */}
        {withTime.map((n) => {
          const t = times.find((_, i) => withTime[i].id === n.id)!;
          const left = ((t.start - globalStart) / range) * 100;
          const width = ((t.end - t.start) / range) * 100;
          return (
            <div key={n.id} style={{ display: "flex", alignItems: "center", height: rowH }}>
              <div
                style={{
                  width: labelW,
                  fontSize: 12,
                  textAlign: "right",
                  paddingRight: padL,
                  overflow: "hidden",
                  textOverflow: "ellipsis",
                  whiteSpace: "nowrap",
                  color: "#374151",
                }}
                title={n.displayName || n.name}
              >
                {n.displayName || n.name}
              </div>
              <div style={{ flex: 1, position: "relative" }}>
                <div
                  style={{
                    position: "absolute",
                    left: `${left}%`,
                    width: `${Math.max(width, 1)}%`,
                    top: 4,
                    height: rowH - 8,
                    borderRadius: 4,
                    background: PHASE_COLORS[n.phase] || "#9ca3af",
                    opacity: 0.85,
                    display: "flex",
                    alignItems: "center",
                    paddingLeft: 6,
                    fontSize: 11,
                    color: "#fff",
                    overflow: "hidden",
                  }}
                >
                  {Math.round((t.end - t.start) / 1000)}s
                </div>
              </div>
            </div>
          );
        })}
      </div>
    </div>
  );
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

function LogViewer({
  workflowName,
  nodeId,
  loading,
  logContent,
  onLoad,
}: {
  workflowName: string;
  nodeId: string;
  loading: boolean;
  logContent: string | null;
  onLoad: (loading: boolean, content: string | null) => void;
}) {
  useEffect(() => {
    onLoad(true, null);
    getWorkflowLogs(workflowName, nodeId)
      .then((res) => onLoad(false, res.logs || "(无日志)"))
      .catch((err) => {
        onLoad(false, null);
        message.error("获取日志失败: " + String(err));
      });
  }, [workflowName, nodeId, onLoad]);

  if (loading) {
    return <div style={{ textAlign: "center", padding: 40 }}><Spin /></div>;
  }
  if (logContent === null) {
    return <div style={{ color: "#999", textAlign: "center", padding: 24 }}>获取日志失败</div>;
  }
  return (
    <pre
      style={{
        fontSize: 11,
        fontFamily: '"SF Mono", "Fira Code", monospace',
        whiteSpace: "pre-wrap",
        wordBreak: "break-all",
        maxHeight: 400,
        overflow: "auto",
        background: "#f8f9fa",
        padding: 12,
        borderRadius: 6,
        margin: 0,
      }}
    >
      {logContent}
    </pre>
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
  const [viewMode, setViewMode] = useState<"dag" | "timeline">("dag");
  const [logLoading, setLogLoading] = useState(false);
  const [logContent, setLogContent] = useState<string | null>(null);

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
        <div style={{ marginLeft: "auto" }}>
          <Segmented
            value={viewMode}
            onChange={(val) => setViewMode(val as "dag" | "timeline")}
            options={[
              { label: <><ApartmentOutlined /> DAG</>, value: "dag" },
              { label: <><BarsOutlined /> 时间线</>, value: "timeline" },
            ]}
          />
        </div>
      </div>
      <div style={{ flex: 1, display: "flex" }}>
        <div style={{ flex: 1 }}>
          {viewMode === "dag" ? (
            <ReactFlowProvider>
              <Flow
                nodes={wf.nodes}
                onNodeSelect={setSelectedNode}
              />
            </ReactFlowProvider>
          ) : (
            <TimelineView nodes={wf.nodes} />
          )}
        </div>
        {selectedNode && (
          <div
            style={{
              width: 360,
              borderLeft: "1px solid #e5e7eb",
              padding: 16,
              overflowY: "auto",
            }}
          >
            <h4 style={{ marginBottom: 12 }}>{selectedNode.displayName || selectedNode.name}</h4>
            <Tabs
              size="small"
              items={[
                {
                  key: "detail",
                  label: "详情",
                  children: (
                    <Descriptions column={1} size="small" colon={false}>
                      <Descriptions.Item label="名称">
                        {selectedNode.name}
                      </Descriptions.Item>
                      <Descriptions.Item label="状态">
                        <Tag color={STATUS_COLORS[selectedNode.phase] || "default"}>
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
                  ),
                },
                {
                  key: "logs",
                  label: "日志",
                  children: (
                    <LogViewer
                      key={selectedNode.id}
                      workflowName={name || ""}
                      nodeId={selectedNode.id}
                      loading={logLoading}
                      logContent={logContent}
                      onLoad={(loading, content) => {
                        setLogLoading(loading);
                        setLogContent(content);
                      }}
                    />
                  ),
                },
              ]}
            />
          </div>
        )}
      </div>
    </div>
  );
}
