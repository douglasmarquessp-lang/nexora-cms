package editorialbrain

import (
	"context"
	"testing"
	"time"
)

func TestClassifyIntent_NewsPT(t *testing.T) {
	ir := ClassifyIntent("Gemini 3 lançado hoje: anúncio oficial", "pt")
	if ir.Intent != IntentBreakingNews {
		t.Errorf("expected breaking_news, got %s", ir.Intent)
	}
	if ir.Confidence < 0.5 || ir.Confidence > 1 {
		t.Errorf("confidence out of range: %v", ir.Confidence)
	}
	if !ir.VersionHint {
		t.Error("expected version hint for versioned topic")
	}
}

func TestClassifyIntent_NewsEN(t *testing.T) {
	ir := ClassifyIntent("OpenAI just announced a new model", "en")
	if ir.Intent != IntentBreakingNews {
		t.Errorf("expected breaking_news, got %s", ir.Intent)
	}
}

func TestClassifyIntent_Tutorial(t *testing.T) {
	ir := ClassifyIntent("como criar um site passo a passo", "pt")
	if ir.Intent != IntentTutorial {
		t.Errorf("expected tutorial, got %s", ir.Intent)
	}
	ir = ClassifyIntent("how to install linux step by step", "en")
	if ir.Intent != IntentTutorial {
		t.Errorf("expected tutorial, got %s", ir.Intent)
	}
}

func TestClassifyIntent_Comparison(t *testing.T) {
	ir := ClassifyIntent("React vs Vue: comparação completa", "pt")
	if ir.Intent != IntentComparison {
		t.Errorf("expected comparison, got %s", ir.Intent)
	}
	ir = ClassifyIntent("gemini vs gpt-6 comparison", "en")
	if ir.Intent != IntentComparison {
		t.Errorf("expected comparison, got %s", ir.Intent)
	}
}

func TestClassifyIntent_Commercial(t *testing.T) {
	ir := ClassifyIntent("quanto custa a assinatura do ChatGPT", "pt")
	if ir.Intent != IntentCommercial {
		t.Errorf("expected commercial, got %s", ir.Intent)
	}
	ir = ClassifyIntent("chatgpt pricing plans worth it", "en")
	if ir.Intent != IntentCommercial {
		t.Errorf("expected commercial, got %s", ir.Intent)
	}
}

func TestClassifyIntent_Navigational(t *testing.T) {
	ir := ClassifyIntent("login gemini site oficial", "pt")
	if ir.Intent != IntentNavigational {
		t.Errorf("expected navigational, got %s", ir.Intent)
	}
}

func TestClassifyIntent_UpdateWithVersionBoost(t *testing.T) {
	ir := ClassifyIntent("GPT-6: o que mudou na nova versão", "pt")
	if ir.Intent != IntentUpdate {
		t.Errorf("expected update, got %s", ir.Intent)
	}
	if !ir.VersionHint {
		t.Error("expected version hint")
	}
}

func TestClassifyIntent_FallbackInformational(t *testing.T) {
	ir := ClassifyIntent("história da inteligência artificial", "pt")
	if ir.Intent != IntentInformational {
		t.Errorf("expected informational fallback, got %s", ir.Intent)
	}
	ir = ClassifyIntent("", "pt")
	if ir.Intent != IntentInformational {
		t.Errorf("expected informational for empty topic, got %s", ir.Intent)
	}
	if ir.Confidence != 0.5 {
		t.Errorf("expected 0.5 confidence for empty topic, got %v", ir.Confidence)
	}
}

func TestClassifyIntent_Deterministic(t *testing.T) {
	a := ClassifyIntent("gemini vs gpt-6 comparison", "en")
	b := ClassifyIntent("gemini vs gpt-6 comparison", "en")
	if a.Intent != b.Intent || a.Confidence != b.Confidence || len(a.Signals) != len(b.Signals) {
		t.Error("classifier must be deterministic")
	}
}

func TestClassifyIntent_TieBreak(t *testing.T) {
	ir := ClassifyIntent("gemini vs gpt-6: comparação e preço e comprar", "pt")
	// comparison (vs 1.8 + comparação 1.8) = commercial (preço 1.8 + comprar 1.8):
	// true tie → comparison wins by precedence
	if ir.Intent != IntentComparison {
		t.Errorf("expected comparison on tie, got %s", ir.Intent)
	}
	if ir.Confidence != 0.5 {
		t.Errorf("expected 0.5 confidence on tie, got %v", ir.Confidence)
	}
}

