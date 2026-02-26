package wizard

import (
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
)

// ArmorClaw brand colors (matching the bash wizard output).
var (
	colorCyan    = lipgloss.Color("6")   // ANSI cyan
	colorGreen   = lipgloss.Color("2")   // ANSI green
	colorYellow  = lipgloss.Color("3")   // ANSI yellow
	colorRed     = lipgloss.Color("1")   // ANSI red
	colorWhite   = lipgloss.Color("15")  // bright white
	colorDimGray = lipgloss.Color("240") // dim gray for descriptions
)

// ArmorClawTheme returns a Huh? theme branded for ArmorClaw.
// It uses the Charm default theme as a base and overrides accent colors
// to match the existing cyan/green terminal output.
func ArmorClawTheme() *huh.Theme {
	t := huh.ThemeCharm()

	// Focused field styling
	t.Focused.Title = t.Focused.Title.Foreground(colorCyan).Bold(true)
	t.Focused.Description = t.Focused.Description.Foreground(colorDimGray)
	t.Focused.ErrorIndicator = t.Focused.ErrorIndicator.Foreground(colorRed)
	t.Focused.ErrorMessage = t.Focused.ErrorMessage.Foreground(colorRed)
	t.Focused.SelectSelector = t.Focused.SelectSelector.Foreground(colorGreen)
	t.Focused.SelectedOption = t.Focused.SelectedOption.Foreground(colorGreen)
	t.Focused.FocusedButton = t.Focused.FocusedButton.
		Background(colorCyan).
		Foreground(colorWhite)
	t.Focused.BlurredButton = t.Focused.BlurredButton.
		Background(lipgloss.Color("237")).
		Foreground(colorDimGray)

	// Blurred (unfocused) field styling
	t.Blurred.Title = t.Blurred.Title.Foreground(colorDimGray)
	t.Blurred.Description = t.Blurred.Description.Foreground(lipgloss.Color("236"))
	t.Blurred.SelectedOption = t.Blurred.SelectedOption.Foreground(colorDimGray)

	return t
}
