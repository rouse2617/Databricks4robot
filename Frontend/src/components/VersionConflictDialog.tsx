import { Modal, Button, Space, Divider, Alert } from 'antd';
import { ExclamationCircleOutlined } from '@ant-design/icons';
import type { PipelineTemplate } from '../api/pipelineApi';

interface VersionConflictDialogProps {
  open: boolean;
  yourVersion: PipelineTemplate;
  latestVersion: PipelineTemplate;
  onViewDiff: () => void;
  onOverwrite: () => void;
  onExport: () => void;
  onCancel: () => void;
}

export default function VersionConflictDialog({
  open,
  yourVersion,
  latestVersion,
  onViewDiff,
  onOverwrite,
  onExport,
  onCancel
}: VersionConflictDialogProps) {
  return (
    <Modal
      title="⚠️ 版本冲突"
      open={open}
      onCancel={onCancel}
      footer={null}
      width={600}
      centered
    >
      <div style={{ display: 'grid', gap: '16px' }}>
        <Alert
          type="warning"
          message="您正在编辑的版本已被他人修改"
          description="请选择以下操作之一来解决冲突"
          icon={<ExclamationCircleOutlined />}
          showIcon
        />

        {/* Version comparison */}
        <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '16px' }}>
          <div>
            <h4>您编辑的版本</h4>
            <div style={{ fontSize: '12px', padding: '12px', backgroundColor: '#fafafa', borderRadius: '4px' }}>
              <div><strong>版本 {yourVersion.version}</strong></div>
              <div style={{ color: '#666', marginTop: '4px' }}>
                修改于：{yourVersion.updatedAt?.slice(0, 19).replace('T', ' ')}
              </div>
              <div style={{ color: '#666', marginTop: '4px' }}>
                节点数：{yourVersion.nodeCount} 步骤
              </div>
            </div>
          </div>

          <div>
            <h4>最新版本</h4>
            <div style={{ fontSize: '12px', padding: '12px', backgroundColor: '#f0f9ff', borderRadius: '4px' }}>
              <div><strong>版本 {latestVersion.version}</strong></div>
              <div style={{ color: '#666', marginTop: '4px' }}>
                修改于：{latestVersion.updatedAt?.slice(0, 19).replace('T', ' ')}
              </div>
              <div style={{ color: '#666', marginTop: '4px' }}>
                节点数：{latestVersion.nodeCount} 步骤
              </div>
            </div>
          </div>
        </div>

        <Divider />

        {/* Action buttons */}
        <div style={{ display: 'grid', gap: '12px' }}>
          <div>
            <h4 style={{ marginBottom: '8px' }}>选择操作：</h4>
            <Space direction="vertical" style={{ width: '100%' }}>
              <Button
                block
                type="default"
                onClick={onViewDiff}
              >
                📊 查看差异（推荐）
              </Button>
              <div style={{ fontSize: '12px', color: '#666' }}>
                对比您的版本和最新版本，查看具体改动
              </div>
            </Space>
          </div>

          <Divider />

          <div>
            <Space direction="vertical" style={{ width: '100%' }}>
              <Button
                block
                type="primary"
                danger
                onClick={onOverwrite}
              >
                ⚡ 覆盖最新版本
              </Button>
              <div style={{ fontSize: '12px', color: '#666' }}>
                使用您的编辑覆盖最新版本（谨慎操作，会丢失他人修改）
              </div>
            </Space>
          </div>

          <Divider />

          <div>
            <Space direction="vertical" style={{ width: '100%' }}>
              <Button
                block
                onClick={onExport}
              >
                💾 导出我的版本
              </Button>
              <div style={{ fontSize: '12px', color: '#666' }}>
                导出您的编辑为 JSON 文件，稍后手动合并
              </div>
            </Space>
          </div>

          <Divider />

          <Button
            block
            onClick={onCancel}
          >
            取消
          </Button>
        </div>
      </div>
    </Modal>
  );
}
