import { useState } from "react";
import { useParams } from "react-router-dom";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { api, ApiError } from "@/api/client";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { LoadingState } from "@/components/LoadingState";
import { EmptyState } from "@/components/EmptyState";
import { ErrorState } from "@/components/ErrorState";
import { cn, formatDate } from "@/lib/utils";
import { useCurrentSiteId, siteQueryKey } from "@/lib/queryKeys";
import {
  type ApprovalRequest,
  type ArticleReview,
  type PublishReadiness,
} from "@/lib/editorial";

const SEVERITY_COLORS: Record<string, string> = {
  error: "bg-red-50 text-red-700 border-red-200",
  warning: "bg-yellow-50 text-yellow-700 border-yellow-200",
  info: "bg-blue-50 text-blue-700 border-blue-200",
};

const REC_STATUS_COLORS: Record<string, string> = {
  ok: "bg-green-50 text-green-700 border-green-200",
  warning: "bg-yellow-50 text-yellow-700 border-yellow-200",
  fail: "bg-red-50 text-red-700 border-red-200",
  info: "bg-blue-50 text-blue-700 border-blue-200",
};

function ScoreCard({ label, value, threshold }: { label: string; value: number; threshold?: number }) {
  const pct = Math.max(0, Math.min(100, value));
  return (
    <Card>
      <CardContent className="p-4">
        <div className="flex items-baseline justify-between">
          <p className="text-xs font-medium text-muted-foreground">{label}</p>
          <p className="text-lg font-semibold" data-testid={`score-${label}`}>
            {value.toFixed(1)}
          </p>
        </div>
        <div className="mt-2 h-1.5 w-full overflow-hidden rounded-full bg-muted">
          <div
            className={cn(
              "h-full rounded-full",
              pct >= 70 ? "bg-green-500" : pct >= 50 ? "bg-yellow-500" : "bg-red-500",
            )}
            style={{ width: `${pct}%` }}
          />
        </div>
        {threshold ? (
          <p className="mt-1 text-[11px] text-muted-foreground">Limite: {threshold}</p>
        ) : null}
      </CardContent>
    </Card>
  );
}

function StatusBadge({ status }: { status: string }) {
  const colors: Record<string, string> = {
    approved: "bg-green-50 text-green-700",
    needs_review: "bg-yellow-50 text-yellow-700",
    draft: "bg-muted text-muted-foreground",
    scheduled: "bg-slate-100 text-slate-800",
    published: "bg-emerald-50 text-emerald-700",
  };
  return (
    <span className={cn("rounded px-2 py-0.5 text-xs font-medium", colors[status] ?? "bg-muted text-muted-foreground")}>
      {status}
    </span>
  );
}

function RecommendationBlock({ recommendations }: { recommendations: ArticleReview["recommendations"] }) {
  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-sm">IA recomenda</CardTitle>
      </CardHeader>
      <CardContent className="space-y-2">
        {recommendations.length === 0 && (
          <p className="text-sm text-muted-foreground">Sem recomendações.</p>
        )}
        {recommendations.map((rec) => (
          <div
            key={rec.label}
            className={cn("rounded-md border p-2.5", REC_STATUS_COLORS[rec.status] ?? "border-border")}
            data-testid={`recommendation-${rec.label}`}
          >
            <div className="flex items-center justify-between">
              <p className="text-sm font-medium">{rec.label}</p>
              {rec.score != null && (
                <p className="text-sm font-semibold">{rec.score.toFixed(0)}/100</p>
              )}
            </div>
            {rec.details && rec.details.length > 0 && (
              <ul className="mt-1.5 list-disc space-y-0.5 pl-4 text-xs text-muted-foreground">
                {rec.details.map((d, i) => (
                  <li key={i}>{d}</li>
                ))}
              </ul>
            )}
          </div>
        ))}
      </CardContent>
    </Card>
  );
}

