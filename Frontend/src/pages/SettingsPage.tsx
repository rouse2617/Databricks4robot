import { Typography, Card, Descriptions } from "antd";
import { SettingOutlined } from "@ant-design/icons";
import { useAuth } from "../hooks/useAuth";

const { Title } = Typography;

export default function SettingsPage() {
  const { token } = useAuth();

  return (
    <div>
      <Title level={4} style={{ margin: 0, marginBottom: 16 }}>
        <SettingOutlined style={{ marginRight: 8 }} />
        设置
      </Title>
      <Card title="当前会话" size="small">
        <Descriptions column={1} size="small">
          <Descriptions.Item label="认证方式">X-Grace-Token (Phase 0)</Descriptions.Item>
          <Descriptions.Item label="Token">
            <code className="text-xs">{token ? `${token.slice(0, 8)}...` : "—"}</code>
          </Descriptions.Item>
          <Descriptions.Item label="API 地址">/api/v1</Descriptions.Item>
        </Descriptions>
      </Card>
    </div>
  );
}