func TestDetectPersona_Developer(t *testing.T) {
	pr := DetectPersona("GPT-6 API para desenvolvedores: SDK e endpoints", "pt")
	if pr.Persona != PersonaDeveloper {
		t.Errorf("expected developer, got %s", pr.Persona)
	}
	pr = DetectPersona("gpt-6 api sdk for developers", "en")
	if pr.Persona != PersonaDeveloper {
		t.Errorf("expected developer, got %s", pr.Persona)
	}
}

func TestDetectPersona_Business(t *testing.T) {
	pr := DetectPersona("preço do plano enterprise e ROI para empresas", "pt")
	if pr.Persona != PersonaBusiness {
		t.Errorf("expected business, got %s", pr.Persona)
	}
}

func TestDetectPersona_Creator(t *testing.T) {
	pr := DetectPersona("gemini para criadores de conteúdo no youtube", "pt")
	if pr.Persona != PersonaCreator {
		t.Errorf("expected creator, got %s", pr.Persona)
	}
}

func TestDetectPersona_GeneralFallback(t *testing.T) {
	pr := DetectPersona("como a internet funciona", "pt")
	if pr.Persona != PersonaGeneral {
		t.Errorf("expected general, got %s", pr.Persona)
	}
	if pr.Audience == "" {
		t.Error("expected audience label")
	}
}

func TestDetectPersona_Deterministic(t *testing.T) {
	a := DetectPersona("GPT-6 API para desenvolvedores", "pt")
	b := DetectPersona("GPT-6 API para desenvolvedores", "pt")
	if a.Persona != b.Persona || a.Confidence != b.Confidence {
		t.Error("persona detector must be deterministic")
	}
}

func TestGenerateOutline_Tutorial(t *testing.T) {
	o := GenerateOutline("instalar linux", "pt", IntentTutorial, PersonaDeveloper)
	if o.SuggestedTitle == "" {
		t.Error("expected suggested title")
	}
	found := map[SectionType]bool{}
	for _, s := range o.Sections {
		found[s.Type] = true
		if s.Title == "" || s.Purpose == "" {
			t.Errorf("section %s missing title/purpose", s.Type)
		}
	}
	if !found[SecIntro] || !found[SecSteps] || !found[SecFAQ] || !found[SecConclusion] {
		t.Errorf("tutorial outline missing key sections: %v", found)
	}
	if len(o.FAQs) == 0 {
		t.Error("expected FAQs")
	}
	if !o.Callouts {
		t.Error("expected callouts for tutorial")
	}
}

func TestGenerateOutline_Comparison(t *testing.T) {
	o := GenerateOutline("react vs vue", "en", IntentComparison, PersonaGeneral)
	found := map[SectionType]bool{}
	for _, s := range o.Sections {
		found[s.Type] = true
	}
	if !found[SecComparison] || !found[SecTable] || !found[SecAlternates] {
		t.Errorf("comparison outline missing sections: %v", found)
	}
	if !o.NeedsTable || !o.Comparisons {
		t.Error("expected needs_table and comparisons flags")
	}
}

func TestGenerateOutline_Ordering(t *testing.T) {
	o := GenerateOutline("x", "pt", IntentInformational, PersonaGeneral)
	prev := 0
	for _, s := range o.Sections {
		if s.Order != prev+1 {
			t.Errorf("sections must be contiguous, got %d after %d", s.Order, prev)
		}
		prev = s.Order
	}
}

func TestGenerateOutline_Deterministic(t *testing.T) {
	a := GenerateOutline("t", "pt", IntentTutorial, PersonaDeveloper)
	b := GenerateOutline("t", "pt", IntentTutorial, PersonaDeveloper)
	if len(a.Sections) != len(b.Sections) {
		t.Error("outline must be deterministic")
	}
	for i := range a.Sections {
		if a.Sections[i].Title != b.Sections[i].Title || a.Sections[i].Order != b.Sections[i].Order {
			t.Error("outline sections must match")
		}
	}
}

func TestGenerateQuestions_ByIntent(t *testing.T) {
	qs := GenerateQuestions("gpt-6", "pt", IntentCommercial)
	ids := map[QuestionID]bool{}
	for _, q := range qs {
		ids[q.ID] = true
	}
	if !ids[QCost] || !ids[QWorthIt] || !ids[QAlternates] {
		t.Errorf("commercial questions missing cost/worth/alternatives: %v", ids)
	}
}

