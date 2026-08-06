export type PipelineStage =
  | "idea"
  | "research"
  | "outline"
  | "writing"
  | "seo"
  | "eeat"
  | "translation"
  | "review"
  | "approval"
  | "scheduled"
  | "published";

export const PIPELINE_STAGES: PipelineStage[] = [
  "idea",
  "research",
  "outline",
  "writing",
  "seo",
  "eeat",
  "translation",
  "review",
  "approval",
  "scheduled",
  "published",
];

export const STAGE_LABELS: Record<PipelineStage, string> = {
  idea: "Ideia",
  research: "Pesquisa",
  outline: "Roteiro",
  writing: "Escrita",
  seo: "SEO",
  eeat: "E-E-A-T",
  translation: "Tradução",
  review: "Revisão",
  approval: "Aprovação",
  scheduled: "Agendado",
  published: "Publicado",
};

export const STAGE_COLORS: Record<PipelineStage, { dot: string; card: string }> = {
  idea: { dot: "bg-slate-400", card: "border-slate-300" },
  research: { dot: "bg-amber-400", card: "border-amber-300" },
  outline: { dot: "bg-amber-500", card: "border-amber-400" },
  writing: { dot: "bg-blue-500", card: "border-blue-300" },
  seo: { dot: "bg-violet-500", card: "border-violet-300" },
  eeat: { dot: "bg-violet-400", card: "border-violet-300" },
  translation: { dot: "bg-orange-500", card: "border-orange-300" },
  review: { dot: "bg-green-500", card: "border-green-300" },
  approval: { dot: "bg-green-600", card: "border-green-400" },
  scheduled: { dot: "bg-slate-900", card: "border-slate-700" },
  published: { dot: "bg-emerald-500", card: "border-emerald-300" },
};

export interface PipelineItem {
  id: string;
  title: string;
  slug?: string;
  stage: PipelineStage;
  engine: string;
  engine_id: string;
  language?: string;
  category_id?: string | null;
  author_id?: string | null;
  seo_score?: number | null;
  eeat_score?: number | null;
  status: string;
  scheduled_at?: string | null;
  updated_at: string;
  actionable: boolean;
}

export interface StageCount {
  stage: PipelineStage;
  count: number;
}

export interface PipelineResponse {
  items: PipelineItem[];
  total: number;
  stages: StageCount[];
}

export interface PipelineStats {
  stage_counts: StageCount[];
  total_items: number;
  avg_seo_score?: number | null;
  avg_eeat_score?: number | null;
  pending_reviews: number;
  pending_approvals: number;
  in_translation: number;
  published_this_week: number;
}

export interface ReviewPost {
  id: string;
  title: string;
  slug: string;
  status: string;
  language?: string;
  seo_score?: number | null;
  seo_analyzed_at?: string | null;
  updated_at: string;
}

export interface ReviewScores {
  seo: number;
  eeat: number;
  freshness: number;
  coverage: number;
  naturalness: number;
  confidence: number;
  final: number;
  decision: string;
  threshold: number;
  created_at: string;
}

export interface ReviewSource {
  url: string;
  title?: string;
  snippet?: string;
  language?: string;
  is_verified: boolean;
  freshness_score?: number | null;
  relevance_score?: number;
  retrieved_at?: string;
}

export interface ReviewLink {
  title: string;
  url: string;
  anchor_text?: string;
  score: number;
  label?: string;
  reliability?: number;
}

export interface ReviewProblem {
  kind: string;
  message: string;
  severity: string;
}

export interface ReviewRecommendation {
  label: string;
  score?: number | null;
  status: string;
  details?: string[];
}

export interface ReadinessCheck {
  stage: string;
  label: string;
  passed: boolean;
  message: string;
}

export interface PublishReadiness {
  post_id: string;
  title: string;
  slug: string;
  ready: boolean;
  blocking?: string;
  checks: ReadinessCheck[];
}

export interface ArticleReview {
  post: ReviewPost;
  review?: ReviewScores;
  sources: ReviewSource[];
  internal_links: ReviewLink[];
  external_links: ReviewLink[];
  problems: ReviewProblem[];
  recommendations: ReviewRecommendation[];
  readiness?: PublishReadiness;
}

export interface ApprovalRequest {
  id: string;
  site_id: string;
  post_id: string;
  requested_by: string;
  status: "pending" | "approved" | "rejected";
  comments?: string;
  reviewed_by?: string | null;
  reviewed_at?: string | null;
  created_at: string;
  updated_at: string;
}
