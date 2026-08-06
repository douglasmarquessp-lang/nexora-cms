package editorialbrain

import (
	"regexp"
	"strings"
)

// personaSignal is a weighted keyword pattern for a reader persona.
type personaSignal struct {
	re     *regexp.Regexp
	weight float64
}

// pre builds a leading-boundary-only pattern: plural forms ("criadores",
// "desenvolvedores") still match their singular cue ("criador").
func pre(s string) *regexp.Regexp { return regexp.MustCompile(`(?i)\b(?:` + s + `)`) }

// personaCues maps each persona to its per-language topic cues (lang "pt"/"en").
var personaCues = map[Persona]map[string][]personaSignal{
	PersonaDeveloper: {
		"pt": {
			{pre(`api|sdk|cli|gpu|llm|modelo de linguagem|framework|biblioteca|linguagem de programação|código|programação|github|docker|deploy|endpoint|webhook|automação|script`), 2.0},
			{pre(`developer|desenvolvedor|engenharia|engineering|token|latência|benchmark|integração|versão`), 2.0},
			{pre(`gpt|gemini|claude|python|javascript|typescript|node|react|kubernetes|linux|self-hosted|open source|código aberto`), 2.0},
		},
		"en": {
			{pre(`api|sdk|cli|gpu|llm|language model|framework|library|programming language|code|programming|github|docker|deploy|endpoint|webhook|automation|script`), 2.0},
			{pre(`developer|engineering|token|latency|benchmark|integration|version`), 2.0},
			{pre(`gpt|gemini|claude|python|javascript|typescript|node|react|kubernetes|linux|self-hosted|open source`), 2.0},
		},
	},
	PersonaBusiness: {
		"pt": {
			{pre(`preço|custo|custo-benefício|roi|investimento|economia|receita|empresa|negócio|market share|mercado|enterprise|corporativo|startup|pme|gestor|gerente`), 2.0},
			{pre(`orçamento|contrato|licença|assinatura|vendas|escala|produtividade|equipe|funcionários`), 2.0},
		},
		"en": {
			{pre(`price|cost|roi|investment|savings|revenue|company|business|market share|market|enterprise|corporate|startup|sme|manager|budget`), 2.0},
			{pre(`contract|license|subscription|sales|scale|productivity|team|employees`), 2.0},
		},
	},
	PersonaCreator: {
		"pt": {
			{pre(`criador|conteúdo|audiência|seguidores|monetização|engajamento|storytelling|copy|copywriting`), 2.0},
			{pre(`youtube|instagram|tiktok|canal|vídeo|streamer|influencer|thumbnail|edição`), 2.0},
			{pre(`blog|dropshipping|marketing|design|produtor`), 2.0},
		},
		"en": {
			{pre(`creator|content|audience|followers|monetization|engagement|storytelling|copy|copywriting`), 2.0},
			{pre(`youtube|instagram|tiktok|channel|video|streamer|influencer|thumbnail|editing`), 2.0},
			{pre(`blog|dropshipping|marketing|design|producer`), 2.0},
		},
	},
}

// personaTieBreak defines precedence when personas tie.
var personaTieBreak = []Persona{PersonaDeveloper, PersonaBusiness, PersonaCreator, PersonaGeneral}

// DetectPersona deterministically infers the most likely reader persona
// from the topic. The article changes accordingly (angle, audience, outline).
func DetectPersona(topic, language string) PersonaResult {
	topic = strings.TrimSpace(topic)
	if topic == "" {
		return PersonaResult{Persona: PersonaGeneral, Confidence: 0.5, Audience: PersonaGeneral.AudienceLabel(language), Language: language}
	}

	scores := make(map[Persona]float64)
	reasons := make(map[Persona][]string)
	lang := "en"
	if language == "pt" {
		lang = "pt"
	}
	for persona, langs := range personaCues {
		for _, c := range langs[lang] {
			if c.re.MatchString(topic) {
				scores[persona] += c.weight
				reasons[persona] = append(reasons[persona], c.re.String())
			}
		}
	}

	winner := PersonaGeneral
	winnerScore := 0.0
	runnerScore := 0.0
	for _, p := range personaTieBreak {
		s := scores[p]
		if s > winnerScore {
			runnerScore = winnerScore
			winner, winnerScore = p, s
		} else if s > runnerScore {
			runnerScore = s
		}
	}

	confidence := 0.5
	if winnerScore > 0 {
		if runnerScore > 0 {
			confidence = winnerScore / (winnerScore + runnerScore)
		} else {
			confidence = 0.8
		}
	}

	return PersonaResult{
		Persona:    winner,
		Confidence: clampScore(confidence),
		Audience:   winner.AudienceLabel(language),
		Reasons:    sortedSignalSet(reasons[winner]),
		Language:   language,
	}
}
