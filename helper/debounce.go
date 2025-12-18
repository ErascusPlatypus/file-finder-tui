package helper

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

type DebounceMsg struct {
	Query string 
	ID int 
}

func Debounce(query string, id int, duration time.Duration) tea.Cmd {
	return tea.Tick(duration, func(t time.Time) tea.Msg {
		return DebounceMsg{Query: query, ID: id}
	})
}