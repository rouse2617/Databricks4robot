import { useEffect, useState } from "react";
import { Card, Col, Row, Table, Tag, Typography, Segmented, Spin, Space, Alert } from "antd";
import {
  DatabaseOutlined,
  FileOutlined,
  CheckCircleOutlined,
  SendOutlined,
  WarningOutlined,
  ArrowUpOutlined,
  ArrowDownOutlined,
  RightOutlined,
} from "@ant-design/icons";
import { useNavigate } from "react-router-dom";
import type { ColumnsType } from "antd/es/table";
import dayjs from "dayjs";
import { mcapFilesApi } from "../api/mcapFiles";
import { assetsApi } from "../api/assets";
import type { Asset } from "../api/types";
import { formatDurationSeconds } from "../lib/assetPresentation";

const { Title, Text } = Typography;

const algoStatusColor: Record<string, string> = {
  ok: "success", failed: "error", running: "warning", pending: "default", blocked: "default",
};

function extractAlgoStatuses(algo: Record<string, string> | undefined) {
  if (!algo) return [];
  const statuses: { key: string; status: string }[] = [];
  for (const [k, v] of Object.entries(algo)) {
    if (k.endsWith(":status")) {
      statuses.push({ key: k.replace(":status", ""), status: v });
    }
  }
  return statuses;
}

/** KPI Card component with SaaS styling */
function KpiCard({
  title, value, suffix, icon, trend, trendLabel, onClick, color,
}: {
  title: string;
  value: string | number;
  suffix?: string;
  icon: React.ReactNode;
  trend?: "up" | "down" | null;
  trendLabel?: string;
  onClick?: () => void;
  color?: string;
}) {
  return (
    <Card
      hoverable={!!onClick}
      onClick={onClick}
      style={{ cursor: onClick ? "pointer" : "default" }}
      styles={{ body: { padding: "20px 24px" } }}
    >
      <div style={{ display: "flex", justifyContent: "space-between", alignItems: "flex-start" }}>
        <div>
          <Text type="secondary" style={{ fontSize: 12, fontWeight: 500, textTransform: "uppercase", letterSpacing: "0.05em" }}>
            {title}
          </Text>
          <div style={{ marginTop: 8, display: "flex", alignItems: "baseline", gap: 4 }}>
            <span style={{ fontSize: 28, fontWeight: 700, color: color ?? "#1E293B", fontVariantNumeric: "tabular-nums" }}>
              {value}
            </span>
            {suffix && <span style={{ fontSize: 14, color: "#64748B", fontWeight: 500 }}>{suffix}</span>}
          </div>
          {trendLabel && (
            <div style={{ marginTop: 6, fontSize: 12 }}>
              {trend === "up" && <ArrowUpOutlined style={{ color: "#16A34A", marginRight: 4 }} />}
              {trend === "down" && <ArrowDownOutlined style={{ color: "#DC2626", marginRight: 4 }} />}
              <Text type="secondary">{trendLabel}</Text>
            </div>
          )}
        </div>
        <div
          style={{
            width: 40, height: 40, borderRadius: 10,
            background: `${color ?? "#2563EB"}12`,
            display: "flex", alignItems: "center", justifyContent: "center",
          }}
        >
          <span style={{ color: color ?? "#2563EB", fontSize: 18 }}>{icon}</span>
        </div>
      </div>
    </Card>
  );
}

// ─── Derived helpers (frontend sampling fallback) ───

interface DerivedStats {
  total: number;
  mcapTotal: number | null;
  algoCounts: Record<string, number>;
  algoTotal: number;
  successRate: string;
  deliveryTotal: number;
  recentAssets: Asset[];
  failedAssets: Asset[];
  isSampled: boolean; // true when data comes from frontend sampling
}

function deriveFromSampling(assets: Asset[], total: number, mcapTotal: number | null): DerivedStats {
  const algoCounts: Record<string, number> = { ok: 0, failed: 0, running: 0, pending: 0, blocked: 0 };
  for (const a of assets) {
    for (const s of extractAlgoStatuses(a.algo_results)) {
      if (s.status in algoCounts) algoCounts[s.status]++;
    }
  }
  const algoTotal = Object.values(algoCounts).reduce((a, b) => a + b, 0);
  const successRate = algoTotal > 0 ? ((algoCounts.ok / algoTotal) * 100).toFixed(1) : "—";
  const deliveryTotal = assets.reduce((sum, a) => sum + (a.delivery_count ?? 0), 0);
  const failedAssets = assets.filter((a) =>
    extractAlgoStatuses(a.algo_results).some((s) => s.status === "failed")
  );

  return {
    total,
    mcapTotal,
    algoCounts,
    algoTotal,
    successRate,
    deliveryTotal,
    recentAssets: assets.slice(0, 8),
    failedAssets,
    isSampled: total > 100,
  };
}

