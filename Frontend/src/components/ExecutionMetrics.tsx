import { Card, Row, Col, Progress, Statistic, Table, Empty } from 'antd';
import { DollarOutlined, ClockCircleOutlined, DatabaseOutlined } from '@ant-design/icons';
import type { PipelineRunNode } from '../api/pipelineApi';

interface ExecutionMetricsProps {
  nodes: PipelineRunNode[];
  totalEstimatedCost?: number | null;
  loading?: boolean;
}

export default function ExecutionMetrics({
  nodes,
  totalEstimatedCost,
  loading = false
}: ExecutionMetricsProps) {
  if (!nodes || nodes.length === 0) {
    return <Empty description="无执行指标数据" />;
  }

  // Calculate metrics
  const metrics = calculateMetrics(nodes);

  // Prepare node summary table
  const nodeData = nodes
    .filter(n => n.estimatedCostUsd)
    .sort((a, b) => (b.estimatedCostUsd || 0) - (a.estimatedCostUsd || 0))
    .slice(0, 10)
    .map((node, idx) => ({
      key: node.id,
      rank: idx + 1,
      name: node.displayName || node.templateName || node.argoNodeName || 'Node',
      cost: node.estimatedCostUsd?.toFixed(2),
      duration: calculateNodeDuration(node.startedAt, node.finishedAt),
      status: node.phase
    }));

  const columns = [
    { title: '排名', dataIndex: 'rank', width: 60 },
    { title: '节点名称', dataIndex: 'name', ellipsis: true },
    { title: '预估成本', dataIndex: 'cost', render: (v: string) => `$${v}` },
    { title: '耗时', dataIndex: 'duration' },
    { title: '状态', dataIndex: 'status', render: (status: string) => getStatusBadge(status) }
  ];

  return (
    <div style={{ display: 'grid', gap: '24px' }}>
      {/* Summary Cards */}
      <Row gutter={16}>
        <Col xs={24} sm={12} lg={6}>
          <Card loading={loading}>
            <Statistic
              title="总成本"
              prefix={<DollarOutlined />}
              value={totalEstimatedCost || metrics.totalCost}
              precision={2}
              suffix="USD"
              valueStyle={{ color: '#f5222d' }}
            />
          </Card>
        </Col>

        <Col xs={24} sm={12} lg={6}>
          <Card loading={loading}>
            <Statistic
              title="总耗时"
              prefix={<ClockCircleOutlined />}
              value={metrics.totalDuration}
              suffix="秒"
            />
          </Card>
        </Col>

        <Col xs={24} sm={12} lg={6}>
          <Card loading={loading}>
            <Statistic
              title="平均成本/秒"
              prefix={<DollarOutlined />}
              value={metrics.costPerSecond}
              precision={4}
              suffix="USD/s"
            />
          </Card>
        </Col>

        <Col xs={24} sm={12} lg={6}>
          <Card loading={loading}>
            <Statistic
              title="节点数"
              prefix={<DatabaseOutlined />}
              value={metrics.nodeCount}
            />
          </Card>
        </Col>
      </Row>

      {/* Phase Distribution */}
      <Card title="执行状态分布" loading={loading}>
        <Row gutter={16}>
          {Object.entries(metrics.phaseDistribution).map(([phase, count]) => (
            <Col key={phase} xs={12} sm={8} lg={6}>
              <div style={{ textAlign: 'center' }}>
                <Progress
                  type="circle"
                  percent={Math.round((count / metrics.nodeCount) * 100)}
                  width={80}
                  format={(percent) => <div style={{ fontSize: '12px' }}>{count}</div>}
                  strokeColor={getPhaseColor(phase)}
                />
                <div style={{ marginTop: '8px', fontSize: '12px' }}>{phase}</div>
              </div>
            </Col>
          ))}
        </Row>
      </Card>

      {/* Cost by Node */}
      <Card title="按节点成本排序 (Top 10)" loading={loading}>
        <Table
          dataSource={nodeData}
          columns={columns}
          pagination={{ pageSize: 10 }}
          size="small"
          scroll={{ x: 500 }}
        />
      </Card>

      {/* Performance Timeline */}
      <Card title="性能趋势" loading={loading}>
        <div style={{ fontSize: '12px', color: '#666' }}>
          <table style={{ width: '100%' }}>
            <tbody>
              <tr>
                <td style={{ paddingRight: '24px' }}>最长耗时节点:</td>
                <td>
                  {metrics.slowestNode?.name} ({metrics.slowestNode?.duration || 0}s)
                </td>
              </tr>
              <tr>
                <td style={{ paddingRight: '24px' }}>最高成本节点:</td>
                <td>
                  {metrics.expensiveNode?.name} ($
                  {metrics.expensiveNode?.cost.toFixed(2)})
                </td>
              </tr>
              <tr>
                <td style={{ paddingRight: '24px' }}>成功率:</td>
                <td>
                  {(
                    ((metrics.phaseDistribution['Succeeded'] || 0) / metrics.nodeCount) *
                    100
                  ).toFixed(1)}
                  %
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </Card>
    </div>
  );
}

function calculateMetrics(nodes: PipelineRunNode[]) {
  let totalCost = 0;
  let totalDuration = 0;
  let slowestDuration = 0;
  let slowestNode: any = null;
  let expensiveNode: any = null;
  let maxCost = 0;

  const phaseDistribution: Record<string, number> = {};

  for (const node of nodes) {
    // Cost
    if (node.estimatedCostUsd) {
      totalCost += node.estimatedCostUsd;
      if (node.estimatedCostUsd > maxCost) {
        maxCost = node.estimatedCostUsd;
        expensiveNode = {
          name: node.displayName || node.templateName || 'Node',
          cost: node.estimatedCostUsd
        };
      }
    }

    // Duration
    const duration = calculateNodeDuration(node.startedAt, node.finishedAt);
    totalDuration += duration;
    if (duration > slowestDuration) {
      slowestDuration = duration;
      slowestNode = {
        name: node.displayName || node.templateName || 'Node',
        duration
      };
    }

    // Phase distribution
    const phase = node.phase || 'Unknown';
    phaseDistribution[phase] = (phaseDistribution[phase] || 0) + 1;
  }

  return {
    totalCost,
    totalDuration,
    costPerSecond: totalDuration > 0 ? totalCost / totalDuration : 0,
    nodeCount: nodes.length,
    phaseDistribution,
    slowestNode,
    expensiveNode
  };
}

function calculateNodeDuration(startStr?: string, endStr?: string) {
  if (!startStr || !endStr) return 0;
  return Math.round((new Date(endStr).getTime() - new Date(startStr).getTime()) / 1000);
}

function getPhaseColor(phase: string) {
  switch (phase) {
    case 'Succeeded':
      return '#52c41a';
    case 'Failed':
      return '#f5222d';
    case 'Running':
      return '#1890ff';
    default:
      return '#999';
  }
}

function getStatusBadge(status: string) {
  const colors: Record<string, string> = {
    Succeeded: 'green',
    Failed: 'red',
    Running: 'blue',
    Pending: 'default',
    Skipped: 'default'
  };
  return <span style={{ color: colors[status] || '#999' }}>●</span>;
}