func TestGenerateQuestions_InterpolatesTopic(t *testing.T) {
	qs := GenerateQuestions("GPT-6", "pt", IntentInformational)
	for _, q := range qs {
		if q.Question == "" {
			t.Error("question text must not be empty")
		}
	}
	found := false
	for _, q := range qs {
		if q.ID == QWhatIs && len(q.Question) > 0 {
			found = true
		}
	}
	if !found {
		t.Error("expected what_is question")
	}
}

func TestCheckQuestions_Answered(t *testing.T) {
	text := "O GPT-6 é um modelo de linguagem. Ele funciona por tokens. Custa R$ 40 por mês. Vale a pena para empresas. As limitações são claras. Quem pode usar são desenvolvedores. Existem alternativas como o Claude."
	qs := GenerateQuestions("gpt-6", "pt", IntentInformational)
	check := CheckQuestions(text, qs)
	if check.AnsweredCount == 0 {
		t.Error("expected some answered questions")
	}
	if check.Total != len(qs) {
		t.Errorf("total mismatch: %d != %d", check.Total, len(qs))
	}
	if check.AnsweredPercent <= 0 {
		t.Errorf("expected positive percent, got %v", check.AnsweredPercent)
	}
}

func TestCheckQuestions_NoneAnswered(t *testing.T) {
	text := "Aqui não tem resposta nenhuma."
	qs := GenerateQuestions("gpt-6", "pt", IntentInformational)
	check := CheckQuestions(text, qs)
	if check.AnsweredCount != 0 {
		t.Errorf("expected 0 answered, got %d", check.AnsweredCount)
	}
	if check.AnsweredPercent != 0 {
		t.Errorf("expected 0 percent, got %v", check.AnsweredPercent)
	}
}

func TestCheckCoverage_Full(t *testing.T) {
	text := "O produto custa R$ 99. As limitações são poucas. A API é completa. Casos de uso reais incluem educação. Alternativas como o X existem. A instalação leva 5 minutos. Requisitos mínimos: 8GB. A comparação com o concorrente está abaixo."
	cov := CheckCoverage(text, "pt")
	if cov.CoveragePercent != 100 {
		t.Errorf("expected 100%% coverage, got %v", cov.CoveragePercent)
	}
	if len(cov.Missing) != 0 {
		t.Errorf("expected no missing, got %v", cov.Missing)
	}
}

func TestCheckCoverage_Partial(t *testing.T) {
	text := "O produto é ótimo e muito usado."
	cov := CheckCoverage(text, "pt")
	if cov.CoveragePercent != 0 {
		t.Errorf("expected 0 coverage, got %v", cov.CoveragePercent)
	}
	if len(cov.Missing) != cov.TotalFacets {
		t.Errorf("expected all missing, got %d/%d", len(cov.Missing), cov.TotalFacets)
	}
	for _, m := range cov.Missing {
		if m.Message == "" {
			t.Error("missing facet must have bilingual message")
		}
	}
}

func TestCheckCoverage_Deterministic(t *testing.T) {
	text := "custa R$ 10, limitações aqui, api aqui"
	a := CheckCoverage(text, "pt")
	b := CheckCoverage(text, "pt")
	if a.CoveragePercent != b.CoveragePercent || len(a.Covered) != len(b.Covered) {
		t.Error("coverage must be deterministic")
	}
}

func TestCheckFluency_Clean(t *testing.T) {
	text := "Este é um parágrafo curto e claro. O texto explica tudo. Nada se repete aqui. Cada frase traz uma ideia nova. O leitor não se cansa. O vocabulário varia bastante."
	rep := CheckFluency(context.Background(), text, "pt", nil)
	if rep.OverallScore < 70 {
		t.Errorf("expected high fluency, got %v", rep.OverallScore)
	}
	if rep.RepeatedSentences != 0 {
		t.Errorf("expected no repeated sentences, got %d", rep.RepeatedSentences)
	}
}

func TestCheckFluency_RepeatedSentences(t *testing.T) {
	text := "O produto é muito bom para uso geral. O produto é muito bom para uso geral. O produto é muito bom para uso geral. Outra frase totalmente diferente aqui."
	rep := CheckFluency(context.Background(), text, "pt", nil)
	if rep.RepeatedSentences == 0 {
		t.Error("expected repeated sentences detected")
	}
	if rep.SentenceRepetition >= rep.OverallScore {
		t.Errorf("repetition penalty must hurt overall score: rep=%v overall=%v", rep.SentenceRepetition, rep.OverallScore)
	}
}

