import { Layout, Menu, Avatar, Dropdown, Typography } from "antd";
import {
  DashboardOutlined,
  DatabaseOutlined,
  FileOutlined,
  RobotOutlined,
  SendOutlined,
  BarChartOutlined,
  TagsOutlined,
  SettingOutlined,
  LogoutOutlined,
  UserOutlined,
} from "@ant-design/icons";
import { useNavigate, useLocation } from "react-router-dom";
import { useAuth } from "../hooks/useAuth";
import type { ReactNode } from "react";

const { Sider, Content } = Layout;

const menuItems = [
  { key: "/dashboard", icon: <DashboardOutlined />, label: "概览" },
  { key: "/assets", icon: <DatabaseOutlined />, label: "资产管理" },
  { key: "/mcap-files", icon: <FileOutlined />, label: "MCAP 文件" },
  { key: "/algo", icon: <RobotOutlined />, label: "算法处理" },
  { key: "/deliveries", icon: <SendOutlined />, label: "交付管理" },
  { type: "divider" as const },
  { key: "/analytics", icon: <BarChartOutlined />, label: "湖仓验证" },
  { key: "/tags", icon: <TagsOutlined />, label: "标签字典" },
  { key: "/settings", icon: <SettingOutlined />, label: "设置" },
];

function resolveSelectedKey(pathname: string): string {
  if (pathname.startsWith("/assets")) return "/assets";
  const exact = menuItems.find((m) => "key" in m && m.key === pathname);
  if (exact && "key" in exact) return exact.key as string;
  const prefix = menuItems.find((m) => "key" in m && pathname.startsWith(m.key as string));
  return prefix && "key" in prefix ? (prefix.key as string) : "/dashboard";
}

export default function AppLayout({ children }: { children: ReactNode }) {
  const navigate = useNavigate();
  const location = useLocation();
  const { logout } = useAuth();

  return (
    <Layout style={{ minHeight: "100vh" }}>
      <Sider
        width={220}
        style={{
          overflow: "auto",
          height: "100vh",
          position: "fixed",
          left: 0,
          top: 0,
          bottom: 0,
          background: "#0F172A",
          display: "flex",
          flexDirection: "column",
        }}
      >
        {/* Logo */}
        <div
          onClick={() => navigate("/dashboard")}
          style={{
            height: 56,
            display: "flex",
            alignItems: "center",
            padding: "0 20px",
            cursor: "pointer",
            borderBottom: "1px solid rgba(255,255,255,0.06)",
          }}
        >
          <div
            style={{
              width: 28,
              height: 28,
              borderRadius: 6,
              background: "linear-gradient(135deg, #2563EB, #3B82F6)",
              display: "flex",
              alignItems: "center",
              justifyContent: "center",
              marginRight: 10,
              flexShrink: 0,
            }}
          >
            <DatabaseOutlined style={{ color: "#fff", fontSize: 14 }} />
          </div>
          <Typography.Text
            strong
            style={{ color: "#F1F5F9", fontSize: 15, letterSpacing: 0.5 }}
          >
            data4cyber
          </Typography.Text>
        </div>

        {/* Navigation */}
        <div style={{ flex: 1, paddingTop: 8 }}>
          <Menu
            mode="inline"
            theme="dark"
            selectedKeys={[resolveSelectedKey(location.pathname)]}
            items={menuItems}
            onClick={({ key }) => navigate(key)}
            style={{ background: "transparent", borderRight: 0 }}
          />
        </div>

        {/* User section */}
        <div
          style={{
            padding: "12px 16px",
            borderTop: "1px solid rgba(255,255,255,0.06)",
          }}
        >
          <Dropdown
            menu={{
              items: [
                { key: "logout", icon: <LogoutOutlined />, label: "退出登录", danger: true },
              ],
              onClick: () => logout(),
            }}
            placement="topRight"
            trigger={["click"]}
          >
            <div
              style={{
                display: "flex",
                alignItems: "center",
                gap: 10,
                padding: "8px 8px",
                borderRadius: 6,
                cursor: "pointer",
                transition: "background 0.2s",
              }}
              className="hover:bg-white/5"
            >
              <Avatar
                size={32}
                icon={<UserOutlined />}
                style={{ background: "#334155", flexShrink: 0 }}
              />
              <div style={{ overflow: "hidden" }}>
                <div style={{ color: "#E2E8F0", fontSize: 13, fontWeight: 500, lineHeight: 1.3 }}>
                  管理员
                </div>
                <div style={{ color: "#64748B", fontSize: 11, lineHeight: 1.3 }}>
                  Phase 0
                </div>
              </div>
            </div>
          </Dropdown>
        </div>
      </Sider>

      <Layout style={{ marginLeft: 220, background: "#F8FAFC" }}>
        <Content
          style={{
            padding: 24,
            minHeight: "100vh",
          }}
        >
          {children}
        </Content>
      </Layout>
    </Layout>
  );
}
