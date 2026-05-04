package picker

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	selectedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("212")).Bold(true)
	dimStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	queryStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("86"))
	headerStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("99"))
)

// Item is an entry in the picker list.
type Item struct {
	ID      int
	Display string // shown in list
	Sub     string // shown as subtitle
	Extra   string // hidden, searched but not displayed
}

type model struct {
	items      []Item
	filtered   []Item
	cursor     int
	offset     int
	query      string
	chosen     *Item
	quit       bool
	termHeight int
}

func (m model) maxVisible() int {
	if m.termHeight <= 0 {
		return 20
	}
	// header=3, footer=2, each item=2 lines
	n := (m.termHeight - 5) / 2
	if n < 1 {
		return 1
	}
	return n
}

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.termHeight = msg.Height
		return m, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			m.quit = true
			return m, tea.Quit
		case "enter":
			if len(m.filtered) > 0 {
				c := m.filtered[m.cursor]
				m.chosen = &c
			}
			return m, tea.Quit
		case "up", "ctrl+p":
			if m.cursor > 0 {
				m.cursor--
				if m.cursor < m.offset {
					m.offset--
				}
			}
		case "down", "ctrl+n":
			if m.cursor < len(m.filtered)-1 {
				m.cursor++
				if m.cursor >= m.offset+m.maxVisible() {
					m.offset++
				}
			}
		case "backspace":
			if len(m.query) > 0 {
				m.query = m.query[:len(m.query)-1]
				m.refilter()
			}
		default:
			if len(msg.Runes) > 0 {
				m.query += string(msg.Runes)
				m.refilter()
			}
		}
	}
	return m, nil
}

func (m *model) refilter() {
	m.cursor = 0
	m.offset = 0
	if m.query == "" {
		m.filtered = m.items
		return
	}
	q := strings.ToLower(m.query)
	var out []Item
	for _, it := range m.items {
		if strings.Contains(strings.ToLower(it.Display), q) ||
			strings.Contains(strings.ToLower(it.Sub), q) ||
			strings.Contains(strings.ToLower(it.Extra), q) {
			out = append(out, it)
		}
	}
	m.filtered = out
}

func (m model) View() string {
	var b strings.Builder

	b.WriteString(headerStyle.Render("  pwf start") + "\n")
	b.WriteString(queryStyle.Render("  > " + m.query + "▋") + "\n\n")

	if len(m.filtered) == 0 {
		b.WriteString(dimStyle.Render("  no matches") + "\n")
	}

	end := m.offset + m.maxVisible()
	if end > len(m.filtered) {
		end = len(m.filtered)
	}
	if m.offset > 0 {
		b.WriteString(dimStyle.Render(fmt.Sprintf("  ... %d above", m.offset)) + "\n")
	}
	for i := m.offset; i < end; i++ {
		it := m.filtered[i]
		line := fmt.Sprintf("  %s", it.Display)
		if i == m.cursor {
			line = selectedStyle.Render("> " + it.Display)
		}
		b.WriteString(line + "\n")
		if it.Sub != "" {
			b.WriteString(dimStyle.Render("    "+it.Sub) + "\n")
		}
	}
	if end < len(m.filtered) {
		b.WriteString(dimStyle.Render(fmt.Sprintf("  ... %d below", len(m.filtered)-end)) + "\n")
	}

	b.WriteString("\n" + dimStyle.Render("  ↑↓ navigate · enter select · esc quit"))
	return b.String()
}

// Run launches the interactive picker and returns the chosen item (nil if cancelled).
func Run(items []Item) (*Item, error) {
	m := model{
		items:    items,
		filtered: items,
	}
	p := tea.NewProgram(m)
	result, err := p.Run()
	if err != nil {
		return nil, err
	}
	final := result.(model)
	if final.quit || final.chosen == nil {
		return nil, nil
	}
	return final.chosen, nil
}