func TestCheckFluency_PassiveVoice(t *testing.T) {
	text := "O relatório foi escrito pela equipe. A decisão foi tomada ontem. O código foi revisado. O sistema foi atualizado."
	rep := CheckFluency(context.Background(), text, "pt", nil)
	if rep.PassiveCount == 0 {
		t.Error("expected passive voice detected")
	}
	if rep.PassiveVoice >= 90 {
		t.Errorf("passive penalty missing: %v", rep.PassiveVoice)
	}
}

func TestCheckFluency_LongParagraph(t *testing.T) {
	words := []string{}
	for i := 0; i < 200; i++ {
		words = append(words, "palavra")
	}
	text := "Primeira frase curta.\n\n" + joinWords(words)
	rep := CheckFluency(context.Background(), text, "pt", nil)
	if rep.LongParagraphs == 0 {
		t.Error("expected long paragraph detected")
	}
	if rep.MaxParagraphWords < 190 {
		t.Errorf("expected max paragraph words ~200, got %d", rep.MaxParagraphWords)
	}
}

func TestCheckFluency_Deterministic(t *testing.T) {
	text := "O produto custa dez reais e funciona bem. A instalação é rápida. O suporte é bom."
	a := CheckFluency(context.Background(), text, "pt", nil)
	b := CheckFluency(context.Background(), text, "pt", nil)
	if a.OverallScore != b.OverallScore || a.ReadabilityScore != b.ReadabilityScore {
		t.Error("fluency must be deterministic")
	}
}

func TestLinkEvidence_VerifiedWithFacts(t *testing.T) {
	facts := []FactEntry{
		{FactType: "price", Entity: "gpt-6", Value: "custa 40 dólares por mês", SourceURL: "https://openai.com/pricing", Confidence: 90},
	}
	sources := []SourceRef{
		{Title: "OpenAI pricing", URL: "https://openai.com/pricing", Snippet: "gpt-6 custa 40 dólares por mês", ReliabilityScore: 90},
	}
	text := "O GPT-6 custa 40 dólares por mês. Outra frase sem números aqui."
	ev := LinkEvidence(text, facts, sources, "pt")
	if ev.ClaimsCount == 0 {
		t.Fatal("expected claims extracted")
	}
	if ev.VerifiedCount == 0 {
		t.Error("expected verified claim with matching facts")
	}
	if ev.EvidenceScore < 90 {
		t.Errorf("expected high evidence score, got %v", ev.EvidenceScore)
	}
	for _, l := range ev.Links {
		if l.Verified {
			if l.SourceURL == "" {
				t.Error("verified claim must carry source url")
			}
			if l.Confidence < 70 {
				t.Errorf("verified claim confidence too low: %v", l.Confidence)
			}
		}
	}
}

func TestLinkEvidence_UnverifiedWithoutFacts(t *testing.T) {
	text := "O produto custa 40 dólares por mês e é o melhor do mercado."
	ev := LinkEvidence(text, nil, nil, "pt")
	if ev.ClaimsCount == 0 {
		t.Fatal("expected claims extracted")
	}
	if ev.VerifiedCount != 0 {
		t.Errorf("expected 0 verified without facts, got %d", ev.VerifiedCount)
	}
	if ev.EvidenceScore >= 70 {
		t.Errorf("expected low evidence score without facts, got %v", ev.EvidenceScore)
	}
}

func TestLinkEvidence_OfficialSourceNote(t *testing.T) {
	facts := []FactEntry{
		{FactType: "price", Entity: "gpt-6", Value: "custa 40 dólares", SourceURL: "https://openai.com/pricing", Confidence: 95},
	}
	text := "O GPT-6 custa 40 dólares."
	ev := LinkEvidence(text, facts, nil, "pt")
	for _, l := range ev.Links {
		if l.Verified && l.Note == "" {
			t.Error("verified claim must have a note")
		}
	}
}

func TestLinkEvidence_Deterministic(t *testing.T) {
	facts := []FactEntry{{FactType: "price", Entity: "gpt-6", Value: "custa 40 dólares", Confidence: 90}}
	text := "O GPT-6 custa 40 dólares por mês."
	a := LinkEvidence(text, facts, nil, "pt")
	b := LinkEvidence(text, facts, nil, "pt")
	if a.EvidenceScore != b.EvidenceScore || a.ClaimsCount != b.ClaimsCount || a.VerifiedCount != b.VerifiedCount {
		t.Error("evidence must be deterministic")
	}
}

