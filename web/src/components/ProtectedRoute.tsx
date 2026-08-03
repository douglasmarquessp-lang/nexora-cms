import { useEffect } from "react";
import { useNavigate, useLocation } from "react-router-dom";
import { useAuthStore } from "@/stores/auth";
import { AdminLayout } from "./AdminLayout";
import { Skeleton } from "@/components/ui/skeleton";

export function ProtectedRoute() {
  const navigate = useNavigate();
  const location = useLocation();
  const { isAuthenticated, isLoading, checkAuth } = useAuthStore();

  useEffect(() => {
    checkAuth();
  }, [checkAuth]);

  useEffect(() => {
    if (!isLoading && !isAuthenticated) {
      const redirect = encodeURIComponent(location.pathname + location.search);
      navigate(`/admin/login?redirect=${redirect}`, { replace: true });
    }
  }, [isLoading, isAuthenticated, navigate, location.pathname, location.search]);

  if (isLoading) {
    return (
      <div className="flex h-screen items-center justify-center">
        <div className="flex flex-col items-center gap-4">
          <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-brand-600 text-sm font-bold text-white">
            N
          </div>
          <div className="space-y-2">
            <Skeleton className="h-4 w-48" />
            <Skeleton className="h-4 w-36" />
          </div>
        </div>
      </div>
    );
  }

  if (!isAuthenticated) {
    return null;
  }

  return <AdminLayout />;
}
