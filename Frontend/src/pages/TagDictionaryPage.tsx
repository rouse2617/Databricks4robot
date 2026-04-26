import { Typography, Card, Empty } from "antd";
import { TagsOutlined } from "@ant-design/icons";

const { Title, Text } = Typography;

export default function TagDictionaryPage() {
  return (
    <div>
      <Title level={4} style={{ margin: 0, marginBottom: 16 }}>
        <TagsOutlined style={{ marginRight: 8 }} />
        标签字典
      </Title>
      <Card>
        <Empty description={false}>
          <Text type="secondary">
            标签字典管理 — Phase 2 实现
          </Text>
          <br />
          <Text type="secondary" className="text-xs">
            对接 tag_registry.yaml，管理允许的 tag key 和 value
          </Text>
        </Empty>
      </Card>
    </div>
  );
}
