import { Routes, Route, Navigate } from "react-router-dom";
import { LoginPage } from "@/pages/Login";
import { DashboardPage } from "@/pages/Dashboard";
import { MediaLibraryPage } from "@/pages/MediaLibrary";
import { PluginsPage } from "@/pages/Plugins";
import { WorkflowDashboardPage } from "@/pages/workflow/Dashboard";
import { EditorialDashboardPage } from "@/pages/editorial/Dashboard";
import { EditorialReviewPage } from "@/pages/editorial/Review";
import { NotFoundPage } from "@/pages/NotFound";
import { ProtectedRoute } from "@/components/ProtectedRoute";

function App() {
  return (
    <Routes>
      <Route path="/" element={<Navigate to="/admin/dashboard" replace />} />
      <Route path="/admin/login" element={<LoginPage />} />
      <Route path="/admin" element={<ProtectedRoute />}>
        <Route index element={<Navigate to="/admin/dashboard" replace />} />
        <Route path="dashboard" element={<DashboardPage />} />
        <Route path="media" element={<MediaLibraryPage />} />
        <Route path="workflow" element={<WorkflowDashboardPage />} />
        <Route path="editorial" element={<EditorialDashboardPage />} />
        <Route path="editorial/review/:id" element={<EditorialReviewPage />} />
        <Route path="plugins" element={<PluginsPage />} />
      </Route>
      <Route path="*" element={<NotFoundPage />} />
    </Routes>
  );
}

export default App;
