package prompt

import (
	"strings"
	"testing"
)

func TestTemplateHasRequiredSections(t *testing.T) {
	out := Build(TemplateInput{
		Objective:     "Implement feature X",
		TaskType:      "feature",
		Backend:       "codex",
		BudgetTokens:  12000,
		ContextDigest: "abc123",
	})

	required := []string{
		"Objective:",
		"Constraints:",
		"Context:",
		"Deliverables:",
		"Test plan:",
		"Definition of Done:",
	}
	for _, section := range required {
		if !strings.Contains(out, section) {
			t.Fatalf("missing section %q in template", section)
		}
	}
}