func TestScoreBlocks_HeadingSplit(t *testing.T) {
	text := "Introdução ao assunto. Primeira frase.\n\n## Preço\n\nCusta 40 dólares.\n\n## Conclusão\n\nResumo final."
	blocks := splitBlocks(text, "pt")
	if len(blocks) == 0 {
		t.Fatal("expected blocks")
	}
	if blocks[0].name != "Introdução" {
		t.Errorf("expected first block named Introdução, got %q", blocks[0].name)
	}
	foundPrice := false
	for _, b := range blocks {
		if b.name == "Preço" {
			foundPrice = true
		}
	}
	if !foundPrice {
		t.Error("expected Preço block")
	}
}

func TestScoreBlocks_EvidenceDensity(t *testing.T) {
	facts := []FactEntry{{Entity: "gpt-6", Value: "custa 40 dólares", Confidence: 90}}
	text := "## Preço\n\nO GPT-6 custa 40 dólares. Preço oficial.\n\n## Histórico\n\nO produto foi criado em 2020 e evoluiu."
	ev := LinkEvidence(text, facts, nil, "pt")
	scores := ScoreBlocks(text, ev, "pt")
	byName := map[string]float64{}
	for _, b := range scores {
		byName[b.Block] = b.Score
	}
	if len(byName) == 0 {
		t.Fatal("expected block scores")
	}
}

func TestScoreBlocks_Deterministic(t *testing.T) {
	text := "## A\n\nFrase com número 42 aqui.\n\n## B\n\nOutra frase."
	ev := LinkEvidence(text, nil, nil, "pt")
	a := ScoreBlocks(text, ev, "pt")
	b := ScoreBlocks(text, ev, "pt")
	if len(a) != len(b) {
		t.Error("block scores must be deterministic")
	}
	for i := range a {
		if a[i].Score != b[i].Score {
			t.Error("block scores must match")
		}
	}
}

func TestCheckSemantic_Entities(t *testing.T) {
	text := "O GPT-6 e o Gemini foram analisados. A API é documentada."
	rep := CheckSemantic("GPT-6", text, "pt", []string{"GPT-6"}, []string{"API"}, 100)
	if len(rep.EntitiesFound) != 1 {
		t.Errorf("expected entity found, got %v", rep.EntitiesFound)
	}
	if len(rep.ConceptsMissing) != 0 {
		t.Errorf("expected concept found, got missing %v", rep.ConceptsMissing)
	}
}

func TestCheckSemantic_MissingTerms(t *testing.T) {
	text := "Texto curto."
	rep := CheckSemantic("inteligência artificial", text, "pt", nil, nil, 0)
	if len(rep.MissingTerms) == 0 {
		t.Error("expected missing terms")
	}
	if len(rep.Issues) == 0 {
		t.Error("expected semantic issues")
	}
	if rep.SemanticScore >= 80 {
		t.Errorf("expected low semantic score, got %v", rep.SemanticScore)
	}
}

func TestCheckSemantic_Deterministic(t *testing.T) {
	text := "O GPT-6 é um modelo. A API existe."
	a := CheckSemantic("GPT-6", text, "pt", []string{"GPT-6"}, []string{"API"}, 100)
	b := CheckSemantic("GPT-6", text, "pt", []string{"GPT-6"}, []string{"API"}, 100)
	if a.SemanticScore != b.SemanticScore || len(a.Issues) != len(b.Issues) {
		t.Error("semantic must be deterministic")
	}
}

func TestComputeEditorialScore_Weights(t *testing.T) {
	s := ComputeEditorialScore(96, 94, 98, 95, 97, 99, 90)
	if s.Decision != DecisionApproved {
		t.Errorf("expected approved, got %s", s.Decision)
	}
	if s.Final < 95 || s.Final > 97 {
		t.Errorf("final out of expected range: %v", s.Final)
	}
}

func TestComputeEditorialScore_BelowThreshold(t *testing.T) {
	s := ComputeEditorialScore(50, 50, 50, 50, 50, 50, 90)
	if s.Decision != DecisionNeedsReview {
		t.Errorf("expected needs_review, got %s", s.Decision)
	}
	if s.Final >= 90 {
		t.Errorf("expected final below 90, got %v", s.Final)
	}
}

