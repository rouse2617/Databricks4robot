import { useState, useMemo } from 'react';
import { Select, Empty, Space, Tag, Button, Tooltip } from 'antd';
import { CopyOutlined, EyeOutlined, DeleteOutlined } from '@ant-design/icons';
import type { PipelineTemplate } from '../api/pipelineApi';

interface VersionSelectorProps {
  versions: PipelineTemplate[];
  selectedVersion?: number;
  onVersionChange: (version: PipelineTemplate) => void;
  userTags?: Record<string, string[]>;
  onCompare?: (v1: PipelineTemplate, v2: PipelineTemplate) => void;
  onDelete?: (version: PipelineTemplate) => void;
}

export default function VersionSelector({
  versions,
  selectedVersion,
  onVersionChange,
  userTags = {},
  onCompare,
  onDelete
}: VersionSelectorProps) {
  const [searchQuery, setSearchQuery] = useState('');
  const [compareMode, setCompareMode] = useState(false);
  const [compareVersion, setCompareVersion] = useState<number | undefined>();

  // Group versions by category
  const groupedVersions = useMemo(() => {
    const tagged = new Set(Object.values(userTags).flat());

    return {
      critical: versions.filter(v => v.activeVersion && tagged.has(v.id)),
      myTags: versions.filter(v => !v.activeVersion && tagged.has(v.id)),
      all: versions.filter(v => !tagged.has(v.id))
    };
  }, [versions, userTags]);

  // Filter by search query
  const filteredGroups = useMemo(() => {
    if (!searchQuery) return groupedVersions;

    const q = searchQuery.toLowerCase();
    return {
      critical: groupedVersions.critical.filter(v =>
        v.updatedAt?.toLowerCase().includes(q) || v.activeVersion
      ),
      myTags: groupedVersions.myTags.filter(v =>
        v.updatedAt?.toLowerCase().includes(q)
      ),
      all: groupedVersions.all.filter(v =>
        v.updatedAt?.toLowerCase().includes(q)
      )
    };
  }, [groupedVersions, searchQuery]);

  const options = [
    {
      label: '🔑 关键版本',
      options: filteredGroups.critical.map(v => ({
        label: renderVersionLabel(v, true),
        value: v.version
      }))
    },
    {
      label: '📌 我的标记',
      options: filteredGroups.myTags.map(v => ({
        label: renderVersionLabel(v),
        value: v.version
      }))
    },
    {
      label: '📋 所有版本',
      options: filteredGroups.all.map(v => ({
        label: renderVersionLabel(v),
        value: v.version
      }))
    }
  ].filter(g => g.options.length > 0);

  const selectedVer = versions.find(v => v.version === selectedVersion);

  return (
    <div style={{ display: 'flex', gap: '12px', alignItems: 'flex-start', flexWrap: 'wrap' }}>
      <div style={{ flex: 1, minWidth: '250px' }}>
        <Select
          placeholder="选择版本..."
          value={selectedVersion}
          onChange={(val) => {
            const ver = versions.find(v => v.version === val);
            if (ver) onVersionChange(ver);
          }}
          onSearch={setSearchQuery}
          searchValue={searchQuery}
          options={options}
          filterOption={false}
          optionLabelProp="label"
          style={{ width: '100%' }}
        />
      </div>

      {selectedVer && (
        <Space size="small">
          {onCompare && (
            <>
              <Button
                size="small"
                icon={<EyeOutlined />}
                onClick={() => setCompareMode(!compareMode)}
              >
                {compareMode ? '关闭对比' : '对比'}
              </Button>
              {compareMode && compareVersion && compareVersion !== selectedVersion && (
                <Tooltip title="查看两个版本的差异">
                  <Button
                    size="small"
                    type="primary"
                    onClick={() => {
                      const ver = versions.find(v => v.version === compareVersion);
                      if (ver) onCompare(selectedVer, ver);
                    }}
                  >
                    查看差异
                  </Button>
                </Tooltip>
              )}
            </>
          )}

          {onDelete && (
            <Button
              size="small"
              danger
              icon={<DeleteOutlined />}
              onClick={() => onDelete(selectedVer)}
            >
              删除
            </Button>
          )}
        </Space>
      )}

      {compareMode && (
        <div style={{ minWidth: '250px' }}>
          <Select
            placeholder="选择对比版本..."
            value={compareVersion}
            onChange={setCompareVersion}
            options={versions
              .filter(v => v.version !== selectedVersion)
              .map(v => ({
                label: `v${v.version} - ${v.updatedAt?.slice(0, 10) || '未知'}`,
                value: v.version
              }))}
            style={{ width: '100%' }}
          />
        </div>
      )}
    </div>
  );
}

function renderVersionLabel(version: PipelineTemplate, isCritical = false) {
  return (
    <div style={{ display: 'flex', gap: '8px', alignItems: 'center' }}>
      <span>{`v${version.version}`}</span>
      {isCritical && <Tag color="gold">活跃</Tag>}
      <span style={{ fontSize: '12px', color: '#999' }}>
        {version.updatedAt?.slice(0, 10) || '未知'}
      </span>
    </div>
  );
}
