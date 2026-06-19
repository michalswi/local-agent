package tui

import (
	"github.com/charmbracelet/lipgloss"
)

var (
	// Apple Dark Mode color palette
	appleBlue   = lipgloss.Color("#0A84FF")
	appleGreen  = lipgloss.Color("#30D158")
	appleRed    = lipgloss.Color("#FF453A")
	appleOrange = lipgloss.Color("#FF9F0A")
	applePurple = lipgloss.Color("#BF5AF2")
	appleGray   = lipgloss.Color("#8E8E93")
	appleDark   = lipgloss.Color("#3A3A3C")
	appleText   = lipgloss.Color("#F2F2F7")
	appleDim    = lipgloss.Color("#636366")

	// Semantic color aliases
	primaryColor   = appleBlue
	secondaryColor = applePurple
	successColor   = appleGreen
	warningColor   = appleOrange
	errorColor     = appleRed
	subtleColor    = appleGray

	// ── Non-interactive view styles ─────────────────────────────

	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(appleText)

	sectionStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(appleBlue).
			MarginTop(1)

	labelStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(appleGray)

	subtleStyle = lipgloss.NewStyle().
			Foreground(appleDim)

	helpStyle = lipgloss.NewStyle().
			Foreground(appleDim).
			MarginTop(1)

	progressStyle = lipgloss.NewStyle().
			Foreground(appleGreen)

	errorStyle = lipgloss.NewStyle().
			Foreground(appleRed).
			Bold(true)

	warningStyle = lipgloss.NewStyle().
			Foreground(appleOrange).
			Bold(true)

	successStyle = lipgloss.NewStyle().
			Foreground(appleGreen).
			Bold(true)

	infoStyle = lipgloss.NewStyle().
			Foreground(appleBlue).
			Bold(true)

	// ── Interactive mode styles ──────────────────────────────────

	// Header bar — rounded card
	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(appleText).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(appleDark).
			Padding(0, 2)

	// Chat bubble — user (right-aligned, blue border)
	userBubbleStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(appleBlue).
			Padding(0, 1).
			Foreground(appleText)

	// Chat bubble — assistant (left-aligned, dim border)
	assistantBubbleStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(appleDark).
				Padding(0, 1).
				Foreground(appleText)

	// Kept for non-bubble fallback / non-interactive use
	userHeaderStyle = lipgloss.NewStyle().
			Foreground(appleBlue).
			Bold(true)

	assistantHeaderStyle = lipgloss.NewStyle().
				Foreground(applePurple).
				Bold(true)

	userMessageStyle = lipgloss.NewStyle().
				Foreground(appleText)

	assistantMessageStyle = lipgloss.NewStyle().
				Foreground(appleText)

	fileHeaderStyle = lipgloss.NewStyle().
			Foreground(appleGreen).
			Bold(true)

	metadataStyle = lipgloss.NewStyle().
			Foreground(appleDim).
			Italic(true)

	messageHeaderStyle = lipgloss.NewStyle().
				Foreground(appleDim).
				Italic(true)

	// Input area prompt
	inputLabelStyle = lipgloss.NewStyle().
			Foreground(appleBlue).
			Bold(true)

	// Footer key hints
	footerStyle = lipgloss.NewStyle().
			Foreground(appleDim)

	processingStyle = lipgloss.NewStyle().
			Foreground(appleOrange).
			Italic(true)

	thinkingStyle = lipgloss.NewStyle().
			Foreground(applePurple).
			Bold(true).
			Italic(true)

	reasoningHeaderStyle = lipgloss.NewStyle().
				Foreground(applePurple).
				Bold(true)

	reasoningLineStyle = lipgloss.NewStyle().
				Foreground(appleGray).
				Italic(true)

	goodbyeStyle = lipgloss.NewStyle().
			Foreground(appleBlue).
			Bold(true).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(appleDark).
			Padding(1, 3).
			MarginTop(1)
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
