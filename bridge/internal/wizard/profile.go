package wizard

import (
	"github.com/charmbracelet/huh"
)

// runProfileForm presents the deployment profile selection.
// Returns the selected profile string or an error if the user aborts.
func runProfileForm(accessible bool) (string, error) {
	var profile string

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Choose your deployment profile").
				Description("This determines the setup flow and default security posture.").
				Options(
					huh.NewOption("Quick Start — Fewest questions, running in ~2 minutes", ProfileQuick),
					huh.NewOption("Enterprise / Compliance — PII/PHI scrubbing, HIPAA, audit logging", ProfileEnterprise),
				).
				Value(&profile),
		),
	)

	form = formOpts(form, accessible)

	if err := form.Run(); err != nil {
		return "", err
	}

	return profile, nil
}
