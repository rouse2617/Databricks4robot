import { BrowserRouter, Route, Routes, Navigate } from "react-router-dom";
import AppLayout from "./components/AppLayout";
import AssetsPage from "./pages/AssetsPage";
import AssetDetailPage from "./pages/AssetDetailPage";
import McapFilesPage from "./pages/McapFilesPage";
import DeliveriesPage from "./pages/DeliveriesPage";
import LoginPage from "./pages/LoginPage";
import { useAuth } from "./hooks/useAuth";

function ProtectedRoutes() {
  const { isAuthenticated } = useAuth();
  if (!isAuthenticated) return <Navigate to="/login" replace />;
  return (
    <AppLayout>
      <Routes>
        <Route path="/" element={<Navigate to="/assets" replace />} />
        <Route path="/assets" element={<AssetsPage />} />
        <Route path="/assets/:id" element={<AssetDetailPage />} />
        <Route path="/mcap-files" element={<McapFilesPage />} />
        <Route path="/deliveries" element={<DeliveriesPage />} />
      </Routes>
    </AppLayout>
  );
}

export default function App() {
  return (
    <BrowserRouter>
      <Routes>
        <Route path="/login" element={<LoginPage />} />
        <Route path="/*" element={<ProtectedRoutes />} />
      </Routes>
    </BrowserRouter>
  );
}
