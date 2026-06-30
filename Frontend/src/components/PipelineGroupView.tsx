import { useState, useEffect } from 'react';
import { Empty, Skeleton, Button, Input, Space, Tag } from 'antd';
import { ReloadOutlined, SearchOutlined } from '@ant-design/icons';
import { listPipelines, type PipelineTemplate } from '../api/pipelineApi';

interface GroupedPipelines {
  title: string;
  icon: string;
  pipelines: PipelineTemplate[];
}

interface PipelineGroupViewProps {
  onSelectPipeline: (pipeline: PipelineTemplate) => void;
}

export function PipelineGroupView({ onSelectPipeline }: PipelineGroupViewProps) {
  const [groups, setGroups] = useState<GroupedPipelines[]>([]);
  const [loading, setLoading] = useState(true);
  const [searchQuery, setSearchQuery] = useState('');

  useEffect(() => {
    loadPipelines();
  }, []);

  const loadPipelines = async () => {
    setLoading(true);
    try {
      const pipelines = await listPipelines({ pageSize: 200 }).then(r => r.items);

      // Group by frequency (simplified: first 5 = "most used", rest = "other")
      const mostUsed = pipelines.slice(0, 5);
      const others = pipelines.slice(5);

      setGroups([
        { title: '🔥 最常用', icon: '🔥', pipelines: mostUsed },
        { title: '📋 全部', icon: '📋', pipelines: others.length > 0 ? others : pipelines }
      ]);
    } catch (err) {
      console.error('Failed to load pipelines:', err);
      setGroups([]);
    } finally {
      setLoading(false);
    }
  };

  const filteredGroups = groups.map(group => ({
    ...group,
    pipelines: group.pipelines.filter(p =>
      p.name.toLowerCase().includes(searchQuery.toLowerCase())
    )
  })).filter(g => g.pipelines.length > 0);

  if (loading) {
    return <Skeleton active paragraph={{ rows: 6 }} />;
  }

  if (filteredGroups.length === 0 && searchQuery) {
    return <Empty description="未找到匹配的流水线" />;
  }

  return (
    <div className="pipeline-group-view" style={{ padding: '16px' }}>
      <Space direction="vertical" style={{ width: '100%', gap: '16px' }}>
        <Space>
          <Input
            placeholder="搜索流水线..."
            prefix={<SearchOutlined />}
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            style={{ width: 220 }}
          />
          <Button icon={<ReloadOutlined />} onClick={loadPipelines} loading={loading} />
        </Space>

        {filteredGroups.map((group) => (
          <div key={group.title}>
            <h3>{group.title}</h3>
            <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(250px, 1fr))', gap: '12px' }}>
              {group.pipelines.map((pipeline) => (
                <div
                  key={pipeline.id}
                  onClick={() => onSelectPipeline(pipeline)}
                  style={{
                    padding: '12px',
                    border: '1px solid #d9d9d9',
                    borderRadius: '4px',
                    cursor: 'pointer',
                    transition: 'all 0.2s'
                  }}
                  onMouseEnter={(e) => e.currentTarget.style.boxShadow = '0 2px 8px rgba(0,0,0,0.12)'}
                  onMouseLeave={(e) => e.currentTarget.style.boxShadow = 'none'}
                >
                  <div style={{ fontWeight: 600, marginBottom: '4px' }}>{pipeline.name}</div>
                  <div style={{ fontSize: '12px', color: '#999', marginBottom: '8px' }}>
                    ID: {pipeline.id.slice(0, 8)}...
                  </div>
                  <div style={{ display: 'flex', gap: '4px', flexWrap: 'wrap' }}>
                    <Tag color="blue">v{pipeline.version}</Tag>
                    {pipeline.nodeCount && <Tag>{pipeline.nodeCount} 步骤</Tag>}
                  </div>
                </div>
              ))}
            </div>
          </div>
        ))}
      </Space>
    </div>
  );
}
