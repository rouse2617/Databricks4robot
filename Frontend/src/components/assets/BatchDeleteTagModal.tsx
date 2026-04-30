// ─── BatchDeleteTagModal — Batch delete a tag from selected assets ───
// Validates: Requirements REQ-3.1.4, REQ-3.1.5

import { useState, useEffect, useCallback } from "react";
import { Modal, Select, Progress, Typography, Space, message } from "antd";
import { tagRegistryApi, type TagRegistryItem } from "../../api/tagRegistry";
import { assetsApi } from "../../api/assets";
import type { BatchTagResult } from "./BatchTagModal";

const { Text } = Typography;

export interface BatchDeleteTagModalProps {
  open: boolean;
  assetIds: string[];
  onClose: () => void;
  onComplete: (result: BatchTagResult) => void;
}

/** Run async tasks with concurrency limit */
async function runWithConcurrency<T>(
  tasks: (() => Promise<T>)[],
  limit: number,
  onProgress: () => void,
): Promise<T[]> {
  const results: T[] = new Array(tasks.length);
  let idx = 0;

  async function worker() {
    while (idx < tasks.length) {
      const i = idx++;
      results[i] = await tasks[i]();
      onProgress();
    }
  }

  await Promise.all(Array.from({ length: Math.min(limit, tasks.length) }, () => worker()));
  return results;
}

export default function BatchDeleteTagModal({
  open,
  assetIds,
  onClose,
  onComplete,
}: BatchDeleteTagModalProps) {
  const [tags, setTags] = useState<TagRegistryItem[]>([]);
  const [loading, setLoading] = useState(false);
  const [selectedKey, setSelectedKey] = useState<string | undefined>();
  const [executing, setExecuting] = useState(false);
  const [progress, setProgress] = useState(0);
  const [msg, msgCtx] = message.useMessage();

  // Fetch tag registry on open
  useEffect(() => {
    if (!open) return;
    setLoading(true);
    tagRegistryApi
      .list()
      .then(setTags)
      .catch(() => msg.error("获取标签注册表失败"))
      .finally(() => setLoading(false));
  }, [open]);

  // Reset form when modal opens
  useEffect(() => {
    if (open) {
      setSelectedKey(undefined);
      setProgress(0);
      setExecuting(false);
    }
  }, [open]);

  const canSubmit = !!selectedKey && !executing;

  const handleExecute = useCallback(async () => {
    if (!selectedKey) return;

    setExecuting(true);
    setProgress(0);

    let success = 0;
    let skipped = 0;
    let failed = 0;
    let completed = 0;

    const tasks = assetIds.map((assetId) => async () => {
      try {
        await assetsApi.deleteTag(assetId, selectedKey);
        success++;
      } catch {
        failed++;
      }
    });

    await runWithConcurrency(tasks, 5, () => {
      completed++;
      setProgress(Math.round((completed / assetIds.length) * 100));
    });

    setExecuting(false);
    onComplete({ success, skipped, failed });
  }, [selectedKey, assetIds, onComplete]);

  return (
    <Modal
      title="批量删除标签"
      open={open}
      onOk={handleExecute}
      onCancel={onClose}
      okText={executing ? "执行中..." : "确认删除"}
      cancelText="取消"
      okButtonProps={{ disabled: !canSubmit, danger: true }}
      closable={!executing}
      maskClosable={!executing}
      destroyOnHidden
    >
      {msgCtx}

      <div style={{ background: "#fff2e8", border: "1px solid #ffbb96", borderRadius: 4, padding: "8px 12px", marginBottom: 16, fontSize: 13 }}>
        将从 <Text strong>{assetIds.length}</Text> 个资产中删除所选标签
      </div>

      <Space direction="vertical" style={{ width: "100%" }} size={16}>
        {/* Tag Key Select */}
        <div>
          <Text style={{ display: "block", marginBottom: 4 }}>选择要删除的标签 Key</Text>
          <Select
            style={{ width: "100%" }}
            placeholder="选择标签 Key"
            loading={loading}
            value={selectedKey}
            onChange={setSelectedKey}
            options={tags.map((t) => ({ label: `${t.key} (${t.type})`, value: t.key }))}
          />
        </div>

        {/* Progress Bar */}
        {executing && <Progress percent={progress} status="active" />}
      </Space>
    </Modal>
  );
}
