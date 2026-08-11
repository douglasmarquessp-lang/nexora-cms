import { useState } from "react";
import { useParams, useNavigate } from "react-router-dom";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "@/api/client";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { LoadingState } from "@/components/LoadingState";
import { EmptyState } from "@/components/EmptyState";
import { ErrorState } from "@/components/ErrorState";
import { cn } from "@/lib/utils";
import { useCurrentSiteId, siteQueryKey } from "@/lib/queryKeys";
import { Check, ExternalLink, ImageOff, RefreshCw, Send, X } from "lucide-react";

interface ImageCredit {
  photographer: string;
  photographer_url: string;
  source_url: string;
}

interface ReviewArticle {
  title: string;
  slug?: string;
  content: string;
  keyword?: string;
  meta_title?: string;
  meta_description?: string;
  featured_image_url?: string;
  featured_image_alt?: string;
  featured_image_credit?: ImageCredit;
  author_name?: string;
  language: string;
  excerpt?: string;
  categories?: string[];
  tags?: string[];
  word_count: number;
  reading_time: number;
}

interface ReviewSEO {
  score: number;
  min_score: number;
  passes: boolean;
  title?: number;
  meta?: number;
  headings?: number;
  keyword?: number;
  readability?: number;
  internal_links?: number;
  external_links?: number;
  eeat?: number;
  images?: number;
  keyword_density?: number;
  word_count?: number;
  issues?: string[];
}

interface ReviewSource {
  id: string;
  url: string;
  title?: string;
  snippet?: string;
  domain?: string;
  reliability_score: number;
  reliability_label?: string;
  is_verified?: boolean;
  published_at?: string;
  retrieved_at?: string;
}

interface WorkflowJob {
  id: string;
  title: string;
  status: string;
  current_step: string;
  progress: number;
  language: string;
  error_message?: string;
  publication_id?: string;
  created_at: string;
  review_status: string;
  revision: number;
  rejection_reason?: string;
}

interface JobReviewDetail {
  job: WorkflowJob;
  article?: ReviewArticle;
  seo?: ReviewSEO;
  version: number;
  approver_id?: string;
  sources?: ReviewSource[];
}

const REVIEW_STATUS_COLORS: Record<string, string> = {
  generated: "bg-yellow-50 text-yellow-700",
  approved: "bg-blue-50 text-blue-700",
  rejected: "bg-red-50 text-red-700",
  published: "bg-green-50 text-green-700",
};

const REVIEW_STATUS_LABELS: Record<string, string> = {
  generated: "Aguardando revisão",
  approved: "Aprovado",
  rejected: "Rejeitado",
  published: "Publicado",
};

function safeUrl(url?: string): string | null {
  if (!url) return null;
  try {
    const parsed = new URL(url);
    if (parsed.protocol !== "http:" && parsed.protocol !== "https:") return null;
    return parsed.toString();
  } catch {
    return null;
  }
}

function ScoreCell({ label, value }: { label: string; value?: number }) {
  const v = value ?? 0;
  return (
    <div className="rounded-md bg-muted/50 px-3 py-2">
      <p className="text-xs text-muted-foreground">{label}</p>
      <p className={cn("text-sm font-semibold", v >= 70 ? "text-green-600" : v >= 40 ? "text-yellow-600" : "text-red-600")}>
        {v.toFixed(1)}
      </p>
    </div>
  );
}

