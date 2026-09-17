package main

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
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
	db          *sql.DB
	history     *History
	textInput   textinput.Model
	queryVP     viewport.Model
	historyVP   viewport.Model
	mode        mode
	queryOutput string
	queryLines  []string
	scrollX     int
	lastError   string
	confirmBurn bool
	confirmStep int
	confirmMsg  string
	burnTarget  string
	width       int
	height      int
	ready       bool
	stacked     bool
	previewRows int
}

func newModel(dbPath string) model {
	db, err := openDB(dbPath)
	if err != nil {
		fmt.Fprintf(nil, "error: %v\n", err)
	}

	ti := textinput.New()
	ti.Placeholder = ""
	ti.Prompt = ""
	ti.Focus()
	ti.CharLimit = 4096

	km := textinput.DefaultKeyMap
	km.DeleteWordBackward = key.NewBinding(key.WithKeys("alt+backspace", "ctrl+w", "ctrl+backspace"))
	km.DeleteWordForward = key.NewBinding(key.WithKeys("alt+delete", "alt+d", "ctrl+delete"))
	ti.KeyMap = km

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

		m.stacked = m.width < 100
		m.previewRows = m.previewRowCount()

		return m.syncLayout(), nil

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
		case "shift+up":
			m.historyVP.LineUp(1)
			return m, nil
		case "shift+down":
			m.historyVP.LineDown(1)
			return m, nil
		case "ctrl+shift+j", "ctrl+J", "ctrl+shift+down":
			m.queryVP.LineDown(1)
			return m, nil
		case "ctrl+shift+k", "ctrl+K", "ctrl+shift+up":
			m.queryVP.LineUp(1)
			return m, nil
		case "ctrl+shift+left":
			if m.scrollX > 0 {
				m.scrollX -= 4
				if m.scrollX < 0 {
					m.scrollX = 0
				}
				m.queryVP.SetContent(m.sliceQuery())
			}
			return m, nil
		case "ctrl+shift+right":
			m.scrollX += 4
			m.queryVP.SetContent(m.sliceQuery())
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
						return m.refreshPreview(), nil
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
				return m.refreshPreview(), nil
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
				return m.refreshPreview(), nil
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
				wasAtBottom := m.historyVP.AtBottom()
				m.historyVP.SetContent(wrapHistory(m.history.entries, m.historyPaneWidth()))
				if wasAtBottom {
					m.historyVP.GotoBottom()
				}

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
			m.mode = modeSQL
			return m.refreshPreview(), nil

		case "up":
			if m.mode == modeSQL && !m.confirmBurn {
				prev := m.history.Prev()
				m.textInput.SetValue(prev)
				m.textInput.SetCursor(len([]rune(prev)))
			}
			return m.refreshPreview(), nil

		case "down":
			if m.mode == modeSQL && !m.confirmBurn {
				next := m.history.Next()
				m.textInput.SetValue(next)
				m.textInput.SetCursor(len([]rune(next)))
			}
			return m.refreshPreview(), nil
		}
	}

	var cmd tea.Cmd
	m.textInput, cmd = m.textInput.Update(msg)
	m = m.refreshPreview()
	if strings.HasPrefix(m.textInput.Value(), "/") {
		m.mode = modeCommand
	} else {
		m.mode = modeSQL
	}
	return m, cmd
}

func (m model) layoutExtra() int {
	if m.stacked {
		return 5
	}
	return 3
}

func (m model) paneHeight() int {
	return max(6, m.height-m.previewRows-3-m.layoutExtra())
}

func (m model) syncLayout() model {
	paneHeight := m.paneHeight()
	queryW, queryH, historyW, historyH := m.paneSize(paneHeight)

	if w := m.width - lipgloss.Width(m.promptIcon()) - 1; w > 0 {
		m.textInput.Width = w
	}

	queryContentH := queryH - 3
	if queryContentH < 1 {
		queryContentH = 1
	}
	historyContentH := historyH - 3
	if historyContentH < 1 {
		historyContentH = 1
	}

	newReady := !m.ready
	if newReady {
		m.queryVP = viewport.New(queryW, queryContentH)
		m.historyVP = viewport.New(historyW, historyContentH)
		m.ready = true
	} else {
		m.queryVP.Width = queryW
		m.queryVP.Height = queryContentH
		m.historyVP.Width = historyW
		m.historyVP.Height = historyContentH
	}

	m.queryVP.SetContent(m.sliceQuery())
	m.historyVP.SetContent(wrapHistory(m.history.entries, m.historyPaneWidth()))

	if newReady {
		m.historyVP.GotoBottom()
	}

	return m
}

