import { useState } from 'react';
import { Button, Space, Input, Empty, Spin } from 'antd';
import { ReloadOutlined, SearchOutlined } from '@ant-design/icons';
import PipelineGroupView from '../components/PipelineGroupView';
import { usePipelineStats } from '../hooks/usePipelineStats';
import { groupPipelinesWithStats, filterPipelines, type GroupedPipelineSet } from '../lib/pipelineGrouping';
import { listPipelines, type PipelineTemplate } from '../api/pipelineApi';

interface PipelineListPanelProps {
  onSelectPipeline?: (pipeline: PipelineTemplate) => void;
}

export function PipelineListPanel({ onSelectPipeline }: PipelineListPanelProps) {
  const [searchQuery, setSearchQuery] = useState('');
  const [allPipelines, setAllPipelines] = useState<PipelineTemplate[]>([]);
  const [groupedPipelines, setGroupedPipelines] = useState<GroupedPipelineSet[]>([]);
  const [loadingPipelines, setLoadingPipelines] = useState(true);

  // Fetch stats with cache
  const { stats, loading: loadingStats } = usePipelineStats({
    window: '30d',
    refetchInterval: 5 * 60 * 1000 // 5 minutes
  });

  // Load all pipelines
  const loadPipelines = async () => {
    setLoadingPipelines(true);
    try {
      const response = await listPipelines({ pageSize: 200 });
      setAllPipelines(response.items);

      // Group with stats
      const grouped = groupPipelinesWithStats(response.items, stats);
      setGroupedPipelines(grouped);
    } catch (error) {
      console.error('Failed to load pipelines:', error);
      setGroupedPipelines([
        {
          title: '📋 全部流水线',
          icon: '📋',
          pipelines: allPipelines,
          count: allPipelines.length
        }
      ]);
    } finally {
      setLoadingPipelines(false);
    }
  };

  // Initial load
  if (allPipelines.length === 0 && !loadingPipelines) {
    loadPipelines();
  }

  const loading = loadingPipelines || loadingStats;

  // Filter grouped pipelines by search query
  const filtered = searchQuery
    ? filterPipelines(groupedPipelines, searchQuery)
    : groupedPipelines;

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: '16px', padding: '16px' }}>
      {/* Header with search and reload */}
      <Space style={{ width: '100%', justifyContent: 'space-between' }}>
        <Input
          placeholder="搜索流水线..."
          prefix={<SearchOutlined />}
          value={searchQuery}
          onChange={(e) => setSearchQuery(e.target.value)}
          style={{ width: 300 }}
        />
        <Button
          icon={<ReloadOutlined />}
          loading={loading}
          onClick={loadPipelines}
        >
          刷新
        </Button>
      </Space>

      {/* Content */}
      {loading ? (
        <div style={{ textAlign: 'center', padding: '60px 0' }}>
          <Spin tip="加载流水线中..." />
        </div>
      ) : filtered.length === 0 ? (
        <Empty
          description={searchQuery ? '未找到匹配的流水线' : '暂无流水线'}
          style={{ marginTop: '60px' }}
        />
      ) : (
        <div style={{ display: 'grid', gap: '24px' }}>
          {filtered.map((group) => (
            <div key={group.title}>
              <h3 style={{ fontSize: '16px', marginBottom: '12px' }}>
                {group.icon} {group.title} ({group.count})
              </h3>
              <div
                style={{
                  display: 'grid',
                  gridTemplateColumns: 'repeat(auto-fill, minmax(250px, 1fr))',
                  gap: '12px'
                }}
              >
                {group.pipelines.map((pipeline) => (
                  <div
                    key={pipeline.id}
                    onClick={() => onSelectPipeline?.(pipeline)}
                    style={{
                      padding: '12px',
                      border: '1px solid #d9d9d9',
                      borderRadius: '4px',
                      cursor: 'pointer',
                      transition: 'all 0.2s',
                      backgroundColor: '#fafafa'
                    }}
                    onMouseEnter={(e) => {
                      e.currentTarget.style.boxShadow = '0 2px 8px rgba(0,0,0,0.15)';
                      e.currentTarget.style.backgroundColor = '#fff';
                    }}
                    onMouseLeave={(e) => {
                      e.currentTarget.style.boxShadow = 'none';
                      e.currentTarget.style.backgroundColor = '#fafafa';
                    }}
                  >
                    <div style={{ fontWeight: 600, marginBottom: '4px' }}>
                      {pipeline.name}
                    </div>
                    <div style={{ fontSize: '12px', color: '#666', marginBottom: '8px' }}>
                      ID: {pipeline.id.slice(0, 8)}...
                    </div>
                    <div style={{ display: 'flex', gap: '4px', flexWrap: 'wrap' }}>
                      <span
                        style={{
                          display: 'inline-block',
                          padding: '2px 8px',
                          backgroundColor: '#e6f7ff',
                          color: '#1890ff',
                          borderRadius: '2px',
                          fontSize: '12px'
                        }}
                      >
                        v{pipeline.version}
                      </span>
                      {pipeline.nodeCount && (
                        <span
                          style={{
                            display: 'inline-block',
                            padding: '2px 8px',
                            backgroundColor: '#f0f0f0',
                            color: '#666',
                            borderRadius: '2px',
                            fontSize: '12px'
                          }}
                        >
                          {pipeline.nodeCount} 步骤
                        </span>
                      )}
                    </div>
                  </div>
                ))}
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