export function WorkflowReviewPage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const currentSiteId = useCurrentSiteId();
  const queryClient = useQueryClient();

  const [metaTitle, setMetaTitle] = useState("");
  const [metaDescription, setMetaDescription] = useState("");
  const [keyword, setKeyword] = useState("");
  const [rejectReason, setRejectReason] = useState("");
  const [confirmingApprove, setConfirmingApprove] = useState(false);
  const [confirmingRegenerate, setConfirmingRegenerate] = useState(false);

  const { data, isLoading, isError, refetch } = useQuery({
    queryKey: siteQueryKey(["workflow-review", id], currentSiteId),
    queryFn: () => api.get<JobReviewDetail>(`/workflow/${id}/review`),
    enabled: !!currentSiteId && !!id,
  });

  const invalidateAll = () => {
    queryClient.invalidateQueries({ queryKey: ["workflow-review"] });
    queryClient.invalidateQueries({ queryKey: ["workflow-jobs"] });
    queryClient.invalidateQueries({ queryKey: ["workflow-dashboard"] });
  };

  const saveDraftMutation = useMutation({
    mutationFn: (body: Record<string, string>) => api.put(`/workflow/${id}/review/draft`, body),
    onSuccess: () => {
      invalidateAll();
      void refetch();
    },
  });

  const approveMutation = useMutation({
    mutationFn: () => api.post(`/workflow/${id}/review/approve`, {
      meta_title: metaTitle || undefined,
      meta_description: metaDescription || undefined,
      keyword: keyword || undefined,
    }),
    onSuccess: () => {
      invalidateAll();
      navigate("/admin/workflow");
    },
  });

  const rejectMutation = useMutation({
    mutationFn: () => api.post(`/workflow/${id}/review/reject`, { reason: rejectReason }),
    onSuccess: () => {
      invalidateAll();
      void refetch();
    },
  });

  const regenerateMutation = useMutation({
    mutationFn: () => api.post(`/workflow/${id}/review/regenerate`),
    onSuccess: () => {
      invalidateAll();
      navigate("/admin/workflow");
    },
  });

  if (isLoading) {
    return <LoadingState variant="full" text="Carregando revisão do artigo..." />;
  }

  if (isError || !data) {
    return (
      <ErrorState
        title="Não foi possível carregar a revisão"
        message="Ocorreu um erro ao buscar o draft do artigo. Tente novamente."
        onRetry={refetch}
      />
    );
  }

  const job = data.job;
  const article = data.article;
  const seo = data.seo;
  const imageUrl = safeUrl(article?.featured_image_url);

  return (
    <div className="space-y-6">
      <div className="flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
        <div className="min-w-0">
          <div className="flex flex-wrap items-center gap-2">
            <h1 className="text-2xl font-semibold tracking-tight">{article?.title || job.title}</h1>
            <span className={cn("rounded px-2 py-0.5 text-xs font-medium", REVIEW_STATUS_COLORS[job.review_status] || "bg-muted text-muted-foreground")}>
              {REVIEW_STATUS_LABELS[job.review_status] || job.review_status}
            </span>
            {job.status === "failed" && <span className="rounded bg-red-50 px-2 py-0.5 text-xs font-medium text-red-700">failed</span>}
          </div>
          <p className="mt-1 text-sm text-muted-foreground">
            Revisão {data.version} · {job.status} · {job.language.toUpperCase()} ·{" "}
            {article?.word_count ? `${article.word_count} palavras · ${article.reading_time} min de leitura` : "sem conteúdo gerado"}
          </p>
          {job.rejection_reason && (
            <p className="mt-2 rounded-md bg-red-50 px-3 py-2 text-sm text-red-700">
              Motivo da rejeição: {job.rejection_reason}
            </p>
          )}
          {job.error_message && job.status === "failed" && (
            <p className="mt-2 rounded-md bg-yellow-50 px-3 py-2 text-sm text-yellow-700">
              {job.error_message}
            </p>
          )}
        </div>
        <div className="flex shrink-0 flex-wrap gap-2">
          <Button variant="outline" size="sm" onClick={() => navigate("/admin/workflow")}>
            Voltar
          </Button>
          {(job.review_status === "generated" || job.review_status === "rejected" || job.status === "failed") && (
            <Button variant="outline" size="sm" onClick={() => setConfirmingRegenerate(true)} data-testid="review-regenerate-button">
              <RefreshCw className="h-4 w-4" />
              Regenerar
            </Button>
          )}
        </div>
      </div>
      {job.review_status === "generated" && (
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground">Meta dados de publicação</CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="grid gap-4 md:grid-cols-3">
              <div className="space-y-1.5">
                <Label htmlFor="review-meta-title">Meta título</Label>
                <Input
                  id="review-meta-title"
                  value={metaTitle}
                  onChange={(e) => setMetaTitle(e.target.value)}
                  placeholder={article?.meta_title || "Meta título (opcional)"}
                  data-testid="review-meta-title"
                />
              </div>
              <div className="space-y-1.5">
                <Label htmlFor="review-meta-desc">Meta description</Label>
                <Input
                  id="review-meta-desc"
                  value={metaDescription}
                  onChange={(e) => setMetaDescription(e.target.value)}
                  placeholder={article?.meta_description || "Meta description (opcional)"}
                  data-testid="review-meta-desc"
                />
              </div>
              <div className="space-y-1.5">
                <Label htmlFor="review-keyword">Keyword</Label>
                <Input
                  id="review-keyword"
                  value={keyword}
                  onChange={(e) => setKeyword(e.target.value)}
                  placeholder={article?.keyword || "Keyword (opcional)"}
                  data-testid="review-keyword"
                />
              </div>
            </div>
            <div className="flex flex-wrap gap-2">
              <Button
                size="sm"
                variant="outline"
                onClick={() =>
                  saveDraftMutation.mutate({
                    ...(metaTitle ? { meta_title: metaTitle } : {}),
                    ...(metaDescription ? { meta_description: metaDescription } : {}),
                    ...(keyword ? { keyword } : {}),
                  })
                }
                disabled={saveDraftMutation.isPending || (!metaTitle && !metaDescription && !keyword)}
                data-testid="review-save-draft"
              >
                Salvar rascunho
              </Button>
              {!confirmingApprove ? (
                <Button size="sm" onClick={() => setConfirmingApprove(true)} data-testid="review-approve-start">
                  <Send className="h-4 w-4" />
                  Aprovar e publicar
                </Button>
              ) : (
                <div className="flex items-center gap-2">
                  <p className="text-xs text-muted-foreground">
                    Este artigo será publicado em AIWorkSimple. Confirmar?
                  </p>
                  <Button
                    size="sm"
                    onClick={() => approveMutation.mutate()}
                    disabled={approveMutation.isPending}
                    data-testid="review-approve-confirm"
                  >
                    <Check className="h-4 w-4" />
                    Confirmar publicação
                  </Button>
                  <Button size="sm" variant="ghost" onClick={() => setConfirmingApprove(false)}>
                    Cancelar
                  </Button>
                </div>
              )}
            </div>
          </CardContent>
        </Card>
      )}

      {job.review_status === "generated" && (
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground">Rejeitar artigo</CardTitle>
          </CardHeader>
          <CardContent className="space-y-3">
            <textarea
              value={rejectReason}
              onChange={(e) => setRejectReason(e.target.value)}
              placeholder="Motivo da rejeição (obrigatório) — o draft permanece salvo e pode ser regenerado."
              rows={2}
              className="w-full rounded-md border border-input bg-background px-3 py-2 text-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
              data-testid="review-reject-reason"
            />
            <Button
              size="sm"
              variant="destructive"
              disabled={rejectMutation.isPending || rejectReason.trim().length === 0}
              onClick={() => rejectMutation.mutate()}
              data-testid="review-reject-submit"
            >
              <X className="h-4 w-4" />
              Rejeitar com este motivo
            </Button>
          </CardContent>
        </Card>
      )}

      {article && (
        <div className="grid gap-6 lg:grid-cols-3">
          <div className="space-y-6 lg:col-span-2">
            <Card>
              <CardHeader className="pb-2">
                <CardTitle className="text-sm font-medium text-muted-foreground">Imagem em destaque</CardTitle>
              </CardHeader>
              <CardContent>
                {imageUrl ? (
                  <figure>
                    <img
                      src={imageUrl}
                      alt={article.featured_image_alt || article.title}
                      className="max-h-80 w-full rounded-md object-cover"
                      data-testid="review-featured-image"
                    />
                    <figcaption className="mt-2 text-xs text-muted-foreground">
                      {article.featured_image_credit?.photographer && (
                        <>
                          Foto de{" "}
                          {safeUrl(article.featured_image_credit.photographer_url) ? (
                            <a
                              href={safeUrl(article.featured_image_credit.photographer_url) ?? undefined}
                              target="_blank"
                              rel="noreferrer noopener"
                              className="text-brand-600 hover:underline"
                            >
                              {article.featured_image_credit.photographer}
                            </a>
                          ) : (
                            article.featured_image_credit.photographer
                          )}
                        </>
                      )}
                      {safeUrl(article.featured_image_credit?.source_url) && (
                        <>
                          {" "}
                          ·{" "}
                          <a
                            href={safeUrl(article.featured_image_credit?.source_url) ?? undefined}
                            target="_blank"
                            rel="noreferrer noopener"
                            className="text-brand-600 hover:underline"
                          >
                            Fonte da imagem
                          </a>
                        </>
                      )}
                    </figcaption>
                  </figure>
                ) : (
                  <div
                    className="flex items-center gap-2 rounded-md bg-muted/50 px-4 py-6 text-sm text-muted-foreground"
                    data-testid="review-no-image"
                  >
                    <ImageOff className="h-4 w-4" />
                    Nenhuma imagem em destaque disponível. O artigo será publicado sem imagem.
                  </div>
                )}
              </CardContent>
            </Card>

            <Card>
              <CardHeader className="pb-2">
                <CardTitle className="text-sm font-medium text-muted-foreground">Artigo ({article.word_count} palavras)</CardTitle>
              </CardHeader>
              <CardContent>
                <div className="whitespace-pre-wrap rounded-md bg-muted/30 p-4 text-sm leading-relaxed" data-testid="review-content">
                  {article.content}
                </div>
              </CardContent>
            </Card>
          </div>

          <div className="space-y-6">
            {seo && (
              <Card>
                <CardHeader className="pb-2">
                  <CardTitle className="flex items-center justify-between text-sm font-medium text-muted-foreground">
                    <span>SEO Score</span>
                    <span
                      data-testid="review-seo-passes"
                      className={cn(
                        "rounded px-2 py-0.5 text-xs font-semibold",
                        seo.passes ? "bg-green-50 text-green-700" : "bg-red-50 text-red-700",
                      )}
                    >
                      {seo.score.toFixed(2)} / {seo.min_score.toFixed(0)} {seo.passes ? "OK" : "BAIXO"}
                    </span>
                  </CardTitle>
                </CardHeader>
                <CardContent className="space-y-3">
                  <div className="grid grid-cols-2 gap-2">
                    <ScoreCell label="Title" value={seo.title} />
                    <ScoreCell label="Meta" value={seo.meta} />
                    <ScoreCell label="Headings" value={seo.headings} />
                    <ScoreCell label="Keyword" value={seo.keyword} />
                    <ScoreCell label="Readability" value={seo.readability} />
                    <ScoreCell label="Internal" value={seo.internal_links} />
                    <ScoreCell label="External" value={seo.external_links} />
                    <ScoreCell label="EEAT" value={seo.eeat} />
                    <ScoreCell label="Images" value={seo.images} />
                  </div>
                  {seo.keyword_density !== undefined && (
                    <p className="text-xs text-muted-foreground">
                      Densidade da keyword: {seo.keyword_density.toFixed(2)}%
                    </p>
                  )}
                  {seo.issues && seo.issues.length > 0 && (
                    <ul className="space-y-1" data-testid="review-seo-issues">
                      {seo.issues.map((issue, i) => (
                        <li key={i} className="text-xs text-muted-foreground">
                          · {issue}
                        </li>
                      ))}
                    </ul>
                  )}
                </CardContent>
              </Card>
            )}

            <Card>
              <CardHeader className="pb-2">
                <CardTitle className="text-sm font-medium text-muted-foreground">Metadados do artigo</CardTitle>
              </CardHeader>
              <CardContent className="space-y-2 text-sm">
                <div>
                  <p className="text-xs text-muted-foreground">Excerpt</p>
                  <p className="text-muted-foreground">{article.excerpt || "—"}</p>
                </div>
                <div>
                  <p className="text-xs text-muted-foreground">Categorias</p>
                  <div className="mt-1 flex flex-wrap gap-1">
                    {article.categories && article.categories.length > 0 ? (
                      article.categories.map((c) => (
                        <span key={c} className="rounded bg-muted px-2 py-0.5 text-xs">{c}</span>
                      ))
                    ) : (
                      <span className="text-xs text-muted-foreground">Nenhuma categoria</span>
                    )}
                  </div>
                </div>
                <div>
                  <p className="text-xs text-muted-foreground">Tags</p>
                  <div className="mt-1 flex flex-wrap gap-1">
                    {article.tags && article.tags.length > 0 ? (
                      article.tags.map((t) => (
                        <span key={t} className="rounded bg-muted px-2 py-0.5 text-xs">{t}</span>
                      ))
                    ) : (
                      <span className="text-xs text-muted-foreground">Sem tags</span>
                    )}
                  </div>
                </div>
                <div>
                  <p className="text-xs text-muted-foreground">Autor</p>
                  <p className="text-muted-foreground">{article.author_name || "—"}</p>
                </div>
              </CardContent>
            </Card>

            <Card>
              <CardHeader className="pb-2">
                <CardTitle className="text-sm font-medium text-muted-foreground">Fontes de pesquisa ({data.sources?.length ?? 0})</CardTitle>
              </CardHeader>
              <CardContent className="space-y-3">
                {!data.sources || data.sources.length === 0 ? (
                  <p className="text-xs text-muted-foreground">
                    Nenhuma fonte de pesquisa registrada para este tópico.
                  </p>
                ) : (
                  data.sources.slice(0, 10).map((source) => (
                    <div key={source.id} className="rounded-md bg-muted/50 px-3 py-2">
                      <a
                        href={safeUrl(source.url) ?? undefined}
                        target="_blank"
                        rel="noreferrer noopener"
                        className="flex items-center gap-1 text-xs font-medium text-brand-600 hover:underline"
                      >
                        <ExternalLink className="h-3 w-3" />
                        {source.title || source.domain || source.url}
                      </a>
                      {source.snippet && <p className="mt-1 text-xs text-muted-foreground">{source.snippet}</p>}
                      <p className="mt-1 text-xs text-muted-foreground">
                        {source.domain} · confiabilidade {source.reliability_score} ({source.reliability_label || "—"})
                        {source.is_verified ? " · verificada" : ""}
                      </p>
                    </div>
                  ))
                )}
              </CardContent>
            </Card>
          </div>
        </div>
      )}

      {!article && (
        <Card>
          <CardContent>
            <EmptyState
              title="Nenhum conteúdo gerado"
              description="Este job ainda não produziu um draft revisável. Execute o workflow primeiro."
            />
          </CardContent>
        </Card>
      )}

      {confirmingRegenerate && (
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground">Regenerar artigo</CardTitle>
          </CardHeader>
          <CardContent className="space-y-3">
            <p className="text-sm text-muted-foreground">
              O draft atual será preservado como revisão {data.version} (imutável). Uma nova revisão será criada e o job
              voltará para rascunho para re-execução.
            </p>
            <div className="flex gap-2">
              <Button
                size="sm"
                onClick={() => regenerateMutation.mutate()}
                disabled={regenerateMutation.isPending}
                data-testid="review-regenerate-confirm"
              >
                <RefreshCw className="h-4 w-4" />
                Regenerar (nova revisão {data.version + 1})
              </Button>
              <Button size="sm" variant="ghost" onClick={() => setConfirmingRegenerate(false)}>
                Cancelar
              </Button>
            </div>
          </CardContent>
        </Card>
      )}
    </div>
  );
}