func TestComputeEditorialScore_ThresholdDefault(t *testing.T) {
	s := ComputeEditorialScore(100, 100, 100, 100, 100, 100, 0)
	if s.Threshold != DefaultMinFinalScore {
		t.Errorf("expected default threshold %v, got %v", DefaultMinFinalScore, s.Threshold)
	}
	if s.Decision != DecisionApproved {
		t.Errorf("expected approved at 100, got %s", s.Decision)
	}
}

func TestComputeEditorialScore_Deterministic(t *testing.T) {
	a := ComputeEditorialScore(80, 80, 80, 80, 80, 80, 90)
	b := ComputeEditorialScore(80, 80, 80, 80, 80, 80, 90)
	if a.Final != b.Final || a.Decision != b.Decision {
		t.Error("score must be deterministic")
	}
}

func TestSourceFreshnessScore(t *testing.T) {
	if got := SourceFreshnessScore(nil, nil); got != 60 {
		t.Errorf("expected 60 for nil date, got %v", got)
	}
	now := time.Now()
	day := 24 * time.Hour
	if got := SourceFreshnessScore(&now, nil); got != 100 {
		t.Errorf("expected 100 for now, got %v", got)
	}
	old := now.Add(-120 * day)
	if got := SourceFreshnessScore(&old, nil); got != 50 {
		t.Errorf("expected 50 for very old, got %v", got)
	}
	recent := now.Add(-3 * day)
	if got := SourceFreshnessScore(&recent, nil); got != 95 {
		t.Errorf("expected 95 for 3 days, got %v", got)
	}
}

func TestSourcesFreshnessScore(t *testing.T) {
	if got := SourcesFreshnessScore(nil); got != 60 {
		t.Errorf("expected neutral 60 without sources, got %v", got)
	}
	now := time.Now()
	sources := []SourceRef{{Title: "a", PublishedAt: &now}, {Title: "b"}}
	if got := SourcesFreshnessScore(sources); got != 80 {
		t.Errorf("expected avg 80 (100+60)/2, got %v", got)
	}
}

func TestContentHashStable(t *testing.T) {
	a := contentHash("Título", "Conteúdo")
	b := contentHash("título", "conteúdo")
	if a != b {
		t.Error("content hash must be case-insensitive stable")
	}
	if len(a) != 16 {
		t.Errorf("expected 16 hex chars, got %d", len(a))
	}
	if contentHash("a", "b") == contentHash("a", "c") {
		t.Error("different content must produce different hashes")
	}
}

func TestTopicHash(t *testing.T) {
	if topicHash("GPT-6") != topicHash("gpt-6") {
		t.Error("topic hash must be case-insensitive")
	}
	if len(topicHash("x")) != 16 {
		t.Error("expected 16 hex chars")
	}
}

func TestTermOverlap(t *testing.T) {
	a := tokenSet("o gato preto corre")
	b := tokenSet("o gato preto dorme")
	if termOverlap(a, b) <= 0 {
		t.Error("expected positive overlap")
	}
	if termOverlap(a, tokenSet("xyz abc")) != 0 {
		t.Error("expected zero overlap for disjoint sets")
	}
	if termOverlap(tokenSet(""), b) != 0 {
		t.Error("expected 0 for empty set")
	}
}

func TestTokenizeAndSentences(t *testing.T) {
	toks := tokenize("Olá, mundo! GPT-6 é ótimo.")
	if len(toks) == 0 {
		t.Error("expected tokens")
	}
	sents := sentences("Primeira frase. Segunda frase!")
	if len(sents) != 2 {
		t.Errorf("expected 2 sentences, got %d", len(sents))
	}
}

func TestSignificantTokens(t *testing.T) {
	toks := significantTokens("o gato e o cachorro correm")
	for _, tok := range toks {
		if stopWords[tok] {
			t.Errorf("stopword leaked: %s", tok)
		}
	}
}

func TestIsNumeric(t *testing.T) {
	if !isNumeric("42") || isNumeric("42a") || isNumeric("") {
		t.Error("isNumeric misbehaves")
	}
}

func TestIsImportantClaim(t *testing.T) {
	if !isImportantClaim("O custo é 42 reais.") {
		t.Error("claim with number must be important")
	}
	if !isImportantClaim("A empresa lançou a versão nova.") {
		t.Error("claim with claim word must be important")
	}
	if isImportantClaim("Olá mundo.") {
		t.Error("short plain sentence must not be important")
	}
}

func joinWords(words []string) string {
	out := ""
	for i, w := range words {
		if i > 0 {
			out += " "
		}
		out += w
	}
	return out
}