func (m model) refreshPreview() model {
	rows := m.previewRowCount()
	if rows != m.previewRows {
		m.previewRows = rows
		return m.syncLayout()
	}
	return m
}

func (m model) previewRowCount() int {
	w := m.width - 3
	value := m.textInput.Value()
	if value == "" || w < 1 || strings.HasPrefix(value, "/") || lipgloss.Width(value) <= w {
		return 0
	}
	rows := len(strings.Split(lipgloss.NewStyle().Width(w).Render(value), "\n"))
	if cap := m.height - m.layoutExtra() - 9; rows > cap {
		return max(0, cap)
	}
	return rows
}

func (m model) queryPreview() string {
	if m.previewRows < 1 {
		return ""
	}
	return mutedStyle.Render(lipgloss.NewStyle().Width(m.width - 3).Render(m.textInput.Value()))
}

func (m model) paneSize(paneHeight int) (int, int, int, int) {
	if m.stacked {
		queryH := int(float64(paneHeight) * 0.7)
		return m.width - 2, queryH, m.width - 2, paneHeight - queryH
	}
	leftWidth := int(float64(m.width) * 0.7)
	rightWidth := m.width - leftWidth
	return leftWidth - 2, paneHeight, rightWidth - 2, paneHeight
}

func (m model) historyPaneWidth() int {
	if m.stacked {
		return m.width - 4
	}
	left := int(float64(m.width) * 0.7)
	return m.width - left - 3
}

func (m model) sliceQuery() string {
	if m.scrollX == 0 || len(m.queryLines) == 0 {
		return m.queryOutput
	}
	visible := make([]string, len(m.queryLines))
	lineWidth := m.width - 6
	if !m.stacked {
		leftWidth := int(float64(m.width) * 0.7)
		lineWidth = leftWidth - 6
	}
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

func (m model) commandPreview() string {
	matches := filterCommands(m.textInput.Value())
	if len(matches) == 0 {
		return mutedStyle.Render("no matching command")
	}
	lines := make([]string, len(matches))
	for i, c := range matches {
		lines[i] = m.highlightMatch(c)
	}
	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

func (m model) highlightMatch(cmd string) string {
	input := strings.ToLower(strings.TrimSpace(m.textInput.Value()))
	lower := strings.ToLower(cmd)
	idx := strings.Index(lower, input)
	if idx < 0 {
		return promptStyle.Render(cmd)
	}
	rendered := cmd[:idx] + matchedStyle.Render(cmd[idx:idx+len(input)]) + cmd[idx+len(input):]
	return promptStyle.Render(rendered)
}

func (m model) promptIcon() string {
	return promptStyle.Render("❯ ")
}

func (m model) View() string {
	if !m.ready {
		return "  initializing..."
	}

	paneHeight := m.paneHeight()
	queryW, queryH, historyW, historyH := m.paneSize(paneHeight)

	queryTitle := titleStyle.Render(" Live Query ")
	historyTitle := titleStyle.Render(" History ")

	queryPane := queryPaneStyle.
		Width(queryW).
		Height(queryH).
		Render(queryTitle + "\n" + m.queryVP.View())

	historyPane := historyPaneStyle.
		Width(historyW).
		Height(historyH).
		Render(historyTitle + "\n" + m.historyVP.View())

	var panes string
	if m.stacked {
		panes = lipgloss.JoinVertical(lipgloss.Top, queryPane, historyPane)
	} else {
		panes = lipgloss.JoinHorizontal(lipgloss.Top, queryPane, historyPane)
	}

	prompt := m.promptIcon() + m.textInput.View()

	var bottom string
	switch {
	case m.confirmBurn:
		bottom = confirmStyle.Render(m.confirmMsg) + "\n" + prompt
	case m.mode == modeCommand:
		bottom = m.commandPreview() + "\n" + prompt
	case m.lastError != "":
		bottom = errorStyle.Render(m.lastError) + "\n" + prompt
	case m.previewRows > 0:
		bottom = m.queryPreview() + "\n" + prompt
	default:
		bottom = prompt
	}

	return panes + "\n" + bottom
}
