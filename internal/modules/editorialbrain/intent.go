package editorialbrain

import (
	"regexp"
	"strings"
)

// intentSignal is a weighted cue contributing to a search intent.
type intentSignal struct {
	re     *regexp.Regexp
	weight float64
}

func re(s string) *regexp.Regexp { return regexp.MustCompile(`(?i)(?:` + s + `)`) }

// intentCues maps each intent to its per-language cue groups (lang "pt"/"en").
// Each group is a separate weighted signal, so several matches accumulate.
var intentCues = map[SearchIntent]map[string][]intentSignal{
	IntentBreakingNews: {
		"pt": {
			{re(`breaking|urgente|última hora|ultima hora|ao vivo|notícia de última|noticia de ultima`), 2.0},
			{re(`acabou de anunciar|acaba de lançar|anunciado hoje|lançado hoje`), 1.8},
			{re(`lançamento|lançou|lançado|lançada`), 1.2},
		},
		"en": {
			{re(`breaking|urgent|live|latest news`), 2.0},
			{re(`just announced|just launched|reported today|launched today`), 1.8},
			{re(`launch|launched`), 1.2},
		},
	},
	IntentTutorial: {
		"pt": {
			{re(`como fazer|como usar|como criar|como configurar|como instalar|passo a passo|guia completo|guia passo a passo|tutorial|aprenda|iniciantes`), 2.0},
			{re(`passo 1|primeiro passo|etapa 1`), 1.2},
		},
		"en": {
			{re(`how to|step by step|tutorial|guide|learn|setup|configure|walkthrough|beginner|dummies|masterclass`), 2.0},
			{re(`step 1|first step`), 1.2},
		},
	},
	IntentComparison: {
		"pt": {
			{re(`\bvs\b|versus|contra\b`), 1.8},
			{re(`comparar|comparaç|comparac|melhor que|alternativa a|diferença entre|diferenca entre|top 5|melhores`), 1.8},
		},
		"en": {
			{re(`\bvs\b|versus|against`), 1.8},
			{re(`compare|comparison|comparative|better than|alternative to|difference between|top 5|best of`), 1.8},
		},
	},
	IntentCommercial: {
		"pt": {
			{re(`quanto custa|custo|preço|precos|valor|barato|caro|vale a pena|assinatura|plano|promoç|desconto`), 1.8},
			{re(`comprar|onde comprar`), 1.8},
		},
		"en": {
			{re(`how much|cost|price|pricing|worth|subscription|plan|cheap|expensive|deal|discount`), 1.8},
			{re(`buy|where to buy|purchase`), 1.8},
		},
	},
	IntentNavigational: {
		"pt": {
			{re(`\blogin\b|entrar no site|site oficial|logar`), 1.5},
			{re(`download|baixar|registro|inscrever-se|criar conta|acessar`), 1.5},
		},
		"en": {
			{re(`\blogin\b|sign in|official site|official website|access`), 1.5},
			{re(`download|register|sign up|create account`), 1.5},
		},
	},
	IntentUpdate: {
		"pt": {
			{re(`atualizaç|o que mudou|novidades|changelog|nova versão|novo recurso|melhorias|o que há de novo`), 1.6},
		},
		"en": {
			{re(`what changed|what's new|changelog|release notes|new version|new feature|upgrade|updated|improvements`), 1.6},
		},
	},
	IntentInformational: {
		"pt": {
			{re(`o que é|o que sao|o que são|como funciona|entenda|guia sobre|significado|história|origem|tipos de|o que significa`), 1.2},
		},
		"en": {
			{re(`what is|what are|how does it work|explained|understand|guide to|meaning|history|origin|types of|overview of`), 1.2},
		},
	},
}

// intentTieBreak defines the precedence used when two intents tie.
var intentTieBreak = []SearchIntent{
	IntentBreakingNews, IntentTutorial, IntentComparison,
	IntentCommercial, IntentNavigational, IntentUpdate, IntentInformational,
}

// ClassifyIntent deterministically classifies the search intent of a topic.
// No AI, no randomness: the same topic always produces the same intent.
func ClassifyIntent(topic, language string) IntentResult {
	topic = strings.TrimSpace(topic)
	if topic == "" {
		return IntentResult{Intent: IntentInformational, Confidence: 0.5, Language: language}
	}

	scores := make(map[SearchIntent]float64)
	signals := make(map[SearchIntent][]string)
	lang := "en"
	if language == "pt" {
		lang = "pt"
	}
	for intent, langs := range intentCues {
		for _, c := range langs[lang] {
			if c.re.MatchString(topic) {
				scores[intent] += c.weight
				signals[intent] = append(signals[intent], c.re.String())
			}
		}
	}

	// Version tokens ("GPT-6", "Gemini 2.5") boost UPDATE.
	versionHint := versionTokenRE.MatchString(topic)
	if versionHint {
		scores[IntentUpdate] += 0.6
	}

	winner := IntentInformational
	winnerScore := 0.0
	runnerScore := 0.0
	for _, intent := range intentTieBreak {
		s := scores[intent]
		if s > winnerScore {
			runnerScore = winnerScore
			winner, winnerScore = intent, s
		} else if s > runnerScore {
			runnerScore = s
		}
	}

	confidence := 0.5
	if winnerScore > 0 {
		if runnerScore > 0 {
			confidence = winnerScore / (winnerScore + runnerScore)
		} else {
			runnerScore = 0
			confidence = 0.8
		}
	}

	winSignals := signals[winner]
	if versionHint {
		winSignals = append(winSignals, "version_token")
	}

	return IntentResult{
		Intent:      winner,
		Confidence:  clampScore(confidence),
		Signals:     sortedSignalSet(winSignals),
		VersionHint: versionHint,
		Language:    language,
	}
}
