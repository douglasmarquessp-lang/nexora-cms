import { Link } from "react-router-dom";
import { Button } from "@/components/ui/button";

export function NotFoundPage() {
  return (
    <div className="flex min-h-screen flex-col items-center justify-center bg-gradient-to-br from-brand-50 to-white">
      <h1 className="text-6xl font-bold text-brand-600">404</h1>
      <p className="mt-2 text-muted-foreground">Página não encontrada</p>
      <Link to="/admin/dashboard" className="mt-4">
        <Button>Voltar ao dashboard</Button>
      </Link>
    </div>
  );
}
