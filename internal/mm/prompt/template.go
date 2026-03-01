package prompt

import (
	"fmt"
	"strings"
)

type TemplateInput struct {
	Objective      string
	TaskType       string
	SkillName      string
	SkillSnippet   string
	ContextDigest  string
	Backend        string
	BudgetTokens   int
	AdditionalHint string
}

func Build(input TemplateInput) string {
	objective := strings.TrimSpace(input.Objective)
	if objective == "" {
		objective = "No explicit objective provided."
	}
	taskType := strings.TrimSpace(input.TaskType)
	if taskType == "" {
		taskType = "general-coding"
	}

	var b strings.Builder
	b.WriteString("Objective:\n")
	b.WriteString("- Task type: " + taskType + "\n")
	b.WriteString("- Backend target: " + strings.TrimSpace(input.Backend) + "\n")
	b.WriteString("- User objective: " + objective + "\n\n")

	b.WriteString("Constraints:\n")
	b.WriteString("- Keep changes minimal and production-safe.\n")
	b.WriteString("- Prefer deterministic steps and explicit validation.\n")
	if input.BudgetTokens > 0 {
		b.WriteString(fmt.Sprintf("- Soft token budget: %d\n", input.BudgetTokens))
	}
	b.WriteString("- Never expose secrets in output.\n\n")

	b.WriteString("Context:\n")
	b.WriteString("- Use provided context bundle and git metadata.\n")
	if strings.TrimSpace(input.ContextDigest) != "" {
		b.WriteString("- Context digest: " + strings.TrimSpace(input.ContextDigest) + "\n")
	}
	if strings.TrimSpace(input.SkillName) != "" {
		b.WriteString("- Skill: " + strings.TrimSpace(input.SkillName) + "\n")
	}
	if snippet := strings.TrimSpace(input.SkillSnippet); snippet != "" {
		b.WriteString("\nSkill Snippet:\n")
		b.WriteString(snippet)
		b.WriteString("\n")
	}
	if hint := strings.TrimSpace(input.AdditionalHint); hint != "" {
		b.WriteString("\nHouse Rules:\n")
		b.WriteString(hint)
		b.WriteString("\n")
	}
	b.WriteString("\n")

	b.WriteString("Deliverables:\n")
	b.WriteString("- Implement requested changes.\n")
	b.WriteString("- Explain what changed and why.\n")
	b.WriteString("- Include key file references.\n\n")

	b.WriteString("Test plan:\n")
	b.WriteString("- Run focused tests/build checks for changed areas.\n")
	b.WriteString("- Report any unrun tests and residual risk.\n\n")

	b.WriteString("Definition of Done:\n")
	b.WriteString("- Requested behavior is implemented.\n")
	b.WriteString("- Build/tests pass for touched paths (or documented blockers).\n")
	b.WriteString("- Summary includes next actions if needed.\n")

	return b.String()
}
