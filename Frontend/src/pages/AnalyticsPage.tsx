import { useEffect, useMemo, useState } from "react";
import { Alert, Badge, Button, Card, Col, Collapse, Descriptions, Row, Spin, Statistic, Table, Tag, Typography } from "antd";
import { BarChartOutlined, CheckCircleOutlined, CloseCircleOutlined, ReloadOutlined, SyncOutlined, WarningOutlined } from "@ant-design/icons";
import ReactECharts from "echarts-for-react";
import type { ColumnsType } from "antd/es/table";
import {
  lakehouseApi,
  type LakehouseItemsResponse,
  type LakehouseStatus,
  type LakehouseTableCount,
  type SyncStatusResponse,
} from "../api/lakehouse";
import { extractApiErrorMessage } from "../lib/apiError";

const { Title, Text } = Typography;

const questionMeta: Record<string, { title: string; answer: string }> = {
  trainingAssets: {
    title: "某次训练当时用了哪些 asset？",
    answer: "通过 Trino 查询 Iceberg Gold 表 gold_dataset_snapshot_items，返回冻结训练候选 manifest。",
  },
  recomputeCandidates: {
    title: "某个算法版本变更后，哪些历史 asset 要重算？",
    answer: "通过 Trino 查询 Silver 当前资产和算法最新状态，筛出已有旧算法结果的重算候选。",
  },
  tagTimeline: {
    title: "某个 tag 是什么时候被算法追加的？",
    answer: "当前 schema 只能展示 tag current-state 的 updated_at；精确追加时间后续需要 asset_mutation_outbox 入湖。",
  },
  qualityDistribution: {
    title: "上个月/近 30 天所有 MCAP segment 的质量分布是什么？",
    answer: "通过 Trino 聚合 Iceberg Silver asset tags，当前 MVP 使用近 30 天窗口。",
  },
  customerReplay: {
    title: "某个客户交付过的数据是否能完整回放？",
    answer: "通过 Trino join deliveries、delivery_items 和 assets，重建客户交付 manifest。",
  },
};

function asRows(value: unknown): Record<string, unknown>[] {
  return Array.isArray(value) ? (value as Record<string, unknown>[]) : [];
}

function DynamicTable({ rows }: { rows: Record<string, unknown>[] }) {
  const columns = useMemo<ColumnsType<Record<string, unknown>>>(() => {
    const keys = Array.from(new Set(rows.flatMap((row) => Object.keys(row)))).slice(0, 8);
    return keys.map((key) => ({
      title: key,
      dataIndex: key,
      key,
      ellipsis: true,
      render: (value: unknown) => {
        if (value === null || value === undefined || value === "") return <Text type="secondary">NULL</Text>;
        return <span className="font-mono text-xs">{String(value)}</span>;
      },
    }));
  }, [rows]);

  if (rows.length === 0) return <Text type="secondary">暂无数据</Text>;

  return (
    <Table
      size="small"
      rowKey={(_, index) => String(index)}
      columns={columns}
      dataSource={rows}
      pagination={false}
      scroll={{ x: true }}
    />
  );
}

function QuestionPanel({
  answer,
  response,
  errorMessage,
}: {
  answer: string;
  response?: LakehouseItemsResponse;
  errorMessage?: string;
}) {
  return (
    <div>
      <Alert type="info" showIcon message={answer} style={{ marginBottom: 12 }} />
      {errorMessage ? (
        <Alert
          type="warning"
          showIcon
          message="该查询暂不可用"
          description={errorMessage}
        />
      ) : (
        <>
          {response && (
            <div style={{ marginBottom: 12 }}>
              {Object.entries(response)
                .filter(([key]) => key !== "items")
                .map(([key, value]) => (
                  <Tag key={key}>
                    {key}: {String(value)}
                  </Tag>
                ))}
            </div>
          )}
          <DynamicTable rows={asRows(response?.items)} />
        </>
      )}
    </div>
  );
}

