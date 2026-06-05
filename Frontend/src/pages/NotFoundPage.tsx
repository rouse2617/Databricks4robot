import { Button, Result } from "antd";
import { useNavigate } from "react-router-dom";

export default function NotFoundPage() {
	const navigate = useNavigate();

	return (
		<div
			style={{
				display: "flex",
				justifyContent: "center",
				alignItems: "center",
				minHeight: "60vh",
			}}
		>
			<Result
				status="404"
				title="404"
				subTitle="页面未找到"
				extra={
					<Button type="primary" onClick={() => navigate("/dashboard")}>
						返回首页
					</Button>
				}
			/>
		</div>
	);
}
