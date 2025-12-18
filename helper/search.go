package helper

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"

	tea "github.com/charmbracelet/bubbletea"
)

type SearchResults []string

type StreamResults struct {
	Path string 
	SearchID int
}

type SearchDone struct {
	SearchID int 
	Total int 
}

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

const maxResults = 150
const maxConcurrency = 50 

var (
	cancelMutex sync.Mutex
	cancelFunc context.CancelFunc
	searchIDGen atomic.Int32
)

func CancelSearch() {
	cancelMutex.Lock()
	defer cancelMutex.Unlock()

	if cancelFunc != nil {
		cancelFunc()
		cancelFunc = nil 
	}
}

func PerformSearch(dir, query string, program *tea.Program) tea.Cmd {
	return func() tea.Msg {
		if query == "" {
			return SearchDone{SearchID: 0, Total: 0}
		}

		CancelSearch()

		cancelMutex.Lock()
		ctx, cancel := context.WithCancel(context.Background())
		cancelFunc = cancel
		searchID := int(searchIDGen.Add(1))
		cancelMutex.Unlock()

		query = strings.ToLower(query)

		go func() {
			var wg sync.WaitGroup
			var count atomic.Int32
			var sem = make(chan struct{}, maxConcurrency)

			var search func(dir string) 
			search = func(dir string) {
				defer wg.Done()

				select {
				case <-ctx.Done():
					return 
				default:
				}

				if count.Load() >= maxResults {
					return
				}

				sem <- struct{}{}
				entries, err := os.ReadDir(dir)
				<- sem 

				if err != nil {
					return  
				}

				for _, e := range entries {
					select {
					case <- ctx.Done():
						return 
					default:
					}

					if count.Load() >= maxResults {
						return  
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
						}

						continue
					}

					if fuzzyMatch(strings.ToLower(name), query) {
						if count.Add(1) <= maxResults {
							program.Send(StreamResults{Path: path, SearchID: searchID})
						}
					}
				}
			}

			wg.Add(1)
			go search(dir)
			wg.Wait()

			program.Send(SearchDone{SearchID: searchID, Total: int(count.Load())})
		}()

		return nil 
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

