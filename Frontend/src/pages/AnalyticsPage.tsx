import { useEffect, useMemo, useState } from "react";
import { Alert, Button, Card, Col, Collapse, Row, Spin, Statistic, Table, Tag, Typography } from "antd";
import { BarChartOutlined, ReloadOutlined } from "@ant-design/icons";
import type { ColumnsType } from "antd/es/table";
import { lakehouseApi, type LakehouseItemsResponse, type LakehouseStatus, type LakehouseTableCount } from "../api/lakehouse";

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

function QuestionPanel({ answer, response }: { answer: string; response?: LakehouseItemsResponse }) {
  return (
    <div>
      <Alert type="info" showIcon message={answer} style={{ marginBottom: 12 }} />
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
    </div>
  );
}

export default function AnalyticsPage() {
  const [loading, setLoading] = useState(true);
  const [status, setStatus] = useState<LakehouseStatus | null>(null);
  const [tables, setTables] = useState<LakehouseTableCount[]>([]);
  const [queryResults, setQueryResults] = useState<Record<string, LakehouseItemsResponse>>({});
  const [error, setError] = useState<string | null>(null);

  const loadLakehouse = () => {
    setLoading(true);
    setError(null);
    Promise.all([
      lakehouseApi.status(),
      lakehouseApi.tables(),
      lakehouseApi.trainingAssets(),
      lakehouseApi.recomputeCandidates(),
      lakehouseApi.tagTimeline(),
      lakehouseApi.qualityDistribution(),
      lakehouseApi.customerReplay(),
    ])
      .then(([statusData, tableData, trainingAssets, recomputeCandidates, tagTimeline, qualityDistribution, customerReplay]) => {
        setStatus(statusData);
        setTables(tableData.items ?? []);
        setQueryResults({
          trainingAssets,
          recomputeCandidates,
          tagTimeline,
          qualityDistribution,
          customerReplay,
        });
      })
      .catch((err) => {
        setError(err.response?.data?.message ?? err.message ?? "加载 Trino 湖仓查询失败");
      })
      .finally(() => setLoading(false));
  };

  useEffect(() => {
    loadLakehouse();
  }, []);

  const tableColumns: ColumnsType<LakehouseTableCount> = [
    {
      title: "Iceberg 表",
      dataIndex: "table_name",
      render: (value: string) => <span className="font-mono text-xs">{value}</span>,
    },
    {
      title: "行数",
      dataIndex: "row_count",
      align: "right",
      render: (value: number) => value.toLocaleString(),
    },
  ];

  const goldCount = tables.find((table) => table.table_name === "gold_dataset_snapshot_items")?.row_count ?? 0;
  const silverAssets = tables.find((table) => table.table_name === "silver_assets_current")?.row_count ?? 0;
  const bronzeEvents = tables.find((table) => table.table_name === "bronze_asset_algo_events")?.row_count ?? 0;

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

          <Row gutter={[16, 16]}>
            <Col xs={24} lg={9}>
              <Card title="Iceberg 表行数" size="small">
                <Table
                  size="small"
                  rowKey="table_name"
                  columns={tableColumns}
                  dataSource={tables}
                  pagination={false}
                />
              </Card>
            </Col>
            <Col xs={24} lg={15}>
              <Card title="关键查询验证" size="small">
                <Collapse
                  defaultActiveKey={["trainingAssets"]}
                  items={Object.entries(questionMeta).map(([key, item]) => ({
                    key,
                    label: item.title,
                    children: <QuestionPanel answer={item.answer} response={queryResults[key]} />,
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
