package tui

import (
	"github.com/charmbracelet/lipgloss"
)

var (
	BorderColor   = lipgloss.Color("#5A5A5A")
	AccentColor   = lipgloss.Color("#00D7FF")
	SelectedColor = lipgloss.Color("#FF6B6B")
	TextColor     = lipgloss.Color("#FFFFFF")
	SubtleColor   = lipgloss.Color("#888888")
	ErrorColor    = lipgloss.Color("#FF5555")
	SuccessColor  = lipgloss.Color("#50FA7B")
)

var (
	BaseStyle = lipgloss.NewStyle().
			Foreground(TextColor).
			Background(lipgloss.Color("#1A1A1A"))

	PaneStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(BorderColor).
			Padding(0, 1).
			Margin(0, 1)

	ActivePaneStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(AccentColor).
			Padding(0, 1).
			Margin(0, 1)

	SelectedItemStyle = lipgloss.NewStyle().
				Background(SelectedColor).
				Foreground(lipgloss.Color("#FFFFFF")).
				Bold(true)

	ItemStyle = lipgloss.NewStyle().
			Foreground(TextColor)

	SubtleItemStyle = lipgloss.NewStyle().
			Foreground(SubtleColor)

	TabActiveStyle = lipgloss.NewStyle().
			Background(AccentColor).
			Foreground(lipgloss.Color("#000000")).
			Padding(0, 2).
			Bold(true)

	TabInactiveStyle = lipgloss.NewStyle().
				Background(BorderColor).
				Foreground(TextColor).
				Padding(0, 2)

	SearchBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(AccentColor).
			Padding(0, 1).
			Margin(0, 1)

	StatusBarStyle = lipgloss.NewStyle().
			Background(BorderColor).
			Foreground(TextColor).
			Padding(0, 1)

	HelpStyle = lipgloss.NewStyle().
			Foreground(SubtleColor).
			Italic(true)

	ErrorStyle = lipgloss.NewStyle().
			Foreground(ErrorColor).
			Bold(true)

	SuccessStyle = lipgloss.NewStyle().
			Foreground(SuccessColor).
			Bold(true)

	HeaderStyle = lipgloss.NewStyle().
			Foreground(AccentColor).
			Bold(true)

	DataTypeStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFB86C")).
			Bold(true)

	TableCellStyle = lipgloss.NewStyle().
			Padding(0, 1).
			MaxWidth(20)

	SelectedHeaderStyle = lipgloss.NewStyle().
				Background(AccentColor).
				Foreground(lipgloss.Color("#000000")).
				Bold(true)

	SelectedRowStyle = lipgloss.NewStyle().
				Background(BorderColor).
				Foreground(TextColor)
)
