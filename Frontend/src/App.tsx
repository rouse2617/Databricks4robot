import { lazy, Suspense } from "react";
import { BrowserRouter, Route, Routes, Navigate } from "react-router-dom";
import { Spin } from "antd";
import AppLayout from "./components/AppLayout";
import ErrorBoundary from "./components/ErrorBoundary";
import { useAuth } from "./hooks/useAuth";

// Lazy-load pages to keep initial bundle small (steering: Priority 2 Bundle Size)
const DashboardPage = lazy(() => import("./pages/DashboardPage"));
const AssetsPage = lazy(() => import("./pages/AssetsPage"));
const AssetDetailPage = lazy(() => import("./pages/AssetDetailPage"));
const McapFilesPage = lazy(() => import("./pages/McapFilesPage"));
const AlgoProcessingPage = lazy(() => import("./pages/AlgoProcessingPage"));
const DeliveriesPage = lazy(() => import("./pages/DeliveriesPage"));
const DeliveryDetailPage = lazy(() => import("./pages/DeliveryDetailPage"));
const AnalyticsPage = lazy(() => import("./pages/AnalyticsPage"));
const TagDictionaryPage = lazy(() => import("./pages/TagDictionaryPage"));
const SettingsPage = lazy(() => import("./pages/SettingsPage"));
const LoginPage = lazy(() => import("./pages/LoginPage"));

function PageLoader() {
  return (
    <div className="flex items-center justify-center" style={{ height: "60vh" }}>
      <Spin size="large" />
    </div>
  );
}

function ProtectedRoutes() {
  const { isAuthenticated } = useAuth();
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
            <Route path="/deliveries" element={<DeliveriesPage />} />
            <Route path="/deliveries/:id" element={<DeliveryDetailPage />} />
            <Route path="/analytics" element={<AnalyticsPage />} />
            <Route path="/tags" element={<TagDictionaryPage />} />
            <Route path="/settings" element={<SettingsPage />} />
          </Routes>
        </Suspense>
      </ErrorBoundary>
    </AppLayout>
  );
}

export default function App() {
  return (
    <BrowserRouter>
      <Suspense fallback={<PageLoader />}>
        <Routes>
          <Route path="/login" element={<LoginPage />} />
          <Route path="/*" element={<ProtectedRoutes />} />
        </Routes>
      </Suspense>
    </BrowserRouter>
  );
}
