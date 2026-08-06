package freshness

import (
	"regexp"
	"strings"
)

// signalRule is one keyword/regex cue for an intent, with a deterministic weight.
type signalRule struct {
	re     *regexp.Regexp
	name   string
	weight float64
}

// intentSignals returns the deterministic cue table for a language.
// Every rule matches case-insensitively; accents are allowed in Go RE2.
func intentSignals(lang string) map[IntentType][]signalRule {
	pt := func(name, pattern string, weight float64) signalRule {
		return signalRule{re: regexp.MustCompile("(?i)" + pattern), name: name, weight: weight}
	}
	switch lang {
	case "pt":
		return map[IntentType][]signalRule{
			IntentNews: {
				pt("noticia", `\b(not[ií]cia|not[ií]cias)\b`, 2.0),
				pt("breaking", `\b(breaking|em tempo real|fato novo)\b`, 2.0),
				pt("anuncio", `\b(anuncia|anunciou|lança|lançou|revela|revelou)\b`, 1.8),
				pt("agora", `\b(hoje|ontem|agora|esta semana|essa semana)\b`, 1.5),
				pt("governo", `\b(governo|presidente|eleiç[ãa]o|ministro|minist[eé]rio)\b`, 1.2),
				pt("conflito", `\b(guerra|ataque|ataques|morre|falece|faleceu)\b`, 1.2),
				pt("mercado", `\b(busca por|cotaç[ãa]o|sobe|cai)\b`, 1.0),
			},
			IntentEvergreen: {
				pt("documentacao", `\b(documenta[çc][ãa]o|refer[êe]ncia|gloss[áa]rio)\b`, 2.0),
				pt("definicao", `\b(o que [ée]|o que s[ãa]o|o que significa|significado de)\b`, 1.8),
				pt("basico", `\b(para iniciantes|introduç[ãa]o ao|b[áa]sico|fundamentos)\b`, 1.6),
				pt("guia", `\b(guia completo|manual de|diret[óo]rio)\b`, 1.4),
				pt("conceito", `\b(conceito|vis[ãa]o geral|por que usar)\b`, 1.2),
			},
			IntentUpdate: {
				pt("versao", `\b(changelog|notas de vers[ãa]o|novidades da vers[ãa]o|release notes)\b`, 2.5),
				pt("novaversao", `\b(nova vers[ãa]o|novo lançamento|atualizaç[ãa]o lançada)\b`, 2.2),
				pt("roadmap", `\b(roadmap|o que vem por a[ií]|pr[óo]ximas novidades)\b`, 1.8),
				pt("compat", `\b(atualizado para|agora compat[ií]vel|suporta a nova vers[ãa]o)\b`, 1.5),
			},
			IntentReview: {
				pt("analise", `\b(an[aá]lise|an[aá]lise completa|avaliaç[ãa]o)\b`, 2.5),
				pt("opiniao", `\b(opini[ãa]o|vale a pena|recomendo|recomendamos)\b`, 1.8),
				pt("teste", `\b(testamos|em nossos testes|nossa experi[êe]ncia com)\b`, 1.8),
				pt("comparativo", `\b(comparaç[ãa]o|comparativo|benchmark|contra o|versus|vs\.)\b`, 1.6),
				pt("proscontras", `\b(prós e contras|pontos fortes|pontos fracos)\b`, 1.5),
			},
			IntentTutorial: {
				pt("passo", `\b(passo a passo|passo-a-passo)\b`, 2.5),
				pt("como", `\b(como fazer|como usar|como instalar|como criar)\b`, 2.2),
				pt("tutorial", `\b(tutorial|guia pr[aá]tico|aprenda|aprendendo)\b`, 2.0),
				pt("procedimento", `\b(instalaç[ãa]o|configuraç[ãa]o|procedimento|exemplo pr[aá]tico)\b`, 1.4),
				pt("solucao", `\b(resolver|soluç[ãa]o de problemas|corrigir)\b`, 1.2),
			},
		}
	default: // en
		return map[IntentType][]signalRule{
			IntentNews: {
				{re: regexp.MustCompile(`(?i)\b(news|breaking|breaking news)\b`), name: "news", weight: 2.0},
				{re: regexp.MustCompile(`(?i)\b(announces|announced|launches|launched|reveals|unveils)\b`), name: "announcement", weight: 1.8},
				{re: regexp.MustCompile(`(?i)\b(today|yesterday|this week|just now)\b`), name: "now", weight: 1.5},
				{re: regexp.MustCompile(`(?i)\b(government|president|election|minister|ministry)\b`), name: "politics", weight: 1.2},
				{re: regexp.MustCompile(`(?i)\b(war|attack|attacks|dies|killed)\b`), name: "conflict", weight: 1.2},
			},
			IntentEvergreen: {
				{re: regexp.MustCompile(`(?i)\b(documentation|reference|glossary)\b`), name: "docs", weight: 2.0},
				{re: regexp.MustCompile(`(?i)\b(what is|what are|definition of|meaning of)\b`), name: "definition", weight: 1.8},
				{re: regexp.MustCompile(`(?i)\b(for beginners|introduction to|basics|fundamentals)\b`), name: "basics", weight: 1.6},
				{re: regexp.MustCompile(`(?i)\b(complete guide|handbook|directory|overview)\b`), name: "guide", weight: 1.4},
				{re: regexp.MustCompile(`(?i)\b(concept|why use)\b`), name: "concept", weight: 1.2},
			},
			IntentUpdate: {
				{re: regexp.MustCompile(`(?i)\b(changelog|release notes|what.s new in)\b`), name: "changelog", weight: 2.5},
				{re: regexp.MustCompile(`(?i)\b(new version|latest version|new release)\b`), name: "newversion", weight: 2.2},
				{re: regexp.MustCompile(`(?i)\b(roadmap|upcoming|coming next)\b`), name: "roadmap", weight: 1.8},
				{re: regexp.MustCompile(`(?i)\b(upgraded to|now compatible with|supports the new)\b`), name: "compat", weight: 1.5},
			},
			IntentReview: {
				{re: regexp.MustCompile(`(?i)\b(review|full review|hands-on|verdict)\b`), name: "review", weight: 2.5},
				{re: regexp.MustCompile(`(?i)\b(opinion|worth it|recommend|recommended)\b`), name: "opinion", weight: 1.8},
				{re: regexp.MustCompile(`(?i)\b(we tested|in our tests|our experience with)\b`), name: "testing", weight: 1.8},
				{re: regexp.MustCompile(`(?i)\b(comparison|benchmark|versus|vs\.|against the)\b`), name: "comparison", weight: 1.6},
				{re: regexp.MustCompile(`(?i)\b(pros and cons|strengths|weaknesses)\b`), name: "proscons", weight: 1.5},
			},
			IntentTutorial: {
				{re: regexp.MustCompile(`(?i)\b(step by step|step-by-step)\b`), name: "steps", weight: 2.5},
				{re: regexp.MustCompile(`(?i)\b(how to|how do i|how can i)\b`), name: "howto", weight: 2.2},
				{re: regexp.MustCompile(`(?i)\b(tutorial|practical guide|learn|learning)\b`), name: "tutorial", weight: 2.0},
				{re: regexp.MustCompile(`(?i)\b(installation|configuration|procedure|walkthrough)\b`), name: "procedure", weight: 1.4},
				{re: regexp.MustCompile(`(?i)\b(troubleshoot|fix|solving)\b`), name: "solution", weight: 1.2},
			},
		}
	}
}