function ProblemsBlock({ problems }: { problems: ArticleReview["problems"] }) {
  if (problems.length === 0) {
    return (
      <Card>
        <CardHeader>
          <CardTitle className="text-sm">Problemas</CardTitle>
        </CardHeader>
        <CardContent>
          <EmptyState title="Nenhum problema" description="Nenhuma pendência detectada." />
        </CardContent>
      </Card>
    );
  }
  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-sm">Problemas ({problems.length})</CardTitle>
      </CardHeader>
      <CardContent className="space-y-2">
        {problems.map((problem, i) => (
          <div
            key={i}
            className={cn(
              "rounded-md border p-2.5 text-sm",
              SEVERITY_COLORS[problem.severity] ?? "border-border",
            )}
            data-testid={`problem-${problem.kind}`}
          >
            <div className="flex items-center gap-2">
              <span className="rounded bg-background px-1.5 py-0.5 text-[11px] font-medium">
                {problem.kind}
              </span>
              <span className="text-xs font-medium uppercase">{problem.severity}</span>
            </div>
            <p className="mt-1 text-xs">{problem.message}</p>
          </div>
        ))}
      </CardContent>
    </Card>
  );
}

function SourcesBlock({ sources }: { sources: ArticleReview["sources"] }) {
  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-sm">Fontes ({sources.length})</CardTitle>
      </CardHeader>
      <CardContent className="space-y-2">
        {sources.length === 0 && (
          <p className="text-sm text-muted-foreground">Nenhuma fonte vinculada.</p>
        )}
        {sources.map((src) => (
          <div key={src.url} className="rounded-md border p-2.5">
            <div className="flex items-center justify-between gap-2">
              <a
                href={src.url}
                target="_blank"
                rel="noreferrer noopener"
                className="truncate text-sm font-medium text-brand-600 hover:underline"
              >
                {src.title || src.url}
              </a>
              <span
                className={cn(
                  "shrink-0 rounded px-1.5 py-0.5 text-[11px] font-medium",
                  src.is_verified ? "bg-green-50 text-green-700" : "bg-yellow-50 text-yellow-700",
                )}
              >
                {src.is_verified ? "verificada" : "não verificada"}
              </span>
            </div>
            {src.snippet && <p className="mt-1 line-clamp-2 text-xs text-muted-foreground">{src.snippet}</p>}
            {src.freshness_score != null && (
              <p className="mt-1 text-[11px] text-muted-foreground">
                Freshness {src.freshness_score.toFixed(2)} · Relevância {src.relevance_score ?? 0}
              </p>
            )}
          </div>
        ))}
      </CardContent>
    </Card>
  );
}

function LinksBlock({
  title,
  links,
  testId,
}: {
  title: string;
  links: ArticleReview["internal_links"] | ArticleReview["external_links"];
  testId: string;
}) {
  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-sm">{title} ({links.length})</CardTitle>
      </CardHeader>
      <CardContent className="space-y-2">
        {links.length === 0 && <p className="text-sm text-muted-foreground">Nenhuma sugestão.</p>}
        {links.map((link) => (
          <div key={link.url + link.title} className="flex items-center justify-between gap-2 rounded-md border p-2.5" data-testid={testId}>
            <div className="min-w-0">
              <a
                href={link.url}
                target="_blank"
                rel="noreferrer noopener"
                className="block truncate text-sm font-medium text-brand-600 hover:underline"
              >
                {link.title}
              </a>
              <p className="text-[11px] text-muted-foreground">
                {link.url}
                {link.label ? ` · ${link.label}` : ""}
              </p>
            </div>
            <span className="shrink-0 rounded bg-muted px-1.5 py-0.5 text-xs font-medium">
              {link.score.toFixed(0)}
            </span>
          </div>
        ))}
      </CardContent>
    </Card>
  );
}

