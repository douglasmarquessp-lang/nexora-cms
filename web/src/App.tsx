import { Routes, Route, Navigate } from "react-router-dom";
import { DashboardPage } from "@/pages/Dashboard";
import { LoginPage } from "@/pages/Login";
import { NotFoundPage } from "@/pages/NotFound";

function App() {
  return (
    <Routes>
      <Route path="/" element={<Navigate to="/admin/dashboard" replace />} />
      <Route path="/admin/login" element={<LoginPage />} />
      <Route path="/admin/dashboard" element={<DashboardPage />} />
      <Route path="*" element={<NotFoundPage />} />
    </Routes>
  );
}

export default App;
