import { Spin } from "antd";
import { lazy, Suspense } from "react";
import { BrowserRouter, Navigate, Route, Routes } from "react-router-dom";
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
const LoginPage = lazy(() => import("./pages/LoginPage"));

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
	if (loading) return <PageLoader />;
	if (!isAuthenticated) return <Navigate to="/login" replace />;
	return (
		<AppLayout>
			<ErrorBoundary>
				<Suspense fallback={<PageLoader />}>
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
							path="/lakehouse"
							element={<Navigate to="/dashboard" replace />}
						/>
						<Route path="/registry" element={<RegistryCenterPage />} />
						<Route path="/tags" element={<Navigate to="/registry" replace />} />
						<Route path="/metrics" element={<MetricsSearchPage />} />
						<Route path="/events" element={<EventsPage />} />
						<Route path="/pipeline" element={<PipelinePage />} />
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