export function EditorialReviewPage() {
  const { id } = useParams<{ id: string }>();
  const currentSiteId = useCurrentSiteId();
  const queryClient = useQueryClient();
  const [scheduleDate, setScheduleDate] = useState("");
  const [readiness, setReadiness] = useState<PublishReadiness | null>(null);
  const [publishError, setPublishError] = useState("");

  const reviewQuery = useQuery({
    queryKey: siteQueryKey(["editorial-review", id], currentSiteId),
    queryFn: () => api.get<ArticleReview>(`/editorial/review/${id}`),
    enabled: !!currentSiteId && !!id,
  });

  const approvalsQuery = useQuery({
    queryKey: siteQueryKey(["editorial-approvals", id], currentSiteId),
    queryFn: () => api.get<ApprovalRequest[]>(`/editorial/approvals`, { params: { post_id: id ?? "" } }),
    enabled: !!currentSiteId && !!id,
  });

  const invalidate = () => {
    queryClient.invalidateQueries({ queryKey: ["editorial-review"] });
    queryClient.invalidateQueries({ queryKey: ["editorial-approvals"] });
    queryClient.invalidateQueries({ queryKey: ["editorial-pipeline"] });
    queryClient.invalidateQueries({ queryKey: ["editorial-stats"] });
    queryClient.invalidateQueries({ queryKey: ["dashboard-editorial"] });
  };

  const publishMutation = useMutation({
    mutationFn: () => api.post<{ status: string }>("/publisher/publish", { post_id: id }),
    onSuccess: () => {
      toast.success("Artigo publicado com sucesso");
      invalidate();
    },
    onError: (error) => {
      if (error instanceof ApiError && error.status === 422) {
        setPublishError(error.message);
        api
          .get<PublishReadiness>(`/editorial/publish-readiness/${id}`)
          .then(setReadiness)
          .catch(() => setReadiness(null));
        return;
      }
      toast.error(error instanceof ApiError ? error.message : "Falha ao publicar");
    },
  });

  const scheduleMutation = useMutation({
    mutationFn: () =>
      api.patch<{ status: string }>(`/posts/${id}/status`, {
        status: "scheduled",
        scheduled_at: scheduleDate ? new Date(scheduleDate).toISOString() : undefined,
      }),
    onSuccess: () => {
      toast.success("Artigo agendado");
      invalidate();
    },
    onError: (error) =>
      toast.error(error instanceof ApiError ? error.message : "Falha ao agendar"),
  });

  const draftMutation = useMutation({
    mutationFn: () => api.patch<{ status: string }>(`/posts/${id}/status`, { status: "draft" }),
    onSuccess: () => {
      toast.success("Movido para rascunho");
      invalidate();
    },
  });

  const requestApprovalMutation = useMutation({
    mutationFn: () => api.post(`/editorial/posts/${id}/approvals`),
    onSuccess: () => {
      toast.success("Aprovação solicitada");
      approvalsQuery.refetch();
    },
  });

  const reviewApprovalMutation = useMutation({
    mutationFn: ({ approvalID, status }: { approvalID: string; status: string }) =>
      api.put(`/editorial/posts/${id}/approvals/${approvalID}/review`, { status }),
    onSuccess: () => {
      toast.success("Aprovação registrada");
      approvalsQuery.refetch();
      invalidate();
    },
  });

  if (reviewQuery.isLoading) return <LoadingState text="Carregando revisão…" />;
  if (reviewQuery.isError || !reviewQuery.data) {
    return (
      <ErrorState
        title="Falha ao carregar revisão"
        message="Não foi possível buscar os dados do artigo."
        onRetry={() => reviewQuery.refetch()}
      />
    );
  }

  const review = reviewQuery.data;
  const approvals = approvalsQuery.data ?? [];
  const pendingApproval = approvals.filter((a) => a.post_id === id).find((a) => a.status === "pending");

  return (
    <div className="space-y-4">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div className="min-w-0">
          <div className="flex items-center gap-2">
            <h1 className="truncate text-xl font-semibold">{review.post.title}</h1>
            <StatusBadge status={review.post.status} />
          </div>
          <p className="text-sm text-muted-foreground">
            /{review.post.slug} · {review.post.language ?? "pt"} · atualizado em{" "}
            {formatDate(review.post.updated_at)}
          </p>
        </div>
        <div className="flex flex-wrap items-center gap-2">
          <Button
            variant="outline"
            size="sm"
            onClick={() => draftMutation.mutate()}
            data-testid="review-action-draft"
          >
            Rascunho
          </Button>
          <div className="flex items-center gap-2">
            <Input
              type="datetime-local"
              value={scheduleDate}
              onChange={(e) => setScheduleDate(e.target.value)}
              className="h-8 w-auto text-xs"
              aria-label="Data de agendamento"
              data-testid="review-schedule-date"
            />
            <Button
              variant="outline"
              size="sm"
              disabled={!scheduleDate}
              onClick={() => scheduleMutation.mutate()}
              data-testid="review-action-schedule"
            >
              Agendar
            </Button>
          </div>
          {pendingApproval ? (
            <Button
              size="sm"
              variant="outline"
              onClick={() =>
                reviewApprovalMutation.mutate({ approvalID: pendingApproval.id, status: "approved" })
              }
              data-testid="review-action-approve"
            >
              Aprovar
            </Button>
          ) : (
            <Button
              size="sm"
              variant="outline"
              onClick={() => requestApprovalMutation.mutate()}
              data-testid="review-action-request-approval"
            >
              Solicitar aprovação
            </Button>
          )}
          <Button size="sm" onClick={() => publishMutation.mutate()} data-testid="review-action-publish">
            Publicar
          </Button>
        </div>
      </div>

      {review.review ? (
        <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
          <ScoreCard label="SEO" value={review.review.seo} />
          <ScoreCard label="EEAT" value={review.review.eeat} />
          <ScoreCard label="Freshness" value={review.review.freshness} />
          <ScoreCard label="Cobertura" value={review.review.coverage} />
          <ScoreCard label="Naturalidade" value={review.review.naturalness} />
          <ScoreCard label="Confiança" value={review.review.confidence} />
          <ScoreCard label="Final" value={review.review.final} threshold={review.review.threshold} />
          <Card>
            <CardHeader>
              <CardTitle className="text-sm">Decisão</CardTitle>
            </CardHeader>
            <CardContent>
              <StatusBadge status={review.review.decision} />
              <p className="mt-2 text-[11px] text-muted-foreground">
                {formatDate(review.review.created_at)}
              </p>
            </CardContent>
          </Card>
        </div>
      ) : (
        <Card>
          <CardContent className="p-4">
            <p className="text-sm text-muted-foreground">
              Este artigo ainda não passou pela nota editorial (IA Editorial Brain).
            </p>
          </CardContent>
        </Card>
      )}

      <div className="grid gap-3 lg:grid-cols-2">
        <RecommendationBlock recommendations={review.recommendations} />
        <ProblemsBlock problems={review.problems} />
      </div>

      <SourcesBlock sources={review.sources} />

      <div className="grid gap-3 lg:grid-cols-2">
        <LinksBlock title="Sugestões de links internos" links={review.internal_links} testId="review-internal-link" />
        <LinksBlock title="Links externos (fontes confiáveis)" links={review.external_links} testId="review-external-link" />
      </div>

      <Dialog
        open={publishError !== ""}
        onOpenChange={(open) => {
          if (!open) {
            setPublishError("");
            setReadiness(null);
          }
        }}
      >
        <DialogContent data-testid="review-readiness-dialog">
          <DialogHeader>
            <DialogTitle>Publicação bloqueada</DialogTitle>
            <DialogDescription>{publishError}</DialogDescription>
          </DialogHeader>
          {readiness ? (
            <div className="space-y-2" data-testid="review-readiness-checks">
              {readiness.checks.map((check) => (
                <div
                  key={check.stage}
                  className={cn(
                    "flex items-start justify-between gap-3 rounded-md border p-2 text-sm",
                    check.passed ? "border-green-200 bg-green-50" : "border-red-200 bg-red-50",
                  )}
                >
                  <div>
                    <p className="font-medium">{check.label}</p>
                    <p className="text-xs text-muted-foreground">{check.message}</p>
                  </div>
                  <span className="text-xs font-semibold">{check.passed ? "OK" : "Falhou"}</span>
                </div>
              ))}
            </div>
          ) : (
            <p className="text-sm text-muted-foreground">Carregando validações…</p>
          )}
          <div className="flex justify-end">
            <Button
              onClick={() => {
                setPublishError("");
                setReadiness(null);
              }}
            >
              Fechar
            </Button>
          </div>
        </DialogContent>
      </Dialog>
    </div>
  );
}
