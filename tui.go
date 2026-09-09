package main

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type mode int

const (
	modeSQL mode = iota
	modeCommand
)

type model struct {
	db            *sql.DB
	history       *History
	textInput     textinput.Model
	queryVP       viewport.Model
	historyVP     viewport.Model
	mode          mode
	queryOutput   string
	queryLines    []string
	scrollX       int
	lastError     string
	confirmBurn   bool
	confirmStep   int
	confirmMsg    string
	burnTarget    string
	width         int
	height        int
	ready         bool
}

func newModel(dbPath string) model {
	db, err := openDB(dbPath)
	if err != nil {
		fmt.Fprintf(nil, "error: %v\n", err)
	}

	ti := textinput.New()
	ti.Placeholder = "~>"
	ti.Focus()
	ti.CharLimit = 4096

	h := newHistory(dbPath)

	return model{
		db:        db,
		history:   h,
		textInput: ti,
		mode:      modeSQL,
	}
}

func (m model) Init() tea.Cmd {
	return textinput.Blink
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		leftWidth := int(float64(m.width) * 0.7)
		rightWidth := m.width - leftWidth
		inputHeight := 3
		paneHeight := m.height - inputHeight - 2

		if !m.ready {
			m.queryVP = viewport.New(leftWidth-2, paneHeight)
			m.historyVP = viewport.New(rightWidth-2, paneHeight)
			m.ready = true
		} else {
			m.queryVP.Width = leftWidth - 2
			m.queryVP.Height = paneHeight
			m.historyVP.Width = rightWidth - 2
			m.historyVP.Height = paneHeight
		}

		m.queryVP.SetContent(m.sliceQuery())
		m.historyVP.SetContent(strings.Join(m.history.entries, "\n"))

		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit

		case "ctrl+j":
			m.historyVP.LineDown(1)
			return m, nil
		case "ctrl+k":
			m.historyVP.LineUp(1)
			return m, nil
		case "ctrl+shift+j", "ctrl+J":
			m.queryVP.LineDown(1)
			return m, nil
		case "ctrl+shift+k", "ctrl+K":
			m.queryVP.LineUp(1)
			return m, nil
		case "ctrl+shift+right", "ctrl+right":
			m.scrollX += 4
			m.queryVP.SetContent(m.sliceQuery())
			return m, nil
		case "ctrl+shift+left", "ctrl+left":
			if m.scrollX > 0 {
				m.scrollX -= 4
				if m.scrollX < 0 {
					m.scrollX = 0
				}
				m.queryVP.SetContent(m.sliceQuery())
			}
			return m, nil

case "enter":
			input := m.textInput.Value()
			if input == "" {
				return m, nil
			}

			if m.confirmBurn {
				if strings.EqualFold(input, "y") {
					if m.confirmStep == 1 {
						m.confirmStep = 2
						m.confirmMsg = fmt.Sprintf("burn %s permanently? type 'y' again to confirm or 'n' to cancel", m.burnTarget)
						m.textInput.SetValue("")
						return m, nil
					}
					switch m.burnTarget {
					case "db":
						if m.db != nil {
							if err := burnDB(m.db); err != nil {
								m.lastError = "burn db failed: " + err.Error()
								m.queryOutput = errorStyle.Render(m.lastError)
							} else {
								m.lastError = ""
								m.queryOutput = "database burned"
							}
						} else {
							m.lastError = "no database connection"
							m.queryOutput = errorStyle.Render(m.lastError)
						}
					case "history":
						m.history.Clear()
						m.historyVP.SetContent("")
						m.lastError = ""
						m.queryOutput = "history burned"
					}
				} else {
					m.lastError = ""
					m.queryOutput = "burn cancelled"
				}
				m.queryLines = strings.Split(m.queryOutput, "\n")
				m.scrollX = 0
				m.queryVP.SetContent(m.sliceQuery())
				m.confirmBurn = false
				m.confirmStep = 0
				m.confirmMsg = ""
				m.burnTarget = ""
				m.textInput.SetValue("")
				return m, nil
			}

			result := handleCommand(input)
			switch result.action {
			case "quit":
				return m, tea.Quit
			case "burn":
				m.confirmBurn = true
				m.confirmStep = 1
				m.burnTarget = result.target
				m.confirmMsg = fmt.Sprintf("burn %s? this will permanently destroy it. type 'y' to confirm or 'n' to cancel", result.target)
				m.textInput.SetValue("")
				return m, nil
			case "clear":
				m.queryOutput = ""
				m.queryLines = nil
				m.scrollX = 0
				m.queryVP.SetContent("")
				m.lastError = ""
			case "error":
				m.lastError = result.text
			case "":
				m.history.Add(input)
				m.historyVP.SetContent(strings.Join(m.history.entries, "\n"))

				if m.db != nil {
					cols, rows, err := executeQueryWithRows(m.db, input)
					if err != nil {
						m.lastError = err.Error()
						m.queryOutput = errorStyle.Render(m.lastError)
					} else if cols == nil {
						m.lastError = ""
						m.queryOutput = successStyle.Render("OK")
					} else {
						m.lastError = ""
						m.queryOutput = formatResults(cols, rows)
					}
				} else {
					m.lastError = "no database connection"
					m.queryOutput = errorStyle.Render(m.lastError)
				}
				m.queryLines = strings.Split(m.queryOutput, "\n")
				m.scrollX = 0
				m.queryVP.SetContent(m.sliceQuery())
			}

			m.textInput.SetValue("")
			return m, nil

		case "up":
			if m.mode == modeSQL && !m.confirmBurn {
				prev := m.history.Prev()
				m.textInput.SetValue(prev)
			}
			return m, nil

		case "down":
			if m.mode == modeSQL && !m.confirmBurn {
				next := m.history.Next()
				m.textInput.SetValue(next)
			}
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

func (m model) sliceQuery() string {
	if m.scrollX == 0 || len(m.queryLines) == 0 {
		return m.queryOutput
	}
	visible := make([]string, len(m.queryLines))
	leftWidth := int(float64(m.width) * 0.7)
	lineWidth := leftWidth - 6
	for i, line := range m.queryLines {
		end := m.scrollX + lineWidth
		if end > len(line) {
			end = len(line)
		}
		if m.scrollX < len(line) {
			visible[i] = line[m.scrollX:end]
		} else {
			visible[i] = ""
		}
	}
	return strings.Join(visible, "\n")
}

func (m model) View() string {
	if !m.ready {
		return "  initializing..."
	}

	leftWidth := int(float64(m.width) * 0.7)
	rightWidth := m.width - leftWidth
	inputHeight := 3
	paneHeight := m.height - inputHeight - 2

	queryTitle := titleStyle.Render(" Live Query ")
	historyTitle := titleStyle.Render(" History ")

	queryPane := queryPaneStyle.
		Width(leftWidth - 2).
		Height(paneHeight).
		Render(queryTitle + "\n" + m.queryVP.View())

	historyPane := historyPaneStyle.
		Width(rightWidth - 2).
		Height(paneHeight).
		Render(historyTitle + "\n" + m.historyVP.View())

	panes := lipgloss.JoinHorizontal(lipgloss.Top, queryPane, historyPane)

	prompt := promptStyle.Render("~> ") + m.textInput.View()

	var bottom string
	switch {
	case m.confirmBurn:
		bottom = confirmStyle.Render(m.confirmMsg) + "\n" + prompt
	case m.lastError != "":
		bottom = errorStyle.Render(m.lastError) + "\n" + prompt
	default:
		bottom = prompt
	}

	return panes + "\n" + bottom
}
