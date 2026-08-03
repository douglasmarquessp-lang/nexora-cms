import { useState, useEffect } from "react";
import { useNavigate, useSearchParams } from "react-router-dom";
import { useAuthStore } from "@/stores/auth";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";

export function LoginPage() {
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();
  const { login, isAuthenticated, isLoading, checkAuth } = useAuthStore();
  const [step, setStep] = useState<"credentials" | "mfa">("credentials");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [mfaCode, setMfaCode] = useState("");
  const [error, setError] = useState("");

  useEffect(() => {
    checkAuth();
  }, [checkAuth]);

  useEffect(() => {
    if (!isLoading && isAuthenticated) {
      const redirect = searchParams.get("redirect") || "/admin/dashboard";
      navigate(redirect, { replace: true });
    }
  }, [isLoading, isAuthenticated, navigate, searchParams]);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError("");

    try {
      const result = await login(email, password);
      if (result.status === "mfa_required") {
        setStep("mfa");
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : "Erro ao fazer login");
    }
  };

  const handleMfaSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError("");

    try {
      await login(email, password, mfaCode);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Erro ao verificar o código");
    }
  };

  if (isLoading) return null;

  return (
    <div className="flex min-h-screen items-center justify-center bg-gradient-to-br from-brand-50 to-white p-4">
      <Card className="w-full max-w-sm">
        <CardHeader className="text-center">
          <div className="mx-auto mb-3 flex h-10 w-10 items-center justify-center rounded-lg bg-brand-600 text-lg font-bold text-white">
            N
          </div>
          {step === "mfa" ? (
            <>
              <CardTitle className="text-xl">Verificação em duas etapas</CardTitle>
              <CardDescription>Digite o código do seu aplicativo autenticador</CardDescription>
            </>
          ) : (
            <>
              <CardTitle className="text-xl">Nexora CMS</CardTitle>
              <CardDescription>Faça login para continuar</CardDescription>
            </>
          )}
        </CardHeader>
        <CardContent>
          {error && (
            <div className="mb-4 rounded-md bg-destructive/10 p-3 text-sm text-destructive">
              {error}
            </div>
          )}

          {step === "mfa" ? (
            <form onSubmit={handleMfaSubmit} className="space-y-4">
              <div className="space-y-2">
                <Label htmlFor="mfa-code">Código</Label>
                <Input
                  id="mfa-code"
                  type="text"
                  inputMode="numeric"
                  required
                  value={mfaCode}
                  onChange={(e) => setMfaCode(e.target.value)}
                  placeholder="000000"
                  autoComplete="one-time-code"
                />
              </div>

              <Button type="submit" className="w-full">
                Verificar código
              </Button>

              <Button
                type="button"
                variant="ghost"
                className="w-full"
                onClick={() => {
                  setStep("credentials");
                  setMfaCode("");
                  setError("");
                }}
              >
                Voltar
              </Button>
            </form>
          ) : (
            <form onSubmit={handleSubmit} className="space-y-4">
              <div className="space-y-2">
                <Label htmlFor="email">Email</Label>
                <Input
                  id="email"
                  type="email"
                  required
                  value={email}
                  onChange={(e) => setEmail(e.target.value)}
                  placeholder="seu@email.com"
                  autoComplete="email"
                />
              </div>

              <div className="space-y-2">
                <Label htmlFor="password">Senha</Label>
                <Input
                  id="password"
                  type="password"
                  required
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  placeholder="********"
                  autoComplete="current-password"
                />
              </div>

              <Button type="submit" className="w-full">
                Entrar
              </Button>
            </form>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
