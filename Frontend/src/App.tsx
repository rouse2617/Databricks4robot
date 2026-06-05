import { Spin } from "antd";
import { lazy, Suspense, useEffect } from "react";
import {
	BrowserRouter,
	Navigate,
	Route,
	Routes,
	useLocation,
} from "react-router-dom";
import AppLayout from "./components/AppLayout";
import ErrorBoundary from "./components/ErrorBoundary";
import { AuthProvider, useAuth } from "./hooks/useAuth";
import "./styles/design-tokens.css";

// Lazy-load pages to keep initial bundle small (steering: Priority 2 Bundle Size)
const DashboardPage = lazy(() => import("./pages/DashboardPage"));
const AssetsPage = lazy(() => import("./pages/AssetsPage"));
const AssetDetailPage = lazy(() => import("./pages/AssetDetailPage"));
const McapFilesPage = lazy(() => import("./pages/McapFilesPage"));
const AlgoProcessingPage = lazy(() => import("./pages/AlgoProcessingPage"));
const AlgoRunsPage = lazy(() => import("./pages/AlgoRunsPage"));
const AlgoRunDetailPage = lazy(() => import("./pages/AlgoRunDetailPage"));
const DeliveriesPage = lazy(() => import("./pages/DeliveriesPage"));
const DeliveryDetailPage = lazy(() => import("./pages/DeliveryDetailPage"));
const RegistryCenterPage = lazy(() => import("./pages/RegistryCenterPage"));
const MetricsSearchPage = lazy(() => import("./pages/MetricsSearchPage"));
const SettingsPage = lazy(() => import("./pages/SettingsPage"));
const EventsPage = lazy(() => import("./pages/EventsPage"));
const PipelinePage = lazy(() => import("./pages/PipelinePage"));
const WorkflowDetailPage = lazy(() => import("./pages/WorkflowDetailPage"));
const LoginPage = lazy(() => import("./pages/LoginPage"));

const pathTitles: Record<string, string> = {
	"/dashboard": "概览 - Cyber Databrew",
	"/assets": "资产管理 - Cyber Databrew",
	"/mcap-files": "MCAP 文件 - Cyber Databrew",
	"/algo": "算法处理 - Cyber Databrew",
	"/algo-runs": "运行记录 - Cyber Databrew",
	"/deliveries": "交付管理 - Cyber Databrew",
	"/registry": "注册中心 - Cyber Databrew",
	"/metrics": "指标检索 - Cyber Databrew",
	"/events": "事件流 - Cyber Databrew",
	"/pipeline": "流水线设计 - Cyber Databrew",
	"/settings": "设置 - Cyber Databrew",
};

function PageLoader() {
	return (
		<div
			className="flex items-center justify-center"
			style={{ height: "60vh" }}
		>
			<Spin size="large" />
		</div>
	);
}

function ProtectedRoutes() {
	const { isAuthenticated, loading } = useAuth();
	const location = useLocation();

	useEffect(() => {
		const matchedPath = Object.keys(pathTitles).find(
			(path) =>
				location.pathname === path || location.pathname.startsWith(`${path}/`),
		);
		document.title = matchedPath ? pathTitles[matchedPath] : "Cyber Databrew";
	}, [location.pathname]);

	if (loading) return <PageLoader />;
	if (!isAuthenticated) return <Navigate to="/login" replace />;
	return (
		<AppLayout>
			<ErrorBoundary>
				<Suspense fallback={<PageLoader />} key={location.pathname}>
					<Routes>
						<Route path="/" element={<Navigate to="/dashboard" replace />} />
						<Route path="/dashboard" element={<DashboardPage />} />
						<Route path="/assets" element={<AssetsPage />} />
						<Route path="/assets/:id" element={<AssetDetailPage />} />
						<Route path="/mcap-files" element={<McapFilesPage />} />
						<Route path="/algo" element={<AlgoProcessingPage />} />
						<Route path="/algo-runs" element={<AlgoRunsPage />} />
						<Route path="/algo-runs/:run_id" element={<AlgoRunDetailPage />} />
						<Route path="/deliveries" element={<DeliveriesPage />} />
						<Route path="/deliveries/:id" element={<DeliveryDetailPage />} />
						<Route
							path="/delivery"
							element={<Navigate to="/deliveries" replace />}
						/>
						<Route
							path="/lakehouse"
							element={<Navigate to="/dashboard" replace />}
						/>
						<Route path="/registry" element={<RegistryCenterPage />} />
						<Route path="/tags" element={<Navigate to="/registry" replace />} />
						<Route path="/metrics" element={<MetricsSearchPage />} />
						<Route path="/events" element={<EventsPage />} />
						<Route path="/pipeline" element={<PipelinePage />} />
						<Route
							path="/components"
							element={<Navigate to="/pipeline?tab=components" replace />}
						/>
						<Route
							path="/workflows"
							element={<Navigate to="/pipeline?tab=executions" replace />}
						/>
						<Route
							path="/workflows/:name"
							element={<WorkflowDetailPage legacyRoute />}
						/>
						<Route
							path="/pipeline/executions/:name"
							element={<WorkflowDetailPage />}
						/>
						<Route path="/settings" element={<SettingsPage />} />
						<Route path="*" element={<Navigate to="/dashboard" replace />} />
					</Routes>
				</Suspense>
			</ErrorBoundary>
		</AppLayout>
	);
}

export default function App() {
	return (
		<AuthProvider>
			<BrowserRouter>
				<Suspense fallback={<PageLoader />}>
					<Routes>
						<Route path="/login" element={<LoginPage />} />
						<Route path="/*" element={<ProtectedRoutes />} />
					</Routes>
				</Suspense>
			</BrowserRouter>
		</AuthProvider>
	);
}