/* ─── Quality Distribution Chart (Task 8.1) ─── */
function QualityDistributionChart({
  data,
  errorMessage,
}: {
  data?: LakehouseItemsResponse;
  errorMessage?: string;
}) {
  if (errorMessage) {
    return (
      <Card title="质量分布" size="small">
        <Alert type="warning" showIcon message="质量分布数据暂不可用" description={errorMessage} />
      </Card>
    );
  }

  const rows = asRows(data?.items);
  if (rows.length === 0) {
    return (
      <Card title="质量分布" size="small">
        <Text type="secondary">暂无质量分布数据</Text>
      </Card>
    );
  }

  const qualityLabels = rows.map((r) => String(r.quality ?? r.tag_value ?? r.key ?? "unknown"));
  const counts = rows.map((r) => Number(r.count ?? r.asset_count ?? r.doc_count ?? 0));

  const colorMap: Record<string, string> = {
    excellent: "#52c41a",
    good: "#73d13d",
    acceptable: "#faad14",
    poor: "#ff7a45",
    unusable: "#ff4d4f",
  };
  const colors = qualityLabels.map((label) => colorMap[label.toLowerCase()] ?? "#1890ff");

  const option = {
    tooltip: { trigger: "item" as const },
    legend: { bottom: 0 },
    series: [
      {
        type: "pie",
        radius: ["40%", "70%"],
        avoidLabelOverlap: false,
        itemStyle: { borderRadius: 6, borderColor: "#fff", borderWidth: 2 },
        label: { show: true, formatter: "{b}: {c} ({d}%)" },
        data: qualityLabels.map((label, i) => ({
          name: label,
          value: counts[i],
          itemStyle: { color: colors[i] },
        })),
      },
    ],
  };

  return (
    <Card title="质量分布" size="small" data-testid="quality-distribution-chart">
      <ReactECharts option={option} style={{ height: 280 }} />
    </Card>
  );
}

/* ─── Table Row Count Bar Chart (Task 8.1) ─── */
function TableRowCountChart({ tables }: { tables: LakehouseTableCount[] }) {
  if (tables.length === 0) {
    return (
      <Card title="Iceberg 表行数" size="small">
        <Text type="secondary">暂无表数据</Text>
      </Card>
    );
  }

  const names = tables.map((t) => t.table_name);
  const counts = tables.map((t) => t.row_count);

  const option = {
    tooltip: { trigger: "axis" as const },
    grid: { left: 180, right: 30, top: 10, bottom: 30 },
    xAxis: { type: "value" as const },
    yAxis: {
      type: "category" as const,
      data: names,
      axisLabel: { fontSize: 11 },
    },
    series: [
      {
        type: "bar",
        data: counts,
        itemStyle: { color: "#1890ff", borderRadius: [0, 4, 4, 0] },
        label: { show: true, position: "right" as const, formatter: "{c}" },
      },
    ],
  };

  return (
    <Card title="Iceberg 表行数" size="small" data-testid="table-row-count-chart">
      <ReactECharts option={option} style={{ height: Math.max(200, tables.length * 40) }} />
    </Card>
  );
}

