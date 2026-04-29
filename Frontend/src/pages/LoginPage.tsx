import { Form, Input, Button, Card, message } from "antd";
import { useNavigate } from "react-router-dom";
import { useAuth } from "../hooks/useAuth";

export default function LoginPage() {
  const { login } = useAuth();
  const navigate = useNavigate();
  const [msg, msgCtx] = message.useMessage();

  const onFinish = ({ token }: { token: string }) => {
    const trimmed = token.trim();
    if (!trimmed) {
      msg.error("Token is required");
      return;
    }
    login(trimmed);
    navigate("/dashboard");
  };

  return (
    <div className="min-h-screen flex items-center justify-center bg-gray-100">
      {msgCtx}
      <Card title="data4cyber Platform" className="w-96 shadow-lg">
        <Form layout="vertical" onFinish={onFinish}>
          <Form.Item
            name="token"
            label="Access Token"
            rules={[{ required: true, message: "Please enter your token" }]}
          >
            <Input.Password placeholder="Enter your GRACE_TOKEN" size="large" />
          </Form.Item>
          <Button type="primary" htmlType="submit" block size="large">
            Sign In
          </Button>
        </Form>
      </Card>
    </div>
  );
}
