package tui

import "github.com/charmbracelet/lipgloss"

// Theme defines the color scheme for the TUI
type Theme struct {
	IsDark bool

	Background  lipgloss.AdaptiveColor
	Foreground  lipgloss.AdaptiveColor
	Muted       lipgloss.AdaptiveColor
	Accent      lipgloss.AdaptiveColor
	AccentSoft  lipgloss.AdaptiveColor
	Success     lipgloss.AdaptiveColor
	Warning     lipgloss.AdaptiveColor
	Danger      lipgloss.AdaptiveColor
	Border      lipgloss.AdaptiveColor
	Card        lipgloss.AdaptiveColor
	ActiveTab   lipgloss.AdaptiveColor
	InactiveTab lipgloss.AdaptiveColor
}

// CurrentTheme returns theme with adaptive colors respecting terminal/OS light or dark background
func CurrentTheme(mode string) Theme {
	isDark := true
	if mode == "light" {
		isDark = false
	}

	return Theme{
		IsDark: isDark,
		Background: lipgloss.AdaptiveColor{
			Light: "#F8FAFC", // Slate 50
			Dark:  "#0F172A", // Slate 900
		},
		Foreground: lipgloss.AdaptiveColor{
			Light: "#0F172A", // Slate 900
			Dark:  "#F8FAFC", // Slate 50
		},
		Muted: lipgloss.AdaptiveColor{
			Light: "#64748B", // Slate 500
			Dark:  "#94A3B8", // Slate 400
		},
		Accent: lipgloss.AdaptiveColor{
			Light: "#4F46E5", // Indigo 600
			Dark:  "#818CF8", // Indigo 400
		},
		AccentSoft: lipgloss.AdaptiveColor{
			Light: "#EEF2FF",
			Dark:  "#1E1B4B",
		},
		Success: lipgloss.AdaptiveColor{
			Light: "#16A34A", // Green 600
			Dark:  "#4ADE80", // Green 400
		},
		Warning: lipgloss.AdaptiveColor{
			Light: "#D97706", // Amber 600
			Dark:  "#FBBF24", // Amber 400
		},
		Danger: lipgloss.AdaptiveColor{
			Light: "#DC2626", // Red 600
			Dark:  "#F87171", // Red 400
		},
		Border: lipgloss.AdaptiveColor{
			Light: "#CBD5E1", // Slate 300
			Dark:  "#334155", // Slate 700
		},
		Card: lipgloss.AdaptiveColor{
			Light: "#FFFFFF",
			Dark:  "#1E293B", // Slate 800
		},
		ActiveTab: lipgloss.AdaptiveColor{
			Light: "#4F46E5",
			Dark:  "#818CF8",
		},
		InactiveTab: lipgloss.AdaptiveColor{
			Light: "#64748B",
			Dark:  "#64748B",
		},
	}
}

// Styles provides pre-computed Lipgloss styles
type Styles struct {
	HeaderTitle  lipgloss.Style
	HeaderSub    lipgloss.Style
	TabActive    lipgloss.Style
	TabInactive  lipgloss.Style
	Card         lipgloss.Style
	SelectedRow  lipgloss.Style
	NormalRow    lipgloss.Style
	BadgeSuccess lipgloss.Style
	BadgeWarning lipgloss.Style
	BadgeDanger  lipgloss.Style
	BadgeInfo    lipgloss.Style
	HelpText     lipgloss.Style
	KeyBadge     lipgloss.Style
}

func MakeStyles(theme Theme) Styles {
	return Styles{
		HeaderTitle: lipgloss.NewStyle().
			Bold(true).
			Foreground(theme.Accent).
			Padding(0, 1),

		HeaderSub: lipgloss.NewStyle().
			Foreground(theme.Muted),

		TabActive: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(theme.Accent).
			Padding(0, 2).
			MarginRight(1),

		TabInactive: lipgloss.NewStyle().
			Foreground(theme.Muted).
			Background(theme.Card).
			Padding(0, 2).
			MarginRight(1),

		Card: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(theme.Border).
			Background(theme.Card).
			Padding(1, 2),

		SelectedRow: lipgloss.NewStyle().
			Bold(true).
			Foreground(theme.Accent).
			Background(theme.AccentSoft).
			Padding(0, 1),

		NormalRow: lipgloss.NewStyle().
			Foreground(theme.Foreground).
			Padding(0, 1),

		BadgeSuccess: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(theme.Success).
			Padding(0, 1),

		BadgeWarning: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#000000")).
			Background(theme.Warning).
			Padding(0, 1),

		BadgeDanger: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(theme.Danger).
			Padding(0, 1),

		BadgeInfo: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(theme.Muted).
			Padding(0, 1),

		HelpText: lipgloss.NewStyle().
			Foreground(theme.Muted).
			MarginTop(1),

		KeyBadge: lipgloss.NewStyle().
			Bold(true).
			Foreground(theme.Foreground).
			Background(theme.Border).
			Padding(0, 1),
	}
}
