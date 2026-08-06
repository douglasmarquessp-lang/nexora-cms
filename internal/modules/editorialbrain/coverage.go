package editorialbrain

import "strings"

// facetDef defines a coverage facet with bilingual name and markers.
type facetDef struct {
	id      FacetID
	name    bi
	markers []string
}

// facetOrder is the fixed order of coverage facets.
var facetOrder = []FacetID{
	FacetPrice, FacetLimitations, FacetAPI, FacetUseCases,
	FacetAlternatives, FacetInstallation, FacetRequirements, FacetComparison,
}

var facetDefs = map[FacetID]facetDef{
	FacetPrice:        {FacetPrice, bi{pt: "Preço", en: "Price"}, []string{"r$", "us$", "usd", "eur", "preço", "preco", "preços", "precos", "price", "cost", "custa", "assinatura", "planos", "pricing"}},
	FacetLimitations:  {FacetLimitations, bi{pt: "Limitações", en: "Limitations"}, []string{"limitaç", "limitac", "limitation", "desvantag", "drawback", "não pode", "nao pode", "cannot", "não suporta", "nao suporta", "does not support", "restriç", "restric"}},
	FacetAPI:          {FacetAPI, bi{pt: "API", en: "API"}, []string{"api", "endpoint", "sdk", "webhook", "integraç", "integrac", "integration", "documentação", "documentacao", "documentation", "developer"}},
	FacetUseCases:     {FacetUseCases, bi{pt: "Casos de uso", en: "Use cases"}, []string{"caso de uso", "casos de uso", "use case", "use cases", "exemplo real", "exemplo prático", "example", "cenário", "cenario", "scenario", "aplicação", "aplicacao"}},
	FacetAlternatives: {FacetAlternatives, bi{pt: "Alternativas", en: "Alternatives"}, []string{"alternativ", "opções", "opcoes", "options", "similar", "similares", "concorrent", "competitor", "em vez de", "instead of", " vs "}},
	FacetInstallation: {FacetInstallation, bi{pt: "Instalação", en: "Installation"}, []string{"instalar", "instalaç", "instalac", "installation", "install", "configurar", "configure", "setup", "implantar", "deploy", "iniciar", "começar"}},
	FacetRequirements: {FacetRequirements, bi{pt: "Requisitos", en: "Requirements"}, []string{"requisito", "requirement", "requer", "requires", "necessário", "necessario", "precisa de", "precisar", "compatível", "compativel", "compatible"}},
	FacetComparison:   {FacetComparison, bi{pt: "Comparação", en: "Comparison"}, []string{"compar", "versus", " vs ", "melhor que", "better than", "diferença", "diferenca", "difference", "benchmark"}},
}

// CheckCoverage deterministically measures how much of the subject the
// article explains and lists exactly what is missing (bilingual).
func CheckCoverage(text string, language string) CoverageReport {
	lower := strings.ToLower(text)
	covered := make([]FacetID, 0, len(facetOrder))
	missing := make([]FacetIssue, 0)
	for _, id := range facetOrder {
		def := facetDefs[id]
		found := false
		for _, m := range def.markers {
			if strings.Contains(lower, m) {
				found = true
				break
			}
		}
		if found {
			covered = append(covered, id)
		} else {
			missing = append(missing, FacetIssue{
				Facet:   id,
				Message: b("Faltou explicar: %s", "Missing: %s").text(language) + " " + def.name.text(language),
			})
		}
	}
	total := len(facetOrder)
	percent := 0.0
	if total > 0 {
		percent = clampScore(float64(len(covered)) / float64(total) * 100)
	}
	return CoverageReport{
		CoveragePercent: percent,
		Covered:         covered,
		Missing:         missing,
		TotalFacets:     total,
	}
}
