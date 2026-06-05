import { Button, Card, Form, Input, message } from "antd";
import { useNavigate } from "react-router-dom";
import { useAuth } from "../hooks/useAuth";

export default function LoginPage() {
	const { login } = useAuth();
	const navigate = useNavigate();
	const [msg, msgCtx] = message.useMessage();

	const onFinish = async ({ email }: { email: string }) => {
		const trimmed = email.trim().toLowerCase();
		if (!trimmed?.includes("@")) {
			msg.error("请输入有效的公司邮箱");
			return;
		}
		try {
			await login(trimmed);
			navigate("/dashboard");
		} catch {
			msg.error("登录失败，该邮箱不被允许");
		}
	};

	return (
		<div className="min-h-screen flex items-center justify-center bg-gray-100">
			{msgCtx}
			<Card title="Cyber Databrew Platform" className="w-96 shadow-lg">
				<Form layout="vertical" onFinish={onFinish}>
					<Form.Item
						name="email"
						label="公司邮箱"
						rules={[
							{ required: true, message: "请输入你的公司邮箱" },
							{ type: "email", message: "请输入有效的邮箱地址" },
						]}
					>
						<Input placeholder="name@cyberorigin.ai" size="large" autoComplete="email" />
					</Form.Item>
					<Button type="primary" htmlType="submit" block size="large">
						登录
					</Button>
				</Form>
			</Card>
		</div>
	);
}
