import { useEffect, useState, useMemo } from "react";
import {
  Typography,
  Card,
  Table,
  Tag,
  Space,
  Alert,
  Input,
} from "antd";
import { TagsOutlined, SearchOutlined } from "@ant-design/icons";
import type { ColumnsType } from "antd/es/table";
import { tagRegistryApi, type TagRegistryItem } from "../api/tagRegistry";

const { Title, Text } = Typography;

export default function TagDictionaryPage() {
  const [items, setItems] = useState<TagRegistryItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [q, setQ] = useState("");

  useEffect(() => {
    let cancelled = false;
    (async () => {
      setLoading(true);
      setError(null);
      try {
        const list = await tagRegistryApi.list();
        if (!cancelled) setItems(list);
      } catch {
        if (!cancelled) setError("加载失败，请检查网络、后端与登录态");
      } finally {
        if (!cancelled) setLoading(false);
      }
    })();
    return () => {
      cancelled = true;
    };
  }, []);

  const filtered = useMemo(() => {
    const s = q.trim().toLowerCase();
    if (!s) return items;
    return items.filter(
      (it) =>
        it.key.toLowerCase().includes(s) ||
        (it.description ?? "").toLowerCase().includes(s) ||
        it.type.toLowerCase().includes(s) ||
        (it.values ?? []).some((v) => v.toLowerCase().includes(s)),
    );
  }, [items, q]);

  const columns: ColumnsType<TagRegistryItem> = [
    {
      title: "Key",
      dataIndex: "key",
      width: 140,
      fixed: "left",
      render: (k: string) => <Text code>{k}</Text>,
    },
    {
      title: "说明",
      dataIndex: "description",
      width: 220,
      ellipsis: true,
      render: (d?: string) => d || "—",
    },
    {
      title: "类型",
      dataIndex: "type",
      width: 100,
      render: (t: string) => <Tag>{t}</Tag>,
    },
    {
      title: "允许取值 / 约束",
      key: "constraint",
      render: (_: unknown, r: TagRegistryItem) => {
        if (r.type === "enum" && r.values?.length) {
          return (
            <Space size={[4, 4]} wrap>
              {r.values.map((v) => (
                <Tag key={v} color="blue">
                  {v}
                </Tag>
              ))}
            </Space>
          );
        }
        if (r.type === "string" && r.max_length) {
          return (
            <Text type="secondary">
              自由文本，最长 {r.max_length} 字符
            </Text>
          );
        }
        if (r.type === "string") {
          return <Text type="secondary">自由文本</Text>;
        }
        return "—";
      },
    },
  ];

  return (
    <div>
      <Title level={4} style={{ margin: 0, marginBottom: 16 }}>
        <TagsOutlined style={{ marginRight: 8 }} />
        标签字典
      </Title>

      <Alert
        type="info"
        showIcon
        style={{ marginBottom: 16 }}
        message="只读注册表"
        description="数据来自 tag_registry.yaml（与校验逻辑一致），支持服务端热重载。变更定义请改仓库内 YAML 并部署；本页用于查阅允许的 key 与枚举值。"
      />

      <Card size="small">
        <Input
          allowClear
          placeholder="按 key、说明、类型或枚举值过滤"
          prefix={<SearchOutlined />}
          value={q}
          onChange={(e) => setQ(e.target.value)}
          style={{ maxWidth: 400, marginBottom: 12 }}
        />
        {error ? (
          <Alert type="error" message={error} showIcon />
        ) : (
          <Table<TagRegistryItem>
            rowKey="key"
            columns={columns}
            dataSource={filtered}
            loading={loading}
            pagination={false}
            scroll={{ x: 720 }}
            size="middle"
          />
        )}
      </Card>
    </div>
  );
}