// versionSignal detects version-ish tokens (GPT-6, Gemini 2.5, v4.1, versão 2.3)
// which push a topic toward UPDATE/NEWS.
var versionSignal = regexp.MustCompile(`(?i)\b([a-z0-9]{2,}[- ]\d+(\.\d+){0,2}\b|v\d+(\.\d+){1,2}\b|[a-z]{2,}\s+\d+(\.\d+){1,2}\b)`)

// ClassifyIntent deterministically classifies a topic+content into one of the
// five intents, returning matched signals and a 0..1 confidence.
func ClassifyIntent(topic, content, lang string) (IntentResult, error) {
	if strings.TrimSpace(topic) == "" && strings.TrimSpace(content) == "" {
		return IntentResult{}, ErrIntentRequired
	}
	if lang != "pt" && lang != "en" {
		return IntentResult{}, ErrInvalidLanguage
	}
	text := strings.ToLower(topic + " " + content)
	signals := intentSignals(lang)

	scores := make(map[IntentType]float64)
	matched := make(map[IntentType][]string)
	for intent, rules := range signals {
		var s float64
		for _, r := range rules {
			if r.re.MatchString(text) {
				s += r.weight
				matched[intent] = append(matched[intent], r.name)
			}
		}
		scores[intent] = s
	}

	// Version tokens push toward update when no stronger cue exists.
	if versionSignal.MatchString(text) {
		scores[IntentUpdate] += 0.6
		matched[IntentUpdate] = append(matched[IntentUpdate], "version_token")
	}

	winner, runnerUp := IntentEvergreen, IntentEvergreen
	winScore, secondScore := 0.0, 0.0
	for _, intent := range ValidIntents {
		s := scores[intent]
		if s > winScore {
			runnerUp, secondScore = winner, winScore
			winner, winScore = intent, s
		} else if s > secondScore {
			runnerUp, secondScore = intent, s
		}
	}

	if winScore == 0 {
		// No cues at all: evergreen is the safe, deterministic default.
		return IntentResult{Intent: IntentEvergreen, Confidence: 0.5,
			Signals: []string{"fallback_evergreen"}}, nil
	}

	conf := winScore / (winScore + secondScore)
	if secondScore == 0 {
		conf = 1.0
	}
	if conf < 0.5 {
		conf = 0.5
	}
	if conf > 1.0 {
		conf = 1.0
	}
	_ = runnerUp
	return IntentResult{Intent: winner, Confidence: conf, Signals: matched[winner]}, nil
}
