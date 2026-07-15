import {
	ApartmentOutlined,
	DashboardOutlined,
	DatabaseOutlined,
	FileOutlined,
	ForkOutlined,
	FundProjectionScreenOutlined,
	HistoryOutlined,
	KeyOutlined,
	LogoutOutlined,
	MenuFoldOutlined,
	MenuUnfoldOutlined,
	RobotOutlined,
	SendOutlined,
	SettingOutlined,
	UnorderedListOutlined,
	UserOutlined,
} from "@ant-design/icons";
import {
	Avatar,
	Button,
	Dropdown,
	Layout,
	Menu,
	Tooltip,
	Typography,
} from "antd";
import type { ReactNode } from "react";
import { useEffect, useState } from "react";
import { useLocation, useNavigate } from "react-router-dom";
import { useAuth } from "../hooks/useAuth";
import { useVersionCheck } from "../hooks/useVersionCheck";
import { getAppVersionLabel } from "../lib/appVersion";
import CmdKSearch from "./CmdKSearch";

const { Sider, Content } = Layout;

const menuItems = [
	{ key: "/dashboard", icon: <DashboardOutlined />, label: "概览" },
	{ type: "divider" as const },
	{ key: "/assets", icon: <DatabaseOutlined />, label: "资产管理" },
	{ key: "/mcap-files", icon: <FileOutlined />, label: "MCAP 文件" },
	{ type: "divider" as const },
	{ key: "/deliveries", icon: <SendOutlined />, label: "交付管理" },
	{ key: "/events", icon: <UnorderedListOutlined />, label: "事件流" },
	{ type: "divider" as const },
	{ key: "/algo-runs", icon: <HistoryOutlined />, label: "算法运行" },
	{ key: "/algo", icon: <RobotOutlined />, label: "算法处理" },
	{ type: "divider" as const },
	{ key: "/pipeline", icon: <ForkOutlined />, label: "流水线" },
	{ type: "divider" as const },
	{ key: "/registry", icon: <ApartmentOutlined />, label: "注册中心" },
	{
		key: "/metrics",
		icon: <FundProjectionScreenOutlined />,
		label: "指标检索",
	},
	{ key: "/api-keys", icon: <KeyOutlined />, label: "API 密钥" },
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
	if (pathname.startsWith("/runs")) return "/pipeline";
	if (pathname.startsWith("/workflows")) return "/pipeline";
	if (pathname.startsWith("/algo-runs")) return "/algo-runs";
	if (pathname.startsWith("/algo")) return "/algo";
	if (pathname.startsWith("/api-keys")) return "/api-keys";
	return "/assets";
}

function isPipelinePagePath(pathname: string): boolean {
	return (
		pathname.startsWith("/pipeline") ||
		pathname.startsWith("/runs") ||
		pathname.startsWith("/workflows")
	);
}

function isFullBleedPage(pathname: string): boolean {
	return isPipelinePagePath(pathname);
}

function resolvePageContainerClass(pathname: string): string {
	if (isFullBleedPage(pathname)) return "";
	if (pathname.startsWith("/dashboard") || pathname.startsWith("/metrics")) {
		return "page-container page-container--dashboard";
	}
	if (pathname.startsWith("/settings")) {
		return "page-container page-container--form";
	}
	if (pathname.startsWith("/algo") && !pathname.startsWith("/algo-runs")) {
		return "page-container page-container--matrix";
	}
	return "page-container page-container--table";
}

export default function AppLayout({ children }: { children: ReactNode }) {
	const navigate = useNavigate();
	const location = useLocation();
	const { logout, user } = useAuth();
	const siderWidth = 220;
	const [collapsed, setCollapsed] = useState<boolean>(
		() => localStorage.getItem("db.sider.collapsed") === "1",
	);
	useVersionCheck();

	const toggleCollapsed = () => {
		setCollapsed((prev) => {
			const next = !prev;
			localStorage.setItem("db.sider.collapsed", next ? "1" : "0");
			return next;
		});
	};
	const fullBleed = isFullBleedPage(location.pathname);
	const pageContainerClass = resolvePageContainerClass(location.pathname);
	const versionLabel = getAppVersionLabel();

	// biome-ignore lint/correctness/useExhaustiveDependencies: scroll to top on route changes
	useEffect(() => {
		window.scrollTo(0, 0);
	}, [location.pathname, location.search]);

	return (
		<Layout style={{ minHeight: "100vh" }}>
			<Sider
				width={siderWidth}
				breakpoint="lg"
				collapsed={collapsed}
				collapsedWidth={0}
				onBreakpoint={(broken) => setCollapsed(broken)}
				trigger={null}
				className="app-sider"
				style={{
					position: "fixed",
					zIndex: 100,
				}}
			>
				{/* logo area */}
				<button
					type="button"
					style={{
						height: 64,
						display: "flex",
						alignItems: "center",
						paddingLeft: 24,
						cursor: "pointer",
						background: "none",
						border: "none",
						width: "100%",
					}}
					onClick={() => navigate("/dashboard")}
				>
					<img
						src="/databrew-icon.svg"
						alt="DataBrew"
						style={{ width: 32, height: 32, marginRight: 12 }}
					/>
					<Typography.Title level={5} style={{ margin: 0, color: "#fff" }}>
						DataBrew
					</Typography.Title>
				</button>

				{/* search */}
				<div style={{ padding: "0 16px 12px" }}>
					<CmdKSearch />
				</div>

				{/* nav */}
				<div className="app-nav-wrapper">
					<Menu
						theme="dark"
						mode="inline"
						selectedKeys={[resolveSelectedKey(location.pathname)]}
						items={menuItems.filter(
							(m) =>
								!("key" in m) ||
								m.key !== "/api-keys" ||
								user?.role === "admin",
						)}
						onClick={({ key }) => navigate(key)}
						style={{ background: "transparent", borderRight: 0 }}
					/>
				</div>

				<div className="app-user-section">
					<div className="app-version" title={versionLabel}>
						{versionLabel}
					</div>
				</div>
			</Sider>
			<Layout style={{ marginLeft: collapsed ? 0 : siderWidth }}>
				<Content style={{ minHeight: "100vh" }}>
					<div
						style={{
							display: "flex",
							justifyContent: "space-between",
							alignItems: "center",
							padding: "12px 24px",
							background: "#fff",
							borderBottom: "1px solid #f0f0f0",
						}}
					>
						<Tooltip
							title={collapsed ? "展开导航栏" : "收起导航栏，扩大工作区"}
							placement="right"
						>
							<Button
								type="text"
								aria-label={collapsed ? "展开导航栏" : "收起导航栏"}
								icon={collapsed ? <MenuUnfoldOutlined /> : <MenuFoldOutlined />}
								onClick={toggleCollapsed}
							/>
						</Tooltip>
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
					<div
						className={fullBleed ? undefined : pageContainerClass}
						style={{
							padding: fullBleed ? 0 : 24,
							minHeight: fullBleed ? 0 : undefined,
						}}
					>
						{children}
					</div>
				</Content>
			</Layout>
		</Layout>
	);
}
