package helper

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"

	tea "github.com/charmbracelet/bubbletea"
)

type SearchResults []string

var ignoredDirs = map[string]struct{}{
    ".git":         {},
    "node_modules": {},
    ".svn":         {},
    ".hg":          {},
    "vendor":       {},
    "__pycache__":  {},
    ".cache":       {},
    ".vscode":      {},
    ".idea":        {},
    "target":       {}, 
    "build":        {},
    "dist":         {},
}

var sem = make(chan struct{}, 50)
const maxResults = 102

func PerformSearch(dir, query string) tea.Cmd {
	return func() tea.Msg {
		if query == "" {
			return SearchResults{}
		}

		query = strings.ToLower(query)

		var results []string
		var mu sync.Mutex
		var wg sync.WaitGroup
		var count atomic.Int32

		var search func(dir string)
		search = func(dir string) {
			defer wg.Done()

			if count.Load() >= maxResults {
				return  
			}

			sem <- struct{}{}
			entries, err := os.ReadDir(dir)
			<-sem

			if err != nil {
				return
			}

			var currResults [] string 

			for _, e := range entries {

				if count.Load() >= maxResults {
					break 
				}
				name := e.Name()
				path := filepath.Join(dir, name)

				if len(name) > 0 && name[0] == '.' {
					continue 
				}

				if e.IsDir() {
					if _, ok := ignoredDirs[name]; !ok {
						wg.Add(1)
						go search(path)
						continue 
					}
				}

				if fuzzyMatch(strings.ToLower(name), query) {
					currResults = append(currResults, path)
				}
			}

			if len(currResults) > 0 {
				mu.Lock()
				results = append(results, currResults...)
				count.Store(int32(len(results)))
				mu.Unlock()
			}
		}

		wg.Add(1)
		go search(dir)
		wg.Wait()

		if len(results) > maxResults {
			results = results[:maxResults]
		}

		return SearchResults(results)
	}
}

func fuzzyMatch(name, query string) bool {
	qi := 0
	for i := 0; i < len(name) && qi < len(query); i++ {
		if name[i] == query[qi] {
			qi++
		}
	}
	return qi == len(query)
}

