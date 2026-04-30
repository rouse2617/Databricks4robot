import { useEffect, useState, useCallback } from "react";
import { Typography, Card, Checkbox, Space, Spin, Alert, Button, Progress, Modal } from "antd";
import { RobotOutlined, RedoOutlined } from "@ant-design/icons";
import { assetsApi } from "../api/assets";
import { algoRegistryApi, type AlgoRegistryItem } from "../api/algoRegistry";
import type { Asset } from "../api/types";
import AlgoMatrixGrid from "../components/algo-matrix/AlgoMatrixGrid";
import { getAlgoStatusFromResults, type CellStatus } from "../lib/algoStatus";
import { collectFailedPairs, useRetryAllFailed } from "../hooks/algo-matrix/useRetryAllFailed";

const { Title } = Typography;

const STATUS_OPTIONS: { label: string; value: CellStatus }[] = [
  { label: "成功", value: "ok" },
  { label: "失败", value: "failed" },
  { label: "运行中", value: "running" },
  { label: "待处理", value: "pending" },
  { label: "已阻塞", value: "blocked" },
];

export default function AlgoProcessingPage() {
  const [algorithms, setAlgorithms] = useState<AlgoRegistryItem[]>([]);
  const [assets, setAssets] = useState<Asset[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(50);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [statusFilter, setStatusFilter] = useState<CellStatus[]>([]);

  const { executing: retrying, progress: retryProgress, total: retryTotal, execute: executeRetry } = useRetryAllFailed();

  // Fetch algo registry once
  useEffect(() => {
    algoRegistryApi.list().then(setAlgorithms).catch(() => {
      setError("无法加载算法注册表");
    });
  }, []);

  // Fetch assets (paginated)
  const fetchAssets = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const res = await assetsApi.list({ page, page_size: pageSize });
      setAssets(res.items ?? []);
      setTotal(res.total);
    } catch {
      setError("加载资产数据失败");
    } finally {
      setLoading(false);
    }
  }, [page, pageSize]);

  useEffect(() => {
    fetchAssets();
  }, [fetchAssets]);

  // Client-side status filtering
  const filteredAssets =
    statusFilter.length === 0
      ? assets
      : assets.filter((asset) =>
          algorithms.some((algo) => {
            const s = getAlgoStatusFromResults(asset.algo_results, algo.key);
            return statusFilter.includes(s);
          }),
        );

  const handlePageChange = (newPage: number, newSize: number) => {
    setPage(newPage);
    setPageSize(newSize);
  };

  // Collect failed pairs from currently displayed (filtered) assets
  const failedPairs = collectFailedPairs(filteredAssets, algorithms);

  const handleRetryAllFailed = async () => {
    if (failedPairs.length === 0) return;
    const result = await executeRetry(failedPairs);
    Modal.info({
      title: "批量重试完成",
      content: (
        <div>
          <p>成功：{result.success} 个</p>
          <p>失败：{result.failed} 个</p>
        </div>
      ),
      okText: "确定",
      onOk: fetchAssets,
    });
  };

  return (
    <div>
      <Title level={4} style={{ margin: 0, marginBottom: 16 }}>
        <RobotOutlined style={{ marginRight: 8 }} />
        算法处理矩阵
      </Title>

      {error && (
        <Alert
          message={error}
          type="error"
          showIcon
          closable
          style={{ marginBottom: 16 }}
        />
      )}

      <Card
        size="small"
        style={{ marginBottom: 16 }}
        styles={{ body: { padding: "8px 16px" } }}
      >
        <Space wrap style={{ width: "100%", justifyContent: "space-between" }}>
          <Space wrap>
            <span style={{ fontWeight: 500 }}>状态过滤：</span>
            <Checkbox.Group
              options={STATUS_OPTIONS}
              value={statusFilter}
              onChange={(vals) => setStatusFilter(vals as CellStatus[])}
            />
          </Space>
          <Button
            type="primary"
            danger
            icon={<RedoOutlined />}
            loading={retrying}
            disabled={failedPairs.length === 0 || loading}
            onClick={handleRetryAllFailed}
          >
            重试全部失败{failedPairs.length > 0 ? ` (${failedPairs.length})` : ""}
          </Button>
        </Space>
        {retrying && (
          <Progress
            percent={retryProgress}
            status="active"
            size="small"
            style={{ marginTop: 8 }}
            format={() => `${Math.round((retryProgress / 100) * retryTotal)}/${retryTotal}`}
          />
        )}
      </Card>

      {loading && algorithms.length === 0 ? (
        <div style={{ textAlign: "center", padding: 60 }}>
          <Spin size="large" />
        </div>
      ) : (
        <AlgoMatrixGrid
          assets={filteredAssets}
          algorithms={algorithms}
          loading={loading}
          total={total}
          page={page}
          pageSize={pageSize}
          onPageChange={handlePageChange}
          onRefresh={fetchAssets}
        />
      )}
    </div>
  );
}
