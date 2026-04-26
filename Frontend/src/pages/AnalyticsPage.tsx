import { Typography, Card, Empty } from "antd";
import { BarChartOutlined } from "@ant-design/icons";

const { Title, Text } = Typography;

export default function AnalyticsPage() {
  return (
    <div>
      <Title level={4} style={{ margin: 0, marginBottom: 16 }}>
        <BarChartOutlined style={{ marginRight: 8 }} />
        数据分析
      </Title>
      <Card>
        <Empty description={false}>
          <Text type="secondary">
            数据分析图表 — Phase 2 实现
          </Text>
          <br />
          <Text type="secondary" className="text-xs">
            算法处理耗时分布 · 每日采集量趋势 · 交付量日历热力图 · 算法成功率饼图
          </Text>
        </Empty>
      </Card>
    </div>
  );
}
