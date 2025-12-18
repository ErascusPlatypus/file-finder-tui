package main

import (
	"fmt"
	"os"
	"pro8_tui/helper"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	titleStyle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205"))
	resultStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("86"))
	selectedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#123456")).Background(lipgloss.Color("#666666")).Bold(true)
	viewportStyle = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("62"))
	rightStyle = lipgloss.NewStyle().MarginLeft(4)
)

var program *tea.Program

type model struct {
	textInput  textinput.Model
	results    []string
	searching  bool
	currentDir string

	cursor     int

	viewport   viewport.Model
	previewing bool

	searchID int 
	debounceID int 

	quitting bool 
}

func initialModel() model {
	ti := textinput.New()
	ti.Placeholder = "Type to search..."
	ti.Focus()
	ti.CharLimit = 64
	ti.Width = 40

	currDir, _ := os.Getwd()

	vp := viewport.New(70, 20)
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

func (m *model) clearAndQuit() (model, tea.Cmd) {
	helper.CancelSearch()
	m.results = [] string{}
	m.textInput.SetValue("")
	m.previewing = false
	m.searching = false 
	m.viewport.SetContent("")
	m.cursor = 0 
	m.quitting = true 
	
	return *m, tea.Tick(time.Millisecond*10, func(t time.Time) tea.Msg {
		return tea.QuitMsg{}
	})
} 

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	if _, ok := msg.(tea.QuitMsg); ok {
		return m, tea.Quit
	}

	if m.quitting {
		return m, nil 
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			if m.previewing {
				m.previewing = false
				m.viewport.SetContent("")
				return m, nil
			}
			return m.clearAndQuit()

		case "ctrl+c":
			return m.clearAndQuit()

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

	case helper.DebounceMsg:
		if msg.ID == m.debounceID && msg.Query != "" {
			m.searchID++ 
			return m, helper.PerformSearch(m.currentDir, msg.Query, program)
		}

		return m, nil 
	
	case helper.StreamResults:
		if msg.SearchID == m.searchID {
			m.results = append(m.results, msg.Path)
		}

		return m, nil 
	
	case helper.SearchDone:
		if msg.SearchID == m.searchID {
			m.searching = false 
		}

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
		helper.CancelSearch()

		m.results = [] string {} 
		m.cursor = 0 
		m.previewing = false 
		m.viewport.SetContent("")

		query := m.textInput.Value()
		if query == "" {
			m.searching = false 
			return m, cmd 
		}

		m.searching = true 
		m.debounceID++ 

		return m, tea.Batch(cmd, helper.Debounce(query, m.debounceID, 150*time.Millisecond))
	}

	return m, cmd
}

func (m model) View() string {
	if m.quitting {
		return "" 
	}
	var sb strings.Builder

	sb.WriteString(titleStyle.Render("File Finder"))
	sb.WriteString("\n\n")
	sb.WriteString(m.textInput.View())
	sb.WriteString("\n\n")

	var left strings.Builder
	leftPanelHeight := 15 
	lineCount := 0 

	if m.searching || len(m.results) > 0 {
		status := "" 
		if m.searching {
			status = " ( searching...)"
		}
		left.WriteString(fmt.Sprintf("Found %d results%s:\n", len(m.results), status))
		lineCount++ 

		displayCount := min(len(m.results), 15)
		for i := range displayCount {
			prefix := "  "
			style := resultStyle

			if i == m.cursor {
				prefix = "> "
				style = selectedStyle
			}

			left.WriteString(style.Render(fmt.Sprintf("%v", prefix + helper.ResFormat(m.results[i], 40))))
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

	sb.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, left.String(), rightStyle.Render(right.String())))

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
	program = tea.NewProgram(initialModel())
	if _, err := program.Run(); err != nil {
		fmt.Printf("Error: %v", err)
		os.Exit(1)
	}
}
