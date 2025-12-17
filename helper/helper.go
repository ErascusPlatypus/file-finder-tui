package helper

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/alecthomas/chroma/v2"
    "github.com/alecthomas/chroma/v2/formatters"
    "github.com/alecthomas/chroma/v2/lexers"
    "github.com/alecthomas/chroma/v2/styles"
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
)

type PreviewMsg struct {
	Content string
	Path string 
}

func LoadPreview(path string) tea.Cmd {
	return func() tea.Msg {
		ext := filepath.Ext(path)
		var data string 
		if ext == ".pdf" {
			d, err := PdfToText(path)
			if err != nil {
				return PreviewMsg{Content: "Unable to open file", Path: path}
			}

			data = d
		} else {
			d, err := os.ReadFile(path)
			if err != nil {
				return PreviewMsg{Content: "Unable to open file", Path: path}
			}

			data = string(d)
		}

		return PreviewMsg{Content: string(data), Path: path}
	}
}

func HighlightContent(code, filename string) string {
	lexer := lexers.Match(filename)
	if lexer == nil {
		lexer = lexers.Analyse(code)
	}
	if lexer == nil {
		lexer = lexers.Fallback
	}
	lexer = chroma.Coalesce(lexer)

	style := styles.Get("onedark")
	if style == nil {
		style = styles.Fallback
	}

	formatter := formatters.Get("terminal256")
	if formatter == nil {
		formatter = formatters.Fallback
	}

	iterator, err := lexer.Tokenise(nil, code)
	if err != nil {
		return code
	}

	var buf bytes.Buffer
	err = formatter.Format(&buf, style, iterator)
	if err != nil {
		return code
	}

	return buf.String()
}

func ResFormat(path string, max int) string {
	if len(path) <= max {
		return path 
	}

	return "..." + path[len(path)-max:]
}

func PdfToText(path string) (string, error) {
	cmd := exec.Command("pdftotext", path, "-")
	out, err := cmd.Output()

	if err != nil {
		return "", err 
	}

	return string(out), nil 
}