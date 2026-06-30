import { useState } from 'react';
import { Timeline, Tag, Button, Drawer, Space, Divider, Empty } from 'antd';
import { ClockCircleOutlined, CheckCircleOutlined, CloseCircleOutlined, LoadingOutlined } from '@ant-design/icons';
import type { PipelineRunNode } from '../api/pipelineApi';

interface ExecutionTimelineProps {
  nodes: PipelineRunNode[];
  loading?: boolean;
}

export default function ExecutionTimeline({ nodes, loading = false }: ExecutionTimelineProps) {
  const [selectedNode, setSelectedNode] = useState<PipelineRunNode | null>(null);
  const [drawerOpen, setDrawerOpen] = useState(false);

  if (!nodes || nodes.length === 0) {
    return <Empty description="无执行节点" />;
  }

  // Sort nodes by start time
  const sortedNodes = [...nodes].sort((a, b) => {
    const timeA = new Date(a.startedAt || a.createdAt || 0).getTime();
    const timeB = new Date(b.startedAt || b.createdAt || 0).getTime();
    return timeA - timeB;
  });

  const items = sortedNodes.map((node) => ({
    dot: getStatusIcon(node.phase),
    color: getStatusColor(node.phase),
    children: (
      <div
        onClick={() => {
          setSelectedNode(node);
          setDrawerOpen(true);
        }}
        style={{ cursor: 'pointer', padding: '8px', borderRadius: '4px' }}
        onMouseEnter={(e) => (e.currentTarget.style.backgroundColor = '#f5f5f5')}
        onMouseLeave={(e) => (e.currentTarget.style.backgroundColor = 'transparent')}
      >
        <div style={{ fontWeight: 500, marginBottom: '4px' }}>
          {node.displayName || node.templateName || node.argoNodeName || 'Node'}
        </div>
        <div style={{ fontSize: '12px', color: '#666' }}>
          <Space split={<Divider type="vertical" style={{ margin: '0 4px' }} />}>
            <span>{getPhaseLabel(node.phase)}</span>
            <span>{formatDuration(node.startedAt, node.finishedAt)}</span>
            {node.message && <span>💬 {node.message.slice(0, 30)}</span>}
          </Space>
        </div>
        <div style={{ marginTop: '8px' }}>
          {node.phase && <Tag color={getStatusColor(node.phase)}>{node.phase}</Tag>}
        </div>
      </div>
    )
  }));

  return (
    <>
      <Timeline items={items} />

      {selectedNode && (
        <Drawer
          title={`节点详情：${selectedNode.displayName || selectedNode.templateName}`}
          placement="right"
          onClose={() => {
            setDrawerOpen(false);
            setTimeout(() => setSelectedNode(null), 300);
          }}
          open={drawerOpen}
          width={600}
        >
          <div style={{ display: 'grid', gap: '16px' }}>
            {/* Basic Info */}
            <div>
              <h4>基本信息</h4>
              <table style={{ width: '100%', fontSize: '12px' }}>
                <tbody>
                  <tr>
                    <td style={{ paddingRight: '12px', color: '#666' }}>ID:</td>
                    <td>{selectedNode.id}</td>
                  </tr>
                  <tr>
                    <td style={{ paddingRight: '12px', color: '#666' }}>状态:</td>
                    <td>
                      <Tag color={getStatusColor(selectedNode.phase)}>
                        {selectedNode.phase || 'Unknown'}
                      </Tag>
                    </td>
                  </tr>
                  <tr>
                    <td style={{ paddingRight: '12px', color: '#666' }}>类型:</td>
                    <td>{selectedNode.type || 'Unknown'}</td>
                  </tr>
                  <tr>
                    <td style={{ paddingRight: '12px', color: '#666' }}>模板:</td>
                    <td>{selectedNode.templateName || '-'}</td>
                  </tr>
                </tbody>
              </table>
            </div>

            <Divider />

            {/* Timing */}
            <div>
              <h4>执行时间</h4>
              <table style={{ width: '100%', fontSize: '12px' }}>
                <tbody>
                  <tr>
                    <td style={{ paddingRight: '12px', color: '#666' }}>开始:</td>
                    <td>{formatTime(selectedNode.startedAt)}</td>
                  </tr>
                  <tr>
                    <td style={{ paddingRight: '12px', color: '#666' }}>结束:</td>
                    <td>{formatTime(selectedNode.finishedAt)}</td>
                  </tr>
                  <tr>
                    <td style={{ paddingRight: '12px', color: '#666' }}>耗时:</td>
                    <td>{formatDuration(selectedNode.startedAt, selectedNode.finishedAt)}</td>
                  </tr>
                </tbody>
              </table>
            </div>

            <Divider />

            {/* Resources */}
            {selectedNode.resourcesDuration && (
              <div>
                <h4>资源使用</h4>
                <table style={{ width: '100%', fontSize: '12px' }}>
                  <tbody>
                    {Object.entries(selectedNode.resourcesDuration).map(([key, value]) => (
                      <tr key={key}>
                        <td style={{ paddingRight: '12px', color: '#666' }}>{key}:</td>
                        <td>{value}ms</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}

            {/* Cost */}
            {selectedNode.estimatedCostUsd && (
              <div>
                <h4>成本估算</h4>
                <div style={{ fontSize: '18px', fontWeight: 'bold', color: '#52c41a' }}>
                  ${selectedNode.estimatedCostUsd.toFixed(2)}
                </div>
              </div>
            )}

            {/* Message */}
            {selectedNode.message && (
              <div>
                <h4>消息</h4>
                <div style={{ padding: '8px', backgroundColor: '#f5f5f5', borderRadius: '4px' }}>
                  {selectedNode.message}
                </div>
              </div>
            )}
          </div>
        </Drawer>
      )}
    </>
  );
}

function getStatusIcon(phase?: string) {
  switch (phase) {
    case 'Succeeded':
      return <CheckCircleOutlined style={{ fontSize: '16px' }} />;
    case 'Failed':
      return <CloseCircleOutlined style={{ fontSize: '16px' }} />;
    case 'Running':
      return <LoadingOutlined style={{ fontSize: '16px' }} />;
    default:
      return <ClockCircleOutlined style={{ fontSize: '16px' }} />;
  }
}

function getStatusColor(phase?: string) {
  switch (phase) {
    case 'Succeeded':
      return 'green';
    case 'Failed':
      return 'red';
    case 'Running':
      return 'blue';
    default:
      return 'gray';
  }
}

function getPhaseLabel(phase?: string) {
  const labels: Record<string, string> = {
    Succeeded: '成功',
    Failed: '失败',
    Running: '运行中',
    Pending: '待执行',
    Skipped: '已跳过'
  };
  return labels[phase || ''] || phase || '未知';
}

function formatTime(dateStr?: string) {
  if (!dateStr) return '-';
  return new Date(dateStr).toLocaleString('zh-CN');
}

function formatDuration(startStr?: string, endStr?: string) {
  if (!startStr || !endStr) return '-';
  const start = new Date(startStr).getTime();
  const end = new Date(endStr).getTime();
  const seconds = Math.round((end - start) / 1000);
  return `${seconds}s`;
}
