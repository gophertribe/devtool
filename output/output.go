package output

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

var (
	// Styles
	infoStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("39")).Bold(true)
	successStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("46")).Bold(true)
	errorStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true)
	stepStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Bold(true)
	dimStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
)

// Info prints an informational message (Blue)
func Info(msg string, args ...any) {
	fmt.Printf("%s %s\n", infoStyle.Render("ℹ"), fmt.Sprintf(msg, args...))
}

// Success prints a success message (Green)
func Success(msg string, args ...any) {
	fmt.Printf("%s %s\n", successStyle.Render("✔"), fmt.Sprintf(msg, args...))
}

// Error prints an error message (Red)
func Error(msg string, args ...any) {
	fmt.Printf("%s %s\n", errorStyle.Render("✖"), fmt.Sprintf(msg, args...))
}

// Step prints a step message (Orange/Yellow)
func Step(msg string, args ...any) {
	fmt.Printf("%s %s\n", stepStyle.Render("➜"), fmt.Sprintf(msg, args...))
}

// Debug prints a debug message (Dim/Grey) - usually handled by slog, but here for completeness if needed
func Debug(msg string, args ...any) {
	fmt.Printf("%s %s\n", dimStyle.Render("🐛"), dimStyle.Render(fmt.Sprintf(msg, args...)))
}
