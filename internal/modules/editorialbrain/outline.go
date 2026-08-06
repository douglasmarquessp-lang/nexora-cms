package editorialbrain

import (
	"fmt"
	"strings"
)

// sectionDef describes a section candidate with its bilingual title/purpose.
type sectionDef struct {
	typ     SectionType
	title   bi
	purpose bi
}

// sectionOrder defines the fixed logical order of outline sections.
var sectionOrder = []SectionType{
	SecIntro, SecWhatIs, SecHowWorks, SecSteps, SecCost, SecComparison,
	SecTable, SecChanges, SecLimits, SecWhoUses, SecAlternates, SecFAQ,
	SecCallout, SecConclusion,
}

// allSections are the bilingual section definitions.
var allSections = map[SectionType]sectionDef{
	SecIntro:      {SecIntro, b("Introdução", "Introduction"), b("Gancho, contexto e o que o leitor vai descobrir.", "Hook, context and what the reader will discover.")},
	SecWhatIs:     {SecWhatIs, b("O que é?", "What is it?"), b("Definição clara do assunto.", "Clear definition of the subject.")},
	SecHowWorks:   {SecHowWorks, b("Como funciona?", "How does it work?"), b("Mecânica e funcionamento explicados de forma simples.", "Mechanics explained in plain terms.")},
	SecSteps:      {SecSteps, b("Passo a passo", "Step by step"), b("Instruções práticas em ordem lógica.", "Practical instructions in logical order.")},
	SecCost:       {SecCost, b("Quanto custa?", "How much does it cost?"), b("Preços, planos e custo-benefício.", "Pricing, plans and cost-benefit.")},
	SecComparison: {SecComparison, b("Comparação", "Comparison"), b("Comparação direta entre opções.", "Direct comparison between options.")},
	SecTable:      {SecTable, b("Tabela comparativa", "Comparison table"), b("Tabela com os pontos-chave lado a lado.", "Table with key points side by side.")},
	SecChanges:    {SecChanges, b("O que mudou?", "What changed?"), b("Novidades, mudanças e melhorias desta versão.", "News, changes and improvements of this version.")},
	SecLimits:     {SecLimits, b("Limitações", "Limitations"), b("Restrições, desvantagens e casos em que não se aplica.", "Restrictions, drawbacks and cases where it does not apply.")},
	SecWhoUses:    {SecWhoUses, b("Quem deve usar?", "Who should use it?"), b("Perfis de usuário ideais e casos de uso.", "Ideal user profiles and use cases.")},
	SecAlternates: {SecAlternates, b("Alternativas", "Alternatives"), b("Opções semelhantes e quando escolher cada uma.", "Similar options and when to choose each.")},
	SecFAQ:        {SecFAQ, b("Perguntas frequentes", "Frequently asked questions"), b("Respostas diretas às dúvidas mais comuns.", "Direct answers to the most common questions.")},
	SecCallout:    {SecCallout, b("Dicas importantes", "Important tips"), b("Callouts com atenção, dicas e avisos.", "Callouts with warnings, tips and notes.")},
	SecConclusion: {SecConclusion, b("Conclusão", "Conclusion"), b("Resumo final e recomendação.", "Final summary and recommendation.")},
}

// sectionsForIntent returns the section types for an intent.
func sectionsForIntent(intent SearchIntent) map[SectionType]bool {
	all := map[SectionType]bool{
		SecIntro: true, SecFAQ: true, SecConclusion: true,
	}
	switch intent {
	case IntentTutorial:
		all[SecWhatIs] = true
		all[SecHowWorks] = true
		all[SecSteps] = true
		all[SecCallout] = true
		all[SecLimits] = true
	case IntentComparison:
		all[SecWhatIs] = true
		all[SecHowWorks] = true
		all[SecComparison] = true
		all[SecTable] = true
		all[SecAlternates] = true
		all[SecCost] = true
		all[SecLimits] = true
	case IntentCommercial:
		all[SecWhatIs] = true
		all[SecCost] = true
		all[SecWhoUses] = true
		all[SecAlternates] = true
		all[SecLimits] = true
	case IntentNavigational:
		all[SecWhatIs] = true
		all[SecHowWorks] = true
		all[SecSteps] = true
	case IntentBreakingNews:
		all[SecWhatIs] = true
		all[SecWhoUses] = true
		all[SecLimits] = true
	case IntentUpdate:
		all[SecWhatIs] = true
		all[SecChanges] = true
		all[SecCost] = true
		all[SecLimits] = true
	default: // informational
		all[SecWhatIs] = true
		all[SecHowWorks] = true
		all[SecWhoUses] = true
		all[SecLimits] = true
		all[SecAlternates] = true
	}
	if intent == IntentComparison {
		all[SecWhoUses] = true
	}
	if intent == IntentTutorial {
		all[SecWhoUses] = true
	}
	return all
}

