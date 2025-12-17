package helper

import (
	"os"
	"path/filepath"
	"strings"
	"sync"

	tea "github.com/charmbracelet/bubbletea"
)

type SearchResults []string

var sem = make(chan struct{}, 50)

func PerformSearch(dir, query string) tea.Cmd {
	return func() tea.Msg {
		if query == "" {
			return SearchResults{}
		}

		var results []string
		var mu sync.Mutex
		var wg sync.WaitGroup

		var search func(dir string)
		search = func(dir string) {
			defer wg.Done()

			sem <- struct{}{}
			defer func() { <-sem }()

			entries, err := os.ReadDir(dir)
			if err != nil {
				return
			}

			for _, e := range entries {
				path := filepath.Join(dir, e.Name())

				if fuzzyMatch(e.Name(), query) {
					mu.Lock()
					results = append(results, path)
					mu.Unlock()
				}

				if e.IsDir() {
					wg.Add(1)
					go search(path)
				}
			}
		}

		wg.Add(1)
		go search(dir)
		wg.Wait()

		return SearchResults(results)
	}
}

func fuzzyMatch(name, query string) bool {
	name = strings.ToLower(name)
	query = strings.ToLower(query)

	qi := 0
	for i := 0; i < len(name) && qi < len(query); i++ {
		if name[i] == query[qi] {
			qi++
		}
	}
	return qi == len(query)
}

