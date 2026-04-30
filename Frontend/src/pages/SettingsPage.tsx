import { useState } from "react";
import { Typography, Card, Descriptions, Button, Alert, Space, Spin, Modal } from "antd";
import { SettingOutlined, ThunderboltOutlined, ExclamationCircleOutlined } from "@ant-design/icons";
import { useAuth } from "../hooks/useAuth";
import { adminApi, type ReindexResult } from "../api/admin";
import { extractApiErrorMessage } from "../lib/apiError";

const { Title, Text } = Typography;

type ReindexPhase = "idle" | "dry_running" | "dry_done" | "reindexing" | "done" | "error";

export default function SettingsPage() {
  const { token } = useAuth();
  const [phase, setPhase] = useState<ReindexPhase>("idle");
  const [dryResult, setDryResult] = useState<ReindexResult | null>(null);
  const [result, setResult] = useState<ReindexResult | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [confirmOpen, setConfirmOpen] = useState(false);

  const handleDryRun = async () => {
    setPhase("dry_running");
    setError(null);
    setDryResult(null);
    setResult(null);
    try {
      const res = await adminApi.reindex(true);
      setDryResult(res);
      setPhase("dry_done");
    } catch (err) {
      setError(extractApiErrorMessage(err, "Dry run 失败"));
      setPhase("error");
    }
  };

  const handleConfirmReindex = () => setConfirmOpen(true);

  const handleReindexSubmit = async () => {
    setPhase("reindexing");
    setError(null);
    try {
      const res = await adminApi.reindex(false);
      setResult(res);
      setPhase("done");
      setConfirmOpen(false);
    } catch (err) {
      setError(extractApiErrorMessage(err, "重建索引失败"));
      setPhase("error");
      setConfirmOpen(false);
    }
  };

  const handleReset = () => {
    setPhase("idle");
    setDryResult(null);
    setResult(null);
    setError(null);
    setConfirmOpen(false);
  };

  return (
    <div>
      <Title level={4} style={{ margin: 0, marginBottom: 16 }}>
        <SettingOutlined style={{ marginRight: 8 }} />
        设置
      </Title>
      <Card title="当前会话" size="small" style={{ marginBottom: 16 }}>
        <Descriptions column={1} size="small">
          <Descriptions.Item label="认证方式">X-Grace-Token (Phase 0)</Descriptions.Item>
          <Descriptions.Item label="Token">
            <code className="text-xs">{token ? `${token.slice(0, 8)}...` : "—"}</code>
          </Descriptions.Item>
          <Descriptions.Item label="API 地址">/api/v1</Descriptions.Item>
        </Descriptions>
      </Card>

      {/* Task 8.3: Rebuild ES Index */}
      <Card
        title={
          <span>
            <ThunderboltOutlined style={{ marginRight: 8 }} />
            重建 ES 索引
          </span>
        }
        size="small"
        data-testid="reindex-card"
      >
        <Text type="secondary" style={{ display: "block", marginBottom: 12 }}>
          当 Elasticsearch 索引与 PostgreSQL 数据不一致时，可手动触发全量重建。
          首先执行 Dry Run 预览影响范围，确认后再执行实际重建。
        </Text>

        {error && (
          <Alert
            type="error"
            showIcon
            message={error}
            style={{ marginBottom: 12 }}
            closable
            onClose={() => setError(null)}
          />
        )}

        {/* Dry run result */}
        {dryResult && phase === "dry_done" && (
          <Alert
            type="info"
            showIcon
            message="Dry Run 完成"
            description={
              <div>
                <p>将重建 <strong>{dryResult.total_assets}</strong> 个资产的索引</p>
                <p>预计耗时：{(dryResult.duration_ms / 1000).toFixed(1)} 秒</p>
              </div>
            }
            style={{ marginBottom: 12 }}
          />
        )}

        {/* Actual reindex result */}
        {result && phase === "done" && (
          <Alert
            type="success"
            showIcon
            message="索引重建完成"
            description={
              <div>
                <p>总资产数：{result.total_assets}</p>
                <p>成功索引：{result.indexed}</p>
                <p>删除文档：{result.deleted}</p>
                <p>失败：{result.failed}</p>
                <p>耗时：{(result.duration_ms / 1000).toFixed(1)} 秒</p>
              </div>
            }
            style={{ marginBottom: 12 }}
          />
        )}

        {/* Loading state */}
        {(phase === "dry_running" || (phase === "reindexing" && !confirmOpen)) && (
          <div style={{ textAlign: "center", padding: "16px 0" }}>
            <Spin />
            <div style={{ marginTop: 8 }}>
              <Text type="secondary">
                {phase === "dry_running" ? "正在执行 Dry Run..." : "正在重建索引..."}
              </Text>
            </div>
          </div>
        )}

        <Space>
          {phase === "idle" && (
            <Button
              type="primary"
              onClick={handleDryRun}
              data-testid="reindex-dry-run-btn"
            >
              Dry Run（预览）
            </Button>
          )}
          {phase === "dry_done" && (
            <>
              <Button
                type="primary"
                danger
                onClick={handleConfirmReindex}
                data-testid="reindex-confirm-btn"
              >
                确认重建
              </Button>
              <Button onClick={handleReset}>取消</Button>
            </>
          )}
          {(phase === "done" || phase === "error") && (
            <Button onClick={handleReset}>重置</Button>
          )}
        </Space>

        <Modal
          title={
            <span>
              <ExclamationCircleOutlined style={{ color: "var(--color-warning, #faad14)", marginRight: 8 }} />
              确认重建 ES 索引
            </span>
          }
          open={confirmOpen}
          okText="确认重建"
          cancelText="取消"
          okButtonProps={{ danger: true }}
          confirmLoading={phase === "reindexing"}
          onOk={handleReindexSubmit}
          onCancel={() => setConfirmOpen(false)}
          destroyOnClose
        >
          <div>
            <p>即将对 <strong>{dryResult?.total_assets ?? 0}</strong> 个资产重建 Elasticsearch 索引。</p>
            <p>此操作可能需要数分钟，期间搜索结果可能不完整。</p>
            <p>确定继续？</p>
          </div>
        </Modal>
      </Card>
    </div>
  );
}
