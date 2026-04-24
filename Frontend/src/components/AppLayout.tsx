import { Layout, Menu } from "antd";
import {
  DatabaseOutlined,
  FileOutlined,
  SendOutlined,
  LogoutOutlined,
} from "@ant-design/icons";
import { useNavigate, useLocation } from "react-router-dom";
import { useAuth } from "../hooks/useAuth";
import type { ReactNode } from "react";

const { Header, Sider, Content } = Layout;

const menuItems = [
  { key: "/assets", icon: <DatabaseOutlined />, label: "Assets" },
  { key: "/mcap-files", icon: <FileOutlined />, label: "MCAP Files" },
  { key: "/deliveries", icon: <SendOutlined />, label: "Deliveries" },
];

export default function AppLayout({ children }: { children: ReactNode }) {
  const navigate = useNavigate();
  const location = useLocation();
  const { logout } = useAuth();

  return (
    <Layout style={{ minHeight: "100vh" }}>
      <Sider theme="dark" width={220}>
        <div className="h-16 flex items-center justify-center text-white font-bold text-lg border-b border-gray-700">
          data4cyber
        </div>
        <Menu
          theme="dark"
          mode="inline"
          selectedKeys={[location.pathname]}
          items={menuItems}
          onClick={({ key }) => navigate(key)}
        />
        <div className="absolute bottom-4 w-full px-4">
          <Menu
            theme="dark"
            mode="inline"
            selectable={false}
            items={[{ key: "logout", icon: <LogoutOutlined />, label: "Logout" }]}
            onClick={logout}
          />
        </div>
      </Sider>
      <Layout>
        <Header className="bg-white border-b px-6 flex items-center">
          <span className="text-gray-600 text-sm">data4cyber Platform</span>
        </Header>
        <Content className="p-6 overflow-auto">{children}</Content>
      </Layout>
    </Layout>
  );
}