/* ─── Sync Status Traffic Light Card (Task 8.2) ─── */
function SyncStatusCard({ syncStatus }: { syncStatus: SyncStatusResponse | null }) {
  if (!syncStatus || !syncStatus.available || !syncStatus.data) {
    return (
      <Card title="同步对账状态" size="small">
        <Text type="secondary">对账数据暂不可用，请先运行 Dagster pipeline 或等待实时 Bronze 对账</Text>
      </Card>
    );
  }

  const d = syncStatus.data;
  const diffPct = d.count_diff_pct * 100;
  const sourceLabel = syncStatus.source === "realtime" ? "实时检查" : "Dagster 对账";

  // Traffic light logic: green (<1%), yellow (1-5%), red (>5% or is_alert)
  let lightColor: string;
  let lightLabel: string;
  let lightIcon: React.ReactNode;
  if (d.is_alert || diffPct > 5) {
    lightColor = "#ff4d4f";
    lightLabel = "差异告警";
    lightIcon = <CloseCircleOutlined />;
  } else if (diffPct > 1) {
    lightColor = "#faad14";
    lightLabel = "轻微差异";
    lightIcon = <WarningOutlined />;
  } else {
    lightColor = "#52c41a";
    lightLabel = "数据一致";
    lightIcon = <CheckCircleOutlined />;
  }

  const checkedAt = new Date(d.checked_at).toLocaleString("zh-CN");
  const diffPctStr = diffPct.toFixed(3) + "%";

  return (
    <Card
      title={
        <span>
          <SyncOutlined style={{ marginRight: 8 }} />
          同步对账状态
        </span>
      }
      size="small"
      extra={
        <Badge
          count={lightLabel}
          style={{ backgroundColor: lightColor }}
        />
      }
      data-testid="sync-status-card"
    >
      {/* Traffic light indicator */}
      <div
        style={{
          display: "flex",
          alignItems: "center",
          gap: 12,
          marginBottom: 16,
          padding: "8px 12px",
          borderRadius: 8,
          backgroundColor: `${lightColor}10`,
          border: `1px solid ${lightColor}40`,
        }}
        data-testid="sync-traffic-light"
      >
        <div
          style={{
            width: 32,
            height: 32,
            borderRadius: "50%",
            backgroundColor: lightColor,
            display: "flex",
            alignItems: "center",
            justifyContent: "center",
            color: "#fff",
            fontSize: 18,
          }}
          aria-label={`同步状态: ${lightLabel}`}
        >
          {lightIcon}
        </div>
        <div>
          <div style={{ fontWeight: 600, color: lightColor }}>{lightLabel}</div>
          <div style={{ fontSize: 12, color: "#666" }}>差异 {diffPctStr}</div>
        </div>
      </div>

      <Descriptions column={2} size="small">
        <Descriptions.Item label="最近同步时间">{checkedAt}</Descriptions.Item>
        <Descriptions.Item label="数据来源">
          <span data-testid="sync-source">{sourceLabel}</span>
        </Descriptions.Item>
        <Descriptions.Item label="Dagster Run ID">
          {d.dagster_run_id ? (
            <Text copyable className="font-mono text-xs">{d.dagster_run_id}</Text>
          ) : (
            <Text type="secondary">—</Text>
          )}
        </Descriptions.Item>
        <Descriptions.Item label="Postgres 行数">
          <span data-testid="pg-count">{d.pg_total_count.toLocaleString()}</span>
        </Descriptions.Item>
        <Descriptions.Item label="Iceberg 行数">
          <span data-testid="iceberg-count">{d.iceberg_total_count.toLocaleString()}</span>
        </Descriptions.Item>
        {typeof d.iceberg_max_seq === "number" && (
          <Descriptions.Item label="Iceberg 最大序号">
            {d.iceberg_max_seq.toLocaleString()}
          </Descriptions.Item>
        )}
        <Descriptions.Item label="差异百分比">
          <span data-testid="diff-pct">{diffPctStr}</span>
        </Descriptions.Item>
        <Descriptions.Item label="对账状态">
          <span style={{ color: lightColor }}>
            {lightIcon} {lightLabel}
          </span>
        </Descriptions.Item>
      </Descriptions>
    </Card>
  );
}

/* ─── Postgres vs Iceberg Comparison Table ─── */
interface ComparisonRow {
  status: string;
  pg: number;
  iceberg: number;
  diff: number;
}

