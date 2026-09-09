package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

type History struct {
	path    string
	entries []string
	cursor  int
}

func newHistory(dbPath string) *History {
	historyPath := strings.TrimSuffix(dbPath, filepath.Ext(dbPath)) + "_history.json"
	entries := loadHistory(historyPath)
	return &History{
		path:    historyPath,
		entries: entries,
		cursor:  len(entries),
	}
}

func (h *History) Add(entry string) {
	if entry == "" {
		return
	}
	h.entries = append(h.entries, entry)
	h.cursor = len(h.entries)
	h.save()
}

func (h *History) Prev() string {
	if h.cursor > 0 {
		h.cursor--
	}
	if h.cursor < len(h.entries) {
		return h.entries[h.cursor]
	}
	return ""
}

func (h *History) Next() string {
	if h.cursor < len(h.entries) {
		h.cursor++
	}
	if h.cursor < len(h.entries) {
		return h.entries[h.cursor]
	}
	return ""
}

func (h *History) Clear() {
	h.entries = nil
	h.cursor = 0
	h.save()
}

func (h *History) save() {
	data, err := json.MarshalIndent(h.entries, "", "  ")
	if err != nil {
		return
	}
	os.WriteFile(h.path, data, 0644)
}

func loadHistory(path string) []string {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var entries []string
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil
	}
	return entries
}

func wrapHistory(entries []string, width int) string {
	if len(entries) == 0 {
		return ""
	}
	if width < 1 {
		width = 1
	}
	lines := strings.Split(lipgloss.NewStyle().Width(width).Render(strings.Join(entries, "\n\n")), "\n")
	for i := range lines {
		lines[i] = strings.TrimRight(lines[i], " ")
	}
	return strings.Join(lines, "\n")
}