// sectionsForPersona adds persona-specific sections.
func sectionsForPersona(persona Persona) map[SectionType]bool {
	all := map[SectionType]bool{}
	switch persona {
	case PersonaDeveloper:
		all[SecSteps] = true
		all[SecLimits] = true
	case PersonaBusiness:
		all[SecCost] = true
		all[SecWhoUses] = true
		all[SecAlternates] = true
	case PersonaCreator:
		all[SecHowWorks] = true
		all[SecCallout] = true
	}
	return all
}

// suggestedTitleFor builds the ideal title for the topic + intent.
func suggestedTitleFor(topic, language string, intent SearchIntent) string {
	topic = strings.TrimSpace(topic)
	switch intent {
	case IntentTutorial:
		return fmt.Sprintf(b("Como %s: guia passo a passo", "How to %s: a step-by-step guide").text(language), topic)
	case IntentComparison:
		return fmt.Sprintf(b("%s: comparação completa", "%s: full comparison").text(language), topic)
	case IntentBreakingNews:
		return fmt.Sprintf(b("%s: o que você precisa saber", "%s: what you need to know").text(language), topic)
	case IntentUpdate:
		return fmt.Sprintf(b("%s: o que mudou na nova versão", "%s: what changed in the new version").text(language), topic)
	case IntentCommercial:
		return fmt.Sprintf(b("%s: preço, custos e vale a pena?", "%s: price, costs and is it worth it?").text(language), topic)
	case IntentNavigational:
		return fmt.Sprintf(b("%s: guia de acesso e uso", "%s: access and usage guide").text(language), topic)
	default:
		return fmt.Sprintf(b("%s: o que é e como funciona", "%s: what it is and how it works").text(language), topic)
	}
}

// GenerateOutline builds the intelligent outline (title, sections, order,
// FAQs, tables, comparisons, callouts, conclusion) before the AI writes.
// Fully deterministic — same input, same outline.
func GenerateOutline(topic, language string, intent SearchIntent, persona Persona) EditorialOutline {
	intentSet := sectionsForIntent(intent)
	personaSet := sectionsForPersona(persona)

	questions := requiredQuestionsFor(topic, language, intent)
	faqs := make([]string, 0, len(questions))
	for _, q := range questions {
		faqs = append(faqs, q.Question)
	}
	if len(faqs) > 5 {
		faqs = faqs[:5]
	}

	sections := make([]OutlineSection, 0)
	order := 1
	for _, t := range sectionOrder {
		if intentSet[t] || personaSet[t] {
			def := allSections[t]
			title := def.title.text(language)
			if t == SecWhatIs {
				title = b("O que é %s?", "What is %s?").text(language)
				title = fmt.Sprintf(title, strings.ToLower(topic))
			}
			sections = append(sections, OutlineSection{
				Order:   order,
				Type:    t,
				Title:   title,
				Purpose: def.purpose.text(language),
			})
			order++
		}
	}

	needsTable := intent == IntentComparison || (persona == PersonaBusiness && intent == IntentCommercial)
	comparisons := intent == IntentComparison
	callouts := intent == IntentTutorial || persona == PersonaCreator

	rationale := b(
		fmt.Sprintf("Ângulo %s direcionado a %s, com seções ordenadas para maximizar retenção.",
			intent.Label(language), persona.AudienceLabel(language)),
		fmt.Sprintf("%s angle targeted at %s, with sections ordered to maximize retention.",
			intent.Label(language), persona.AudienceLabel(language)),
	).text(language)

	return EditorialOutline{
		SuggestedTitle: suggestedTitleFor(topic, language, intent),
		Sections:       sections,
		FAQs:           faqs,
		NeedsTable:     needsTable,
		Comparisons:    comparisons,
		Callouts:       callouts,
		Rationale:      rationale,
	}
}
