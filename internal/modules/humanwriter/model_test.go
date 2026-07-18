package humanwriter

import (
	"testing"
)

func TestProfileSlugs_AllValid(t *testing.T) {
	expected := []ProfileSlug{
		ProfileJournalist, ProfileTechWriter, ProfileSoftwareReviewer,
		ProfileNewsReporter, ProfileTutorialAuthor, ProfileEditorialWriter,
		ProfileOpinionWriter, ProfileEvergreenWriter,
	}

	if len(AllProfiles) != len(expected) {
		t.Errorf("expected %d profiles, got %d", len(expected), len(AllProfiles))
	}

	seen := make(map[ProfileSlug]bool)
	for _, p := range AllProfiles {
		if seen[p] {
			t.Errorf("duplicate profile slug: %s", p)
		}
		seen[p] = true
		if _, ok := ProfileDefaults[p]; !ok {
			t.Errorf("missing defaults for profile: %s", p)
		}
	}
}

func TestRuleKeys_AllValid(t *testing.T) {
	expected := map[string]bool{
		RuleKeys.AvoidAICliches:             true,
		RuleKeys.AvoidRepetitiveOpenings:    true,
		RuleKeys.AvoidRepetitiveConclusions: true,
		RuleKeys.NaturalParagraphSizes:      true,
		RuleKeys.VariableSentenceLengths:    true,
		RuleKeys.NaturalConnectorRotation:   true,
		RuleKeys.QuoteInsertionSupport:      true,
		RuleKeys.StatisticInsertionSupport:  true,
		RuleKeys.ExpertOpinionPlaceholders:  true,
	}

	if len(AllRuleKeys) != len(expected) {
		t.Errorf("expected %d rule keys, got %d", len(expected), len(AllRuleKeys))
	}

	for _, k := range AllRuleKeys {
		if !expected[k] {
			t.Errorf("unexpected rule key: %s", k)
		}
		if _, ok := RuleCategories[k]; !ok {
			t.Errorf("missing category for rule: %s", k)
		}
	}
}

func TestProfileDisplayNames_AllProfiles(t *testing.T) {
	for _, p := range AllProfiles {
		if _, ok := ProfileDisplayNames[p]; !ok {
			t.Errorf("missing display name for profile: %s", p)
		}
	}
}

func TestRuleCategories_ValidCategories(t *testing.T) {
	validCats := map[string]bool{
		"style": true, "structure": true, "readability": true,
		"flow": true, "enrichment": true,
	}
	for k, cat := range RuleCategories {
		if !validCats[cat] {
			t.Errorf("rule %s has invalid category: %s", k, cat)
		}
	}
}

func TestEventTypes_Valid(t *testing.T) {
	events := []string{
		string(EventHumanCreated), string(EventHumanized), string(EventProfileCreated),
		string(EventProfileUpdated), string(EventProfileDeleted), string(EventRuleToggled),
		string(EventPersonaCreated), string(EventPersonaUpdated), string(EventPersonaDeleted),
		string(EventBatchHumanized),
	}
	for _, e := range events {
		if e == "" {
			t.Error("empty event type")
		}
	}
}

func TestSentinelErrors_NotEmpty(t *testing.T) {
	errs := []error{
		ErrProfileNotFound, ErrRuleNotFound, ErrPersonaNotFound,
		ErrVocabularyNotFound, ErrTransitionNotFound, ErrPatternNotFound,
		ErrTemplateNotFound, ErrHistoryNotFound, ErrInvalidSlug,
		ErrInvalidText, ErrInvalidLanguage, ErrProfileExists,
		ErrDatabaseNotAvail,
	}
	for _, err := range errs {
		if err.Error() == "" {
			t.Error("empty error message")
		}
	}
}
