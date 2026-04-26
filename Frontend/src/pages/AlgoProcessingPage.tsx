import { Typography, Card, Empty } from "antd";
import { RobotOutlined } from "@ant-design/icons";

const { Title, Text } = Typography;

export default function AlgoProcessingPage() {
  return (
    <div>
      <Title level={4} style={{ margin: 0, marginBottom: 16 }}>
        <RobotOutlined style={{ marginRight: 8 }} />
        算法处理
      </Title>
      <Card>
        <Empty description={false}>
          <Text type="secondary">
            算法处理矩阵视图 — Phase 2 实现
          </Text>
          <br />
          <Text type="secondary" className="text-xs">
            行 = Asset，列 = 算法（sam2 / tracker / deblur / action），色块 = 状态
          </Text>
        </Empty>
      </Card>
    </div>
  );
}
