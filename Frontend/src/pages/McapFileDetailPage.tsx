import { Button, Typography } from "antd";
import { useNavigate } from "react-router-dom";

const { Title } = Typography;

export default function McapFileDetailPage() {
	const navigate = useNavigate();

	return (
		<div style={{ padding: 24 }}>
			<Button onClick={() => navigate(-1)} style={{ marginBottom: 16 }}>
				返回
			</Button>
			<Title level={3}>MCAP 文件详情</Title>
		</div>
	);
}
