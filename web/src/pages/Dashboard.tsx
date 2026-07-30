import { useQuery } from "@tanstack/react-query";
import { api } from "@/api/client";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { Activity, Server, User } from "lucide-react";

export function DashboardPage() {
  const { data: health, isLoading } = useQuery({
    queryKey: ["health"],
    queryFn: () => api.get<{ status: string; version: string; timestamp: string }>("/health"),
  });

  const cards = [
    {
      title: "Status do Sistema",
      value: health?.status ?? "---",
      icon: Server,
      color: "text-green-600",
    },
    {
      title: "Versão",
      value: health?.version ? `v${health.version}` : "---",
      icon: Activity,
      color: "text-brand-600",
    },
  ];

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-semibold tracking-tight">Dashboard</h1>
        <p className="text-sm text-muted-foreground">Visão geral do sistema</p>
      </div>

      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
        {cards.map((card) => (
          <Card key={card.title}>
            <CardHeader className="flex flex-row items-center justify-between pb-2">
              <CardTitle className="text-sm font-medium text-muted-foreground">
                {card.title}
              </CardTitle>
              <card.icon className="h-4 w-4 text-muted-foreground" />
            </CardHeader>
            <CardContent>
              {isLoading ? (
                <Skeleton className="h-7 w-24" />
              ) : (
                <div className={`text-2xl font-bold ${card.color}`}>{card.value}</div>
              )}
            </CardContent>
          </Card>
        ))}
      </div>
    </div>
  );
}
