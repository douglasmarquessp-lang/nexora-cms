package editorialbrain

import (
	"fmt"
	"strings"
)

// questionDef defines a required question with bilingual text and
// answer-detection keywords.
type questionDef struct {
	id       QuestionID
	question bi
	markers  []string
}

// questionDefs is the master list of questions an article must answer.
var questionDefs = map[QuestionID]questionDef{
	QWhatIs: {
		id:       QWhatIs,
		question: b("O que é %s?", "What is %s?"),
		markers:  []string{"o que é", "é um", "é uma", "é uma ferramenta", "what is", "is a", "is an", "refere-se", "refers to", "definid"},
	},
	QHowWorks: {
		id:       QHowWorks,
		question: b("Como funciona?", "How does it work?"),
		markers:  []string{"como funciona", "como usar", "como fazer", "funciona por", "how it works", "how to use", "how to do", "works by", "passo"},
	},
	QCost: {
		id:       QCost,
		question: b("Quanto custa?", "How much does it cost?"),
		markers:  []string{"custa", "preço", "preco", "preços", "precos", "price", "cost", "pricing", "r$", "us$", "usd", "eur", "assinatura", "subscription", "plano", "plan"},
	},
	QWorthIt: {
		id:       QWorthIt,
		question: b("Vale a pena?", "Is it worth it?"),
		markers:  []string{"vale a pena", "worth", "vantagens", "vantagem", "benefícios", "beneficios", "benefits", "pros", "cons", "prós", "contras"},
	},
	QWhatChanged: {
		id:       QWhatChanged,
		question: b("O que mudou?", "What changed?"),
		markers:  []string{"mudou", "novo", "nova", "novidade", "atualizaç", "atualizac", "changed", "new", "update", "release notes", "changelog", "melhorias", "improvements"},
	},
	QWhoCanUse: {
		id:       QWhoCanUse,
		question: b("Quem pode usar?", "Who can use it?"),
		markers:  []string{"quem pode", "para quem", "ideal para", "perfeito para", "indicado para", "who can", "for whom", "perfect for", "ideal for", "best for"},
	},
	QLimits: {
		id:       QLimits,
		question: b("Quais são as limitações?", "What are the limitations?"),
		markers:  []string{"limitaç", "limitac", "limitation", "desvantag", "drawback", "não pode", "nao pode", "cannot", "não suporta", "nao suporta", "does not support", "restriç", "restric"},
	},
	QAlternates: {
		id:       QAlternates,
		question: b("Quais são as alternativas?", "What are the alternatives?"),
		markers:  []string{"alternativ", "opções", "opcoes", "options", "similar", "similares", "concorrent", "competitor", "em vez de", "instead of", "melhores"},
	},
}

// intentsQuestions maps each intent to its required question set.
var intentsQuestions = map[SearchIntent][]QuestionID{
	IntentTutorial:     {QWhatIs, QHowWorks, QWhoCanUse, QLimits},
	IntentComparison:   {QWhatIs, QCost, QWorthIt, QLimits, QAlternates},
	IntentCommercial:   {QWhatIs, QCost, QWorthIt, QWhoCanUse, QLimits, QAlternates},
	IntentNavigational: {QWhatIs, QHowWorks, QWhoCanUse},
	IntentBreakingNews: {QWhatIs, QWhatChanged, QWhoCanUse},
	IntentUpdate:       {QWhatIs, QWhatChanged, QCost, QLimits},
	IntentInformational: {QWhatIs, QHowWorks, QWhoCanUse, QLimits, QAlternates},
}

// requiredQuestionsFor builds the question list for topic+intent.
func requiredQuestionsFor(topic, language string, intent SearchIntent) []RequiredQuestion {
	ids, ok := intentsQuestions[intent]
	if !ok {
		ids = intentsQuestions[IntentInformational]
	}
	out := make([]RequiredQuestion, 0, len(ids))
	for _, id := range ids {
		def := questionDefs[id]
		text := def.question.text(language)
		if strings.Contains(text, "%s") {
			text = fmt.Sprintf(text, strings.ToLower(strings.TrimSpace(topic)))
		}
		out = append(out, RequiredQuestion{ID: id, Question: text})
	}
	return out
}

// GenerateQuestions builds the list of questions the article must answer.
func GenerateQuestions(topic, language string, intent SearchIntent) []RequiredQuestion {
	return requiredQuestionsFor(topic, language, intent)
}

// verifyQuestion determines whether the article text answers the question.
func verifyQuestion(q RequiredQuestion, text string) (bool, string) {
	lower := strings.ToLower(text)
	def := questionDefs[q.ID]
	for _, m := range def.markers {
		if strings.Contains(lower, m) {
			return true, firstMatchingSentence(text, m)
		}
	}
	return false, ""
}

// firstMatchingSentence returns the first sentence containing the marker.
func firstMatchingSentence(text, marker string) string {
	marker = strings.ToLower(marker)
	for _, s := range sentences(text) {
		if strings.Contains(strings.ToLower(s), marker) {
			return s
		}
	}
	return ""
}

// CheckQuestions verifies every required question was answered in the text.
func CheckQuestions(text string, questions []RequiredQuestion) QuestionCheck {
	answered := 0
	out := make([]RequiredQuestion, 0, len(questions))
	for _, q := range questions {
		ok, ev := verifyQuestion(q, text)
		if ok {
			answered++
		}
		q.Answered = ok
		q.Evidence = ev
		out = append(out, q)
	}
	total := len(out)
	percent := 0.0
	if total > 0 {
		percent = clampScore(float64(answered) / float64(total) * 100)
	}
	return QuestionCheck{
		Questions:       out,
		AnsweredCount:   answered,
		Total:           total,
		AnsweredPercent: percent,
	}
}
