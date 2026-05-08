import {
	ApartmentOutlined,
	BarChartOutlined,
	DashboardOutlined,
	DatabaseOutlined,
	FileOutlined,
	FundProjectionScreenOutlined,
	LogoutOutlined,
	RobotOutlined,
	SendOutlined,
	SettingOutlined,
	UnorderedListOutlined,
	UserOutlined,
} from "@ant-design/icons";
import { Avatar, Dropdown, Grid, Layout, Menu, Typography } from "antd";
import type { ReactNode } from "react";
import { useLocation, useNavigate } from "react-router-dom";
import { useAuth } from "../hooks/useAuth";

const { Sider, Content } = Layout;
const { useBreakpoint } = Grid;

const menuItems = [
	{ key: "/dashboard", icon: <DashboardOutlined />, label: "概览" },
	{ key: "/assets", icon: <DatabaseOutlined />, label: "资产管理" },
	{ key: "/mcap-files", icon: <FileOutlined />, label: "MCAP 文件" },
	{ key: "/algo", icon: <RobotOutlined />, label: "算法处理" },
	{ key: "/deliveries", icon: <SendOutlined />, label: "交付管理" },
	{ key: "/analytics", icon: <BarChartOutlined />, label: "湖仓验证" },
	{ type: "divider" as const },
	{ key: "/events", icon: <UnorderedListOutlined />, label: "事件流" },
	{ key: "/registry", icon: <ApartmentOutlined />, label: "注册中心" },
	{
		key: "/metrics-search",
		icon: <FundProjectionScreenOutlined />,
		label: "指标检索",
	},
	{ key: "/settings", icon: <SettingOutlined />, label: "设置" },
];

function resolveSelectedKey(pathname: string): string {
	if (pathname.startsWith("/assets")) return "/assets";
	if (pathname.startsWith("/analytics")) return "/analytics";
	const exact = menuItems.find((m) => "key" in m && m.key === pathname);
	if (exact && "key" in exact) return exact.key as string;
	const prefix = menuItems.find(
		(m) => "key" in m && pathname.startsWith(m.key as string),
	);
	return prefix && "key" in prefix ? (prefix.key as string) : "/dashboard";
}

export default function AppLayout({ children }: { children: ReactNode }) {
	const navigate = useNavigate();
	const location = useLocation();
	const { logout } = useAuth();
	const screens = useBreakpoint();
	const isMobile = !screens.lg;
	const siderWidth = 220;

	return (
		<Layout style={{ minHeight: "100vh" }}>
			<Sider
				width={siderWidth}
				breakpoint="lg"
				collapsedWidth={isMobile ? 0 : 80}
				collapsible
				style={{
					overflow: "auto",
					height: "100vh",
					position: isMobile ? "sticky" : "fixed",
					left: 0,
					top: 0,
					bottom: 0,
					background: "#0F172A",
					display: "flex",
					flexDirection: "column",
				}}
			>
				{/* Logo */}
				<button
					type="button"
					onClick={() => navigate("/dashboard")}
					style={{
						height: 56,
						display: "flex",
						alignItems: "center",
						padding: "0 20px",
						cursor: "pointer",
						borderBottom: "1px solid rgba(255,255,255,0.06)",
						width: "100%",
						background: "transparent",
						border: "none",
						textAlign: "left",
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
				</button>

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
								{
									key: "logout",
									icon: <LogoutOutlined />,
									label: "退出登录",
									danger: true,
								},
							],
							onClick: async () => {
								await logout();
							},
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
								<div
									style={{
										color: "#E2E8F0",
										fontSize: 13,
										fontWeight: 500,
										lineHeight: 1.3,
									}}
								>
									管理员
								</div>
								<div
									style={{ color: "#64748B", fontSize: 11, lineHeight: 1.3 }}
								>
									Phase 0
								</div>
							</div>
						</div>
					</Dropdown>
				</div>
			</Sider>

			<Layout
				style={{ marginLeft: isMobile ? 0 : siderWidth, background: "#F8FAFC" }}
			>
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