function PgIcebergComparisonTable({ syncStatus }: { syncStatus: SyncStatusResponse | null }) {
  if (!syncStatus?.available || !syncStatus.data) {
    return null;
  }

  const statusDiff = syncStatus.data.status_diff;
  const dataSource: ComparisonRow[] = Object.entries(statusDiff).map(([status, counts]) => ({
    status,
    pg: counts.pg,
    iceberg: counts.iceberg,
    diff: counts.diff,
  }));

  // Add total row
  dataSource.push({
    status: "合计",
    pg: syncStatus.data.pg_total_count,
    iceberg: syncStatus.data.iceberg_total_count,
    diff: syncStatus.data.pg_total_count - syncStatus.data.iceberg_total_count,
  });

  const columns: ColumnsType<ComparisonRow> = [
    {
      title: "状态",
      dataIndex: "status",
      render: (val: string) =>
        val === "合计" ? <Text strong>{val}</Text> : <Tag>{val}</Tag>,
    },
    {
      title: "Postgres 行数",
      dataIndex: "pg",
      align: "right",
      render: (val: number) => val.toLocaleString(),
    },
    {
      title: "Iceberg 行数",
      dataIndex: "iceberg",
      align: "right",
      render: (val: number) => val.toLocaleString(),
    },
    {
      title: "差异",
      dataIndex: "diff",
      align: "right",
      render: (val: number) => {
        if (val === 0) return <Text type="success">0</Text>;
        return <Text type={val > 0 ? "warning" : "danger"}>{val > 0 ? `+${val}` : val}</Text>;
      },
    },
  ];

  return (
    <Card title="Postgres vs Iceberg 行数对比" size="small">
      <Table<ComparisonRow>
        size="small"
        rowKey="status"
        columns={columns}
        dataSource={dataSource}
        pagination={false}
      />
    </Card>
  );
}

/* ─── Main Page ─── */
const QUERY_KEYS = [
  "trainingAssets",
  "recomputeCandidates",
  "tagTimeline",
  "qualityDistribution",
  "customerReplay",
] as const;

