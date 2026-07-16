import { Routes, Route, Navigate } from "react-router-dom";
import { DashboardPage } from "@/pages/Dashboard";
import { LoginPage } from "@/pages/Login";
import { MediaLibraryPage } from "@/pages/MediaLibrary";
import { PluginsPage } from "@/pages/Plugins";
import { NotFoundPage } from "@/pages/NotFound";

function App() {
  return (
    <Routes>
      <Route path="/" element={<Navigate to="/admin/dashboard" replace />} />
      <Route path="/admin/login" element={<LoginPage />} />
      <Route path="/admin/dashboard" element={<DashboardPage />} />
      <Route path="/admin/media" element={<MediaLibraryPage />} />
      <Route path="/admin/plugins" element={<PluginsPage />} />
      <Route path="*" element={<NotFoundPage />} />
    </Routes>
  );
}

export default App;
