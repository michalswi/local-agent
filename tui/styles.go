package tui

import (
	"github.com/charmbracelet/lipgloss"
)

var (
	// Colors
	primaryColor   = lipgloss.Color("#7D56F4")
	secondaryColor = lipgloss.Color("#FF79C6")
	successColor   = lipgloss.Color("#50FA7B")
	warningColor   = lipgloss.Color("#FFB86C")
	errorColor     = lipgloss.Color("#FF5555")
	subtleColor    = lipgloss.Color("#6272A4")

	// Styles
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(primaryColor).
			MarginBottom(1)

	sectionStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(secondaryColor).
			MarginTop(1)

	labelStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#8BE9FD"))

	subtleStyle = lipgloss.NewStyle().
			Foreground(subtleColor)

	helpStyle = lipgloss.NewStyle().
			Foreground(subtleColor).
			Italic(true).
			MarginTop(1)

	progressStyle = lipgloss.NewStyle().
			Foreground(successColor)

	errorStyle = lipgloss.NewStyle().
			Foreground(errorColor).
			Bold(true)

	warningStyle = lipgloss.NewStyle().
			Foreground(warningColor).
			Bold(true)

	successStyle = lipgloss.NewStyle().
			Foreground(successColor).
			Bold(true)

	infoStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#8BE9FD")).
			Bold(true)

	// Interactive mode styles
	headerStyle = lipgloss.NewStyle().
			Foreground(primaryColor).
			Bold(true).
			Padding(0, 1)

	userHeaderStyle = lipgloss.NewStyle().
			Foreground(successColor).
			Bold(true)

	assistantHeaderStyle = lipgloss.NewStyle().
				Foreground(secondaryColor).
				Bold(true)

	userMessageStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#F8F8F2"))

	assistantMessageStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#F8F8F2"))

	fileHeaderStyle = lipgloss.NewStyle().
			Foreground(successColor).
			Bold(true)

	metadataStyle = lipgloss.NewStyle().
			Foreground(subtleColor).
			Italic(true)

	messageHeaderStyle = lipgloss.NewStyle().
				Foreground(subtleColor).
				Italic(true)

	inputLabelStyle = lipgloss.NewStyle().
			Foreground(secondaryColor).
			Bold(true)

	footerStyle = lipgloss.NewStyle().
			Foreground(subtleColor).
			Italic(true)

	processingStyle = lipgloss.NewStyle().
			Foreground(warningColor).
			Italic(true)

	goodbyeStyle = lipgloss.NewStyle().
			Foreground(successColor).
			Bold(true).
			Padding(1)
)

func getSeverityStyle(severity string) lipgloss.Style {
	switch severity {
	case "high", "critical", "error":
		return errorStyle
	case "medium", "warning":
		return warningStyle
	case "low", "info":
		return infoStyle
	default:
		return subtleStyle
	}
}