export default function AnalyticsPage() {
  const [loading, setLoading] = useState(true);
  const [status, setStatus] = useState<LakehouseStatus | null>(null);
  const [tables, setTables] = useState<LakehouseTableCount[]>([]);
  const [syncStatus, setSyncStatus] = useState<SyncStatusResponse | null>(null);
  const [queryResults, setQueryResults] = useState<Record<string, LakehouseItemsResponse>>({});
  const [queryErrors, setQueryErrors] = useState<Record<string, string>>({});
  const [error, setError] = useState<string | null>(null);

  const loadLakehouse = () => {
    setLoading(true);
    setError(null);
    setQueryErrors({});

    // Use allSettled so one slow/failing endpoint doesn't blank out the whole
    // page. Required endpoints (status + tables) drive the top-level error;
    // optional query endpoints are degraded into per-panel error messages.
    Promise.allSettled([
      lakehouseApi.status(),
      lakehouseApi.tables(),
      lakehouseApi.syncStatus(),
      lakehouseApi.trainingAssets(),
      lakehouseApi.recomputeCandidates(),
      lakehouseApi.tagTimeline(),
      lakehouseApi.qualityDistribution(),
      lakehouseApi.customerReplay(),
    ])
      .then(([statusRes, tablesRes, syncRes, ...queryRes]) => {
        if (statusRes.status === "fulfilled") {
          setStatus(statusRes.value);
        } else {
          setStatus(null);
        }

        if (tablesRes.status === "fulfilled") {
          setTables(tablesRes.value.items ?? []);
        } else {
          setTables([]);
        }

        // syncStatus has its own "available: false" fallback shape.
        if (syncRes.status === "fulfilled") {
          setSyncStatus(syncRes.value);
        } else {
          setSyncStatus({ available: false } as SyncStatusResponse);
        }

        // Top-level error only when both critical endpoints fail.
        if (statusRes.status === "rejected" && tablesRes.status === "rejected") {
          setError(extractApiErrorMessage(statusRes.reason, "加载 Trino 湖仓查询失败"));
        }

        const successResults: Record<string, LakehouseItemsResponse> = {};
        const errors: Record<string, string> = {};
        QUERY_KEYS.forEach((key, idx) => {
          const r = queryRes[idx];
          if (r.status === "fulfilled") {
            successResults[key] = r.value;
          } else {
            errors[key] = extractApiErrorMessage(r.reason, "查询失败");
          }
        });
        setQueryResults(successResults);
        setQueryErrors(errors);
      })
      .finally(() => setLoading(false));
  };

  useEffect(() => {
    loadLakehouse();
  }, []);

  const goldCount = tables.find((t) => t.table_name === "gold_dataset_snapshot_items")?.row_count ?? 0;
  const silverAssets = tables.find((t) => t.table_name === "silver_assets_current")?.row_count ?? 0;
  const bronzeEvents = tables.find((t) => t.table_name === "bronze_asset_algo_events")?.row_count ?? 0;

  if (loading) {
    return (
      <div className="flex items-center justify-center" style={{ height: "60vh" }}>
        <Spin size="large" />
      </div>
    );
  }

  return (
    <div>
      <div style={{ display: "flex", justifyContent: "space-between", alignItems: "flex-start", marginBottom: 16 }}>
        <div>
          <Title level={4} style={{ margin: 0 }}>
            <BarChartOutlined style={{ marginRight: 8 }} />
            湖仓验证
          </Title>
          <Text type="secondary">
            Postgres → Iceberg Bronze/Silver/Gold MVP 查询结果
          </Text>
        </div>
        <Button icon={<ReloadOutlined />} onClick={loadLakehouse}>
          刷新查询
        </Button>
      </div>

      {error && (
        <Alert
          type="warning"
          showIcon
          message="Trino 湖仓查询暂不可用"
          description={`${error}。请确认 make iceberg-up、make iceberg-mvp-host 和 Trino 服务已完成。`}
          style={{ marginBottom: 16 }}
        />
      )}

      {error ? null : (
        <>
          <Alert
            type={status?.healthy ? "success" : "warning"}
            showIcon
            message={status?.healthy ? "Trino 查询层已连接" : "Trino 查询层未就绪"}
            description={
              <span>
                Catalog：<Tag>{status?.catalog ?? "iceberg"}</Tag>
                Schema：<Tag>{status?.schema ?? "robot"}</Tag>
                查询引擎：<Tag>Trino</Tag>
              </span>
            }
            style={{ marginBottom: 16 }}
          />

          {/* Sync Status Traffic Light Card (Task 8.2) */}
          <Row gutter={[16, 16]} style={{ marginBottom: 16 }}>
            <Col span={24}>
              <SyncStatusCard syncStatus={syncStatus} />
            </Col>
          </Row>

          <Row gutter={[16, 16]} style={{ marginBottom: 16 }}>
            <Col xs={24} md={8}>
              <Card>
                <Statistic title="Silver 当前资产" value={silverAssets} />
              </Card>
            </Col>
            <Col xs={24} md={8}>
              <Card>
                <Statistic title="Bronze 算法事件" value={bronzeEvents} />
              </Card>
            </Col>
            <Col xs={24} md={8}>
              <Card>
                <Statistic title="Gold 训练候选" value={goldCount} />
              </Card>
            </Col>
          </Row>

          {/* Visualization Cards (Task 8.1) */}
          <Row gutter={[16, 16]} style={{ marginBottom: 16 }}>
            <Col xs={24} lg={12}>
              <QualityDistributionChart
                data={queryResults.qualityDistribution}
                errorMessage={queryErrors.qualityDistribution}
              />
            </Col>
            <Col xs={24} lg={12}>
              <TableRowCountChart tables={tables} />
            </Col>
          </Row>

          <Row gutter={[16, 16]} style={{ marginBottom: 16 }}>
            {/* Postgres vs Iceberg Comparison */}
            <Col span={24}>
              <PgIcebergComparisonTable syncStatus={syncStatus} />
            </Col>
          </Row>

          <Row gutter={[16, 16]}>
            <Col span={24}>
              <Card title="关键查询验证" size="small">
                <Collapse
                  defaultActiveKey={["trainingAssets"]}
                  items={Object.entries(questionMeta).map(([key, item]) => ({
                    key,
                    label: item.title,
                    children: (
                      <QuestionPanel
                        answer={item.answer}
                        response={queryResults[key]}
                        errorMessage={queryErrors[key]}
                      />
                    ),
                  }))}
                />
              </Card>
            </Col>
          </Row>
        </>
      )}
    </div>
  );
}
