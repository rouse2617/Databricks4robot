import { KeyOutlined } from "@ant-design/icons";
import { Typography } from "antd";

import ApiKeysManager from "../components/ApiKeysManager";

export default function ApiKeysPage() {
	return (
		<div>
			<Typography.Title level={4} style={{ margin: 0, marginBottom: 16 }}>
				<KeyOutlined style={{ marginRight: 8 }} />
				API 密钥
			</Typography.Title>
			<ApiKeysManager />
		</div>
	);
}
