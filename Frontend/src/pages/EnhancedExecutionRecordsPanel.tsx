import { useState, useMemo } from 'react';
import { Tabs, Empty, Spin, Button, Space, Input } from 'antd';
import { ReloadOutlined, SearchOutlined } from '@ant-design/icons';
import ExecutionTimeline from '../components/ExecutionTimeline';
import ExecutionMetrics from '../components/ExecutionMetrics';
import { listPipelineRuns, type PipelineRun } from '../api/pipelineApi';

interface EnhancedExecutionRecordsPanelProps {
  active?: boolean;
  defaultRunId?: string;
}

export function EnhancedExecutionRecordsPanel({
  active = true,
  defaultRunId
}: EnhancedExecutionRecordsPanelProps) {
  const [runs, setRuns] = useState<PipelineRun[]>([]);
  const [selectedRunId, setSelectedRunId] = useState<string | undefined>(defaultRunId);
  const [searchQuery, setSearchQuery] = useState('');
  const [loading, setLoading] = useState(false);

  // Load runs
  const loadRuns = async () => {
    setLoading(true);
    try {
      const response = await listPipelineRuns({ pageSize: 50 });
      setRuns(response.items);
    } catch (error) {
      console.error('Failed to load runs:', error);
      setRuns([]);
    } finally {
      setLoading(false);
    }
  };

  // Auto-load on mount if active
  if (active && runs.length === 0 && !loading) {
    loadRuns();
  }

  // Filter runs by search query
  const filteredRuns = useMemo(() => {
    if (!searchQuery) return runs;
    const q = searchQuery.toLowerCase();
    return runs.filter(
      (r) =>
        r.workflowName?.toLowerCase().includes(q) ||
        r.id.toLowerCase().includes(q) ||
        r.status?.toLowerCase().includes(q)
    );
  }, [runs, searchQuery]);

  // Get selected run
  const selectedRun = useMemo(
    () => runs.find((r) => r.id === selectedRunId),
    [runs, selectedRunId]
  );

  return (
    <div style={{ display: 'grid', gridTemplateColumns: '300px 1fr', gap: '16px', padding: '16px', height: '100%' }}>
      {/* Left panel: Run list */}
      <div style={{ display: 'flex', flexDirection: 'column', gap: '12px', borderRight: '1px solid #d9d9d9', paddingRight: '12px', maxHeight: '80vh', overflow: 'auto' }}>
        <div>
          <h3 style={{ marginBottom: '12px' }}>执行记录</h3>
          <Space style={{ width: '100%', marginBottom: '12px' }}>
            <Input
              placeholder="搜索..."
              prefix={<SearchOutlined />}
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              style={{ flex: 1 }}
              size="small"
            />
            <Button
              icon={<ReloadOutlined />}
              size="small"
              loading={loading}
              onClick={loadRuns}
            />
          </Space>
        </div>

        {loading ? (
          <Spin />
        ) : filteredRuns.length === 0 ? (
          <Empty description="无执行记录" />
        ) : (
          <div style={{ display: 'grid', gap: '8px' }}>
            {filteredRuns.map((run) => (
              <div
                key={run.id}
                onClick={() => setSelectedRunId(run.id)}
                style={{
                  padding: '12px',
                  backgroundColor: selectedRunId === run.id ? '#e6f7ff' : '#fafafa',
                  border: selectedRunId === run.id ? '1px solid #1890ff' : '1px solid #d9d9d9',
                  borderRadius: '4px',
                  cursor: 'pointer',
                  transition: 'all 0.2s'
                }}
              >
                <div style={{ fontWeight: 500, marginBottom: '4px', fontSize: '12px' }}>
                  {run.workflowName}
                </div>
                <div style={{ fontSize: '11px', color: '#666', marginBottom: '4px' }}>
                  ID: {run.id.slice(0, 12)}...
                </div>
                <div style={{ fontSize: '11px' }}>
                  <span
                    style={{
                      display: 'inline-block',
                      padding: '2px 6px',
                      backgroundColor: getStatusColor(run.status),
                      color: '#fff',
                      borderRadius: '2px',
                      marginRight: '4px'
                    }}
                  >
                    {getStatusLabel(run.status)}
                  </span>
                </div>
              </div>
            ))}
          </div>
        )}
      </div>

      {/* Right panel: Run details */}
      <div style={{ overflow: 'auto' }}>
        {!selectedRun ? (
          <Empty
            description="选择一个执行记录查看详情"
            style={{ marginTop: '60px' }}
          />
        ) : (
          <Tabs
            items={[
              {
                key: 'timeline',
                label: '⏱️ 执行时间轴',
                children: (
                  <div style={{ marginTop: '16px' }}>
                    <ExecutionTimeline
                      nodes={selectedRun.nodes || []}
                      loading={false}
                    />
                  </div>
                )
              },
              {
                key: 'metrics',
                label: '📊 性能指标',
                children: (
                  <div style={{ marginTop: '16px' }}>
                    <ExecutionMetrics
                      nodes={selectedRun.nodes || []}
                      totalEstimatedCost={selectedRun.totalEstimatedCost}
                      loading={false}
                    />
                  </div>
                )
              },
              {
                key: 'details',
                label: '📋 基本信息',
                children: (
                  <div style={{ marginTop: '16px', fontSize: '12px' }}>
                    <table style={{ width: '100%' }}>
                      <tbody>
                        <tr>
                          <td style={{ paddingRight: '24px', color: '#666' }}>ID:</td>
                          <td>{selectedRun.id}</td>
                        </tr>
                        <tr>
                          <td style={{ paddingRight: '24px', color: '#666' }}>状态:</td>
                          <td>{selectedRun.status}</td>
                        </tr>
                        <tr>
                          <td style={{ paddingRight: '24px', color: '#666' }}>创建时间:</td>
                          <td>{selectedRun.createdAt?.slice(0, 19).replace('T', ' ')}</td>
                        </tr>
                        <tr>
                          <td style={{ paddingRight: '24px', color: '#666' }}>完成时间:</td>
                          <td>{selectedRun.finishedAt?.slice(0, 19).replace('T', ' ') || '-'}</td>
                        </tr>
                        <tr>
                          <td style={{ paddingRight: '24px', color: '#666' }}>资产数:</td>
                          <td>{selectedRun.assetCount || 0}</td>
                        </tr>
                        <tr>
                          <td style={{ paddingRight: '24px', color: '#666' }}>节点数:</td>
                          <td>{selectedRun.nodes?.length || 0}</td>
                        </tr>
                      </tbody>
                    </table>
                  </div>
                )
              }
            ]}
          />
        )}
      </div>
    </div>
  );
}

function getStatusColor(status?: string): string {
  switch (status) {
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

function getStatusLabel(status?: string): string {
  const labels: Record<string, string> = {
    Succeeded: '成功',
    Failed: '失败',
    Running: '运行中',
    Pending: '待执行'
  };
  return labels[status || ''] || status || '未知';
}