export default function DashboardPage() {
  const navigate = useNavigate();
  const [loading, setLoading] = useState(true);
  const [derived, setDerived] = useState<DerivedStats | null>(null);
  const [dimension, setDimension] = useState<string>("资产维度");

  useEffect(() => {
    let cancelled = false;

    const load = async () => {
      const mcapPromise = mcapFilesApi
        .list({ page: 1, page_size: 1 })
        .then((d) => d.total ?? 0)
        .catch(() => null);

      try {
        const [data, mcapTotal] = await Promise.all([
          assetsApi.list({ page: 1, page_size: 100, sort_by: "-updated_at" }),
          mcapPromise,
        ]);
        if (!cancelled) {
          setDerived(deriveFromSampling(data.items ?? [], data.total ?? 0, mcapTotal));
        }
      } catch {
        if (!cancelled) setDerived(null);
      } finally {
        if (!cancelled) setLoading(false);
      }
    };

    load();
    return () => { cancelled = true; };
  }, []);

  const failedColumns: ColumnsType<Asset> = [
    {
      title: "Asset",
      dataIndex: "asset_id",
      render: (id: string) => (
        <a className="font-mono text-xs cursor-pointer" onClick={() => navigate(`/assets/${id}`)}>
          {id.slice(0, 8)}…
        </a>
      ),
    },
    {
      title: "失败算法",
      key: "failed_algo",
      render: (_: unknown, record: Asset) =>
        extractAlgoStatuses(record.algo_results)
          .filter((s) => s.status === "failed")
          .map((f) => <Tag color="error" key={f.key} style={{ fontSize: 11 }}>{f.key.split("@")[0]}</Tag>),
    },
    {
      title: "时间",
      dataIndex: "updated_at",
      width: 120,
      render: (v: string) => <Text type="secondary" style={{ fontSize: 12 }}>{dayjs(v).format("MM-DD HH:mm")}</Text>,
    },
  ];

  if (loading) {
    return (
      <div className="flex items-center justify-center" style={{ height: "60vh" }}>
        <Spin size="large" />
      </div>
    );
  }

  if (!derived) {
    return (
      <div style={{ maxWidth: 1400 }}>
        <Title level={4} style={{ margin: "0 0 16px 0", fontWeight: 600 }}>概览</Title>
        <Alert
          type="error"
          showIcon
          message="数据加载失败"
          description="无法连接到后端服务，请检查网络或联系管理员。"
        />
      </div>
    );
  }

  const { total, mcapTotal, algoCounts, algoTotal, successRate, deliveryTotal, recentAssets, failedAssets, isSampled } = derived;

  return (
    <div style={{ maxWidth: 1400 }}>
      {/* Page header */}
      <div style={{ marginBottom: 24 }}>
        <Title level={4} style={{ margin: 0, fontWeight: 600 }}>概览</Title>
        <Text type="secondary" style={{ fontSize: 13 }}>平台运行状态一览</Text>
      </div>

      {/* Sampling degradation notice */}
      {isSampled && (
        <Alert
          type="warning"
          showIcon
          message="统计数据基于最近 100 条资产采样，算法成功率与交付总量仅供参考"
          style={{ marginBottom: 16 }}
          closable
        />
      )}

      {/* KPI Row */}
      <Row gutter={[16, 16]} style={{ marginBottom: 24 }}>
        <Col xs={24} sm={12} lg={6}>
          <KpiCard
            title="资产总量"
            value={total}
            icon={<DatabaseOutlined />}
            onClick={() => navigate("/assets")}
            trendLabel="本周新增"
          />
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <KpiCard
            title="MCAP 文件"
            value={mcapTotal ?? "—"}
            icon={<FileOutlined />}
            onClick={() => navigate("/mcap-files")}
            color="#7C3AED"
          />
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <KpiCard
            title="算法成功率"
            value={successRate}
            suffix="%"
            icon={<CheckCircleOutlined />}
            onClick={() => navigate("/algo")}
            color={Number(successRate) > 90 ? "#16A34A" : Number(successRate) > 70 ? "#D97706" : "#DC2626"}
          />
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <KpiCard
            title="本月交付"
            value={deliveryTotal}
            suffix="次"
            icon={<SendOutlined />}
            onClick={() => navigate("/deliveries")}
            color="#0891B2"
          />
        </Col>
      </Row>

      {/* Dimension tabs */}
      <Segmented
        options={["资产维度", "算法维度", "交付维度"]}
        value={dimension}
        onChange={(v) => setDimension(v as string)}
        style={{ marginBottom: 16 }}
      />

      <Row gutter={[16, 16]}>
        {/* Algo status distribution */}
        <Col xs={24} lg={14}>
          <Card
            title="算法状态分布"
            size="small"
            styles={{ body: { padding: 16 } }}
          >
            <Row gutter={[12, 12]}>
              {Object.entries(algoCounts).map(([status, count]) => (
                <Col span={8} key={status}>
                  <div
                    style={{
                      textAlign: "center",
                      padding: "16px 8px",
                      borderRadius: 8,
                      background: "#F8FAFC",
                      border: "1px solid #F1F5F9",
                    }}
                  >
                    <Tag color={algoStatusColor[status]} style={{ marginBottom: 8 }}>{status}</Tag>
                    <div style={{ fontSize: 24, fontWeight: 700, color: "#1E293B", fontVariantNumeric: "tabular-nums" }}>
                      {count}
                    </div>
                  </div>
                </Col>
              ))}
            </Row>
            {algoTotal > 0 && (
              <div style={{ marginTop: 12, fontSize: 12, color: "#64748B" }}>
                共 {algoTotal} 个算法任务 · 成功率 {successRate}%
                {isSampled && (
                  <span style={{ marginLeft: 8, color: "#94A3B8" }}>
                    (基于最近 100 条资产采样)
                  </span>
                )}
              </div>
            )}
          </Card>
        </Col>

        {/* Failed tasks */}
        <Col xs={24} lg={10}>
          <Card
            title={
              <Space>
                <WarningOutlined style={{ color: "#DC2626" }} />
                <span>需要关注</span>
                <Tag color="error">{failedAssets.length}</Tag>
              </Space>
            }
            size="small"
            extra={
              failedAssets.length > 0 ? (
                <a onClick={() => navigate("/algo")} style={{ fontSize: 12 }}>
                  查看全部 <RightOutlined />
                </a>
              ) : null
            }
            styles={{ body: { padding: failedAssets.length > 0 ? 0 : 16 } }}
          >
            {failedAssets.length > 0 ? (
              <Table
                rowKey="asset_id"
                columns={failedColumns}
                dataSource={failedAssets.slice(0, 5)}
                pagination={false}
                size="small"
                showHeader={false}
              />
            ) : (
              <div style={{ textAlign: "center", padding: "32px 0", color: "#94A3B8" }}>
                <CheckCircleOutlined style={{ fontSize: 28, marginBottom: 8, display: "block" }} />
                暂无失败任务
              </div>
            )}
          </Card>
        </Col>
      </Row>

      {/* Recent assets */}
      <Card
        title="最近更新"
        size="small"
        style={{ marginTop: 16 }}
        extra={
          <a onClick={() => navigate("/assets")} style={{ fontSize: 12 }}>
            查看全部 <RightOutlined />
          </a>
        }
        styles={{ body: { padding: 0 } }}
      >
        <Table
          rowKey="asset_id"
          dataSource={recentAssets.slice(0, 8)}
          size="small"
          pagination={false}
          columns={[
            {
              title: "Asset ID",
              dataIndex: "asset_id",
              width: 120,
              render: (id: string) => (
                <a className="font-mono text-xs cursor-pointer" onClick={() => navigate(`/assets/${id}`)}>
                  {id.slice(0, 8)}…
                </a>
              ),
            },
            {
              title: "状态",
              dataIndex: "status",
              width: 90,
              render: (s: string) => <Tag>{s}</Tag>,
            },
            {
              title: "时长",
              key: "duration",
              width: 70,
              render: (_: unknown, r: Asset) =>
                formatDurationSeconds(r),
            },
            {
              title: "环境",
              key: "env",
              width: 80,
              render: (_: unknown, r: Asset) => r.env ?? "—",
            },
            {
              title: "算法",
              key: "algo",
              render: (_: unknown, r: Asset) => {
                const statuses = extractAlgoStatuses(r.algo_results);
                if (statuses.length === 0) return <Text type="secondary">—</Text>;
                return (
                  <Space size={2}>
                    {statuses.slice(0, 3).map((s) => (
                      <Tag key={s.key} color={algoStatusColor[s.status]} style={{ fontSize: 10, margin: 0 }}>
                        {s.key.split("@")[0]}
                      </Tag>
                    ))}
                  </Space>
                );
              },
            },
            {
              title: "更新时间",
              dataIndex: "updated_at",
              width: 130,
              render: (v: string) => (
                <Text type="secondary" style={{ fontSize: 12 }}>
                  {dayjs(v).format("MM-DD HH:mm")}
                </Text>
              ),
            },
          ]}
        />
      </Card>
    </div>
  );
}
