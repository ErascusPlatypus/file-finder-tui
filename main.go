package main

import (
	"fmt"
	"os"
	"strings"
	"pro8_tui/helper"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

var (
	titleStyle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205"))
	resultStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("86"))
	selectedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#123456")).Background(lipgloss.Color("#666666")).Bold(true)
	viewportStyle = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("62"))
)

type model struct {
	textInput  textinput.Model
	results    []string
	searching  bool
	currentDir string

	cursor     int

	viewport   viewport.Model
	previewing bool
}

func initialModel() model {
	ti := textinput.New()
	ti.Placeholder = "Type to search..."
	ti.Focus()
	ti.CharLimit = 64
	ti.Width = 40

	currDir, _ := os.Getwd()

	vp := viewport.New(60, 20)
	vp.Style = viewportStyle

	return model{
		textInput:  ti,
		results:    []string{},
		currentDir: currDir,
		cursor:     0,
		viewport:   vp,
	}
}

func (m model) Init() tea.Cmd {
	return textinput.Blink
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			if m.previewing {
				m.previewing = false
				m.viewport.SetContent("")
				return m, nil
			}
			return m, tea.Quit

		case "ctrl+c":
			return m, tea.Quit

		case "down":
			if m.previewing {
				m.viewport.ScrollDown(1)
			} else if m.cursor < len(m.results)-1 {
				m.cursor++
			}
			return m, nil

		case "up":
			if m.previewing {
				m.viewport.ScrollUp(1)
			} else if m.cursor > 0 {
				m.cursor--
			}
			return m, nil

		case "pgdown", "ctrl+d":
			if m.previewing {
				m.viewport.ScrollDown(5)
			}
			return m, nil

		case "pgup", "ctrl+u":
			if m.previewing {
				m.viewport.ScrollUp(5)
			}
			return m, nil

		case "enter":
			if m.previewing || len(m.results) == 0 {
				return m, nil
			}

			path := m.results[m.cursor]
			info, err := os.Stat(path)
			if err != nil || info.IsDir() {
				return m, nil
			}

			m.previewing = true
			return m, helper.LoadPreview(path)
		}

	case helper.SearchResults:
		m.results = msg
		m.cursor = 0
		m.previewing = false
		m.viewport.SetContent("")
		m.searching = false
		return m, nil

	case helper.PreviewMsg:
		highlightedContent := helper.HighlightContent(msg.Content, msg.Path)
		m.viewport.SetContent(highlightedContent)
		m.viewport.GotoTop()
		return m, nil
	}

	if m.previewing {
		return m, nil
	}

	prev := m.textInput.Value()
	m.textInput, cmd = m.textInput.Update(msg)

	if m.textInput.Value() != prev {
		m.searching = true
		return m, tea.Batch(cmd,
			helper.PerformSearch(m.currentDir, m.textInput.Value()),
		)
	}

	return m, cmd
}

func (m model) View() string {
	var sb strings.Builder

	sb.WriteString(titleStyle.Render("File Finder"))
	sb.WriteString("\n\n")
	sb.WriteString(m.textInput.View())
	sb.WriteString("\n\n")

	var left strings.Builder
	leftPanelHeight := 15 
	lineCount := 0 

	if m.searching {
		left.WriteString("Searching...")
		lineCount = 1 
	} else if len(m.results) > 0 {
		left.WriteString(fmt.Sprintf("Found %d results:\n", len(m.results)))
		lineCount++ 

		displayCount := min(len(m.results), 15)
		for i := range displayCount {
			prefix := "  "
			style := resultStyle

			if i == m.cursor {
				prefix = "> "
				style = selectedStyle
			}

			left.WriteString(style.Render(prefix + m.results[i]))
			left.WriteString("\n")
			lineCount++
		}
		if len(m.results) > 15 {
			left.WriteString(fmt.Sprintf("  ... and %d more\n", len(m.results)-15))
			lineCount++
		}
	} else if m.textInput.Value() != "" {
		left.WriteString("No results found")
		lineCount = 1 
	}

	for range (leftPanelHeight - lineCount) {
		left.WriteString("\n")
	}

	var right strings.Builder

	if m.previewing {
		right.WriteString(m.viewport.View())
	} else {
		right.WriteString("  Press Enter to preview file")
		for i := 1; i < leftPanelHeight ; i++ {
			right.WriteString("\n")
		}
	}

	sb.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, left.String(), right.String()))

	sb.WriteString("\n\nEsc: ")
	if m.previewing {
		sb.WriteString("close preview")
	} else {
		sb.WriteString("quit")
	}
	sb.WriteString(" | ↑/↓: navigate | Enter: preview")

	return sb.String()
}

func main() {
	p := tea.NewProgram(initialModel())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v", err)
		os.Exit(1)
	}
}
