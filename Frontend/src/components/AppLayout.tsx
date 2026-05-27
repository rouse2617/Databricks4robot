import {
	ApartmentOutlined,
	DashboardOutlined,
	DatabaseOutlined,
	FileOutlined,
	FundProjectionScreenOutlined,
	HistoryOutlined,
	LogoutOutlined,
	RobotOutlined,
	SendOutlined,
	SettingOutlined,
	UnorderedListOutlined,
	UserOutlined,
} from "@ant-design/icons";
import { Avatar, Dropdown, Grid, Layout, Menu, Typography } from "antd";
import type { ReactNode } from "react";
import { useEffect } from "react";
import { useLocation, useNavigate } from "react-router-dom";
import { useAuth } from "../hooks/useAuth";
import CmdKSearch from "./CmdKSearch";

const { Sider, Content } = Layout;
const { useBreakpoint } = Grid;

const menuItems = [
	{ key: "/dashboard", icon: <DashboardOutlined />, label: "概览" },
	{ type: "divider" as const },
	{ key: "/assets", icon: <DatabaseOutlined />, label: "资产管理" },
	{ key: "/mcap-files", icon: <FileOutlined />, label: "MCAP 文件" },
	{ type: "divider" as const },
	{ key: "/deliveries", icon: <SendOutlined />, label: "交付管理" },
	{ key: "/events", icon: <UnorderedListOutlined />, label: "事件流" },
	{ type: "divider" as const },
	{ key: "/algo-runs", icon: <HistoryOutlined />, label: "运行记录" },
	{ key: "/algo", icon: <RobotOutlined />, label: "算法处理" },
	{ type: "divider" as const },
	{ key: "/pipeline", icon: <ApartmentOutlined />, label: "流水线" },
	{ type: "divider" as const },
	{ key: "/registry", icon: <ApartmentOutlined />, label: "注册中心" },
	{
		key: "/metrics",
		icon: <FundProjectionScreenOutlined />,
		label: "指标检索",
	},
	{ key: "/settings", icon: <SettingOutlined />, label: "设置" },
];

function resolveSelectedKey(pathname: string): string {
	if (pathname.startsWith("/dashboard")) return "/dashboard";
	if (pathname.startsWith("/assets")) return "/assets";
	if (pathname.startsWith("/metrics")) return "/metrics";
	if (pathname.startsWith("/deliveries")) return "/deliveries";
	if (pathname.startsWith("/mcap-files")) return "/mcap-files";
	if (pathname.startsWith("/events")) return "/events";
	if (pathname.startsWith("/settings")) return "/settings";
	if (pathname.startsWith("/registry")) return "/registry";
	if (pathname.startsWith("/pipeline")) return "/pipeline";
	if (pathname.startsWith("/algo-runs")) return "/algo-runs";
	if (pathname.startsWith("/algo")) return "/algo";
	return "/assets";
}

export default function AppLayout({ children }: { children: ReactNode }) {
	const navigate = useNavigate();
	const location = useLocation();
	const { logout } = useAuth();
	const screens = useBreakpoint();
	const isMobile = !screens.lg;
	const siderWidth = 220;

	// Browser restores scroll position on back-nav, no dep needed
	useEffect(() => {
		window.scrollTo(0, 0);
	}, []);

	return (
		<>
			<Layout style={{ minHeight: "100vh" }}>
				<Sider
					width={siderWidth}
					breakpoint="lg"
					collapsedWidth={0}
					trigger={null}
					style={{
						overflow: "auto",
						height: "100vh",
						position: "fixed",
						left: 0,
						top: 0,
						bottom: 0,
						zIndex: 100,
					}}
				>
					{/* logo area */}
					<div
						role="button"
						tabIndex={0}
						style={{
							height: 64,
							display: "flex",
							alignItems: "center",
							paddingLeft: 24,
							cursor: "pointer",
						}}
						onClick={() => navigate("/dashboard")}
						onKeyDown={(e) => {
							if (e.key === "Enter") navigate("/dashboard");
						}}
					>
						<img
							src="/favicon.svg"
							alt="DataBrew"
							style={{ width: 28, height: 28, marginRight: 10 }}
						/>
						<Typography.Title level={5} style={{ margin: 0, color: "#fff" }}>
							DataBrew
						</Typography.Title>
					</div>

					{/* search */}
					<div style={{ padding: "0 16px 12px" }}>
						<CmdKSearch />
					</div>

					{/* nav */}
					<Menu
						theme="dark"
						mode="inline"
						selectedKeys={[resolveSelectedKey(location.pathname)]}
						items={menuItems}
						onClick={({ key }) => {
							if (key === "/pipeline") {
								window.open(
									"https://cyber-databrew-pipeline-ui-dev-234851712830.us-central1.run.app/pipeline",
									"_blank",
								);
							} else {
								navigate(key);
							}
						}}
					/>
				</Sider>
				<Layout style={{ marginLeft: isMobile ? 0 : siderWidth }}>
					<Content style={{ minHeight: "100vh" }}>
						<div
							style={{
								display: "flex",
								justifyContent: "flex-end",
								padding: "12px 24px",
								background: "#fff",
								borderBottom: "1px solid #f0f0f0",
							}}
						>
							<Dropdown
								menu={{
									items: [
										{
											key: "logout",
											icon: <LogoutOutlined />,
											label: "退出登录",
											onClick: () => {
												logout();
												navigate("/login");
											},
										},
									],
								}}
								placement="bottomRight"
							>
								<Avatar
									size="small"
									icon={<UserOutlined />}
									style={{ cursor: "pointer", backgroundColor: "#1677ff" }}
								/>
							</Dropdown>
						</div>
						<div style={{ padding: 24 }}>{children}</div>
					</Content>
				</Layout>
			</Layout>
		</>
	);
}
