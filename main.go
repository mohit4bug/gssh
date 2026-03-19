package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	sTitle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("212"))
	sGreen  = lipgloss.NewStyle().Foreground(lipgloss.Color("82")).Bold(true)
	sDim    = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	sAccent = lipgloss.NewStyle().Foreground(lipgloss.Color("212"))
	sBold   = lipgloss.NewStyle().Bold(true)
	sErr    = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	sHelp   = lipgloss.NewStyle().Foreground(lipgloss.Color("238"))
	sKey    = lipgloss.NewStyle().Foreground(lipgloss.Color("195"))
)

type screen int

const (
	screenList screen = iota
	screenAdd
	screenKey
	screenConfirm
	screenEdit
)

type model struct {
	screen   screen
	accounts []Account
	cursor   int

	inputs  [3]textinput.Model
	focused int

	pubKey string
	err    string
}

func initial() model {
	accs, _ := loadAccounts()

	cursor := 0
	for i, a := range accs {
		if a.Active {
			cursor = i
		}
	}

	placeholders := []string{"personal", "johndoe", "john@example.com"}
	var inputs [3]textinput.Model
	for i := range inputs {
		t := textinput.New()
		t.Placeholder = placeholders[i]
		t.Prompt = ""
		t.CharLimit = 80
		inputs[i] = t
	}
	inputs[0].Focus()

	return model{accounts: accs, cursor: cursor, inputs: inputs}
}

type (
	msgUpdated struct{ accounts []Account }
	msgKey     struct {
		accounts []Account
		pubKey   string
	}
	msgErr struct{ err error }
)

func (m model) Init() tea.Cmd { return textinput.Blink }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.KeyMsg:
		switch m.screen {
		case screenList:
			return m.handleList(msg)
		case screenAdd:
			return m.handleAdd(msg)
		case screenEdit:
			return m.handleEdit(msg)
		case screenConfirm:
			return m.handleConfirm(msg)
		case screenKey:
			if msg.String() == "ctrl+c" {
				return m, tea.Quit
			}
			m.screen = screenList
			return m, nil
		}

	case msgUpdated:
		m.accounts = msg.accounts
		if m.cursor >= len(m.accounts) && m.cursor > 0 {
			m.cursor = len(m.accounts) - 1
		}
		m.screen = screenList
		m.err = ""
		return m, nil

	case msgKey:
		m.accounts = msg.accounts
		m.cursor = len(msg.accounts) - 1
		m.pubKey = msg.pubKey
		m.screen = screenKey
		m.err = ""
		return m, nil

	case msgErr:
		m.err = msg.err.Error()
		m.screen = screenList
		return m, nil
	}

	if m.screen == screenAdd || m.screen == screenEdit {
		var cmd tea.Cmd
		m.inputs[m.focused], cmd = m.inputs[m.focused].Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m model) handleList(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		return m, tea.Quit
	case "up":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down":
		if m.cursor < len(m.accounts)-1 {
			m.cursor++
		}
	case "enter", " ":
		if len(m.accounts) > 0 {
			return m, cmdActivate(m.accounts, m.cursor)
		}
	case "a":
		for i := range m.inputs {
			m.inputs[i].SetValue("")
			m.inputs[i].Blur()
		}
		m.inputs[0].Focus()
		m.focused = 0
		m.err = ""
		m.screen = screenAdd
	case "e":
		if len(m.accounts) > 0 {
			a := m.accounts[m.cursor]
			m.inputs[0].SetValue(a.Alias)
			m.inputs[0].Focus()
			m.focused = 0
			m.err = ""
			m.screen = screenEdit
		}
	case "d":
		if len(m.accounts) > 0 {
			m.screen = screenConfirm
		}
	case "k":
		if len(m.accounts) > 0 {
			key, err := readPubKey(m.accounts[m.cursor].KeyPath)
			if err != nil {
				m.err = err.Error()
			} else {
				m.pubKey = key
				m.screen = screenKey
			}
		}
	}
	return m, nil
}

func (m model) handleAdd(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit
	case "esc":
		m.screen = screenList
		m.err = ""
		return m, nil
	case "tab", "down":
		m.inputs[m.focused].Blur()
		m.focused = (m.focused + 1) % 3
		m.inputs[m.focused].Focus()
		return m, nil
	case "shift+tab", "up":
		m.inputs[m.focused].Blur()
		m.focused = (m.focused + 2) % 3
		m.inputs[m.focused].Focus()
		return m, nil
	case "enter":
		if m.focused < 2 {
			m.inputs[m.focused].Blur()
			m.focused++
			m.inputs[m.focused].Focus()
			return m, nil
		}
		alias := strings.TrimSpace(m.inputs[0].Value())
		username := strings.TrimSpace(m.inputs[1].Value())
		email := strings.TrimSpace(m.inputs[2].Value())
		if alias == "" || username == "" || email == "" {
			m.err = "all fields are required"
			return m, nil
		}
		return m, cmdAdd(m.accounts, alias, username, email)
	}

	var cmd tea.Cmd
	m.inputs[m.focused], cmd = m.inputs[m.focused].Update(msg)
	return m, cmd
}

func (m model) handleEdit(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit
	case "esc":
		m.screen = screenList
		m.err = ""
		return m, nil
	case "enter":
		alias := strings.TrimSpace(m.inputs[0].Value())
		if alias == "" {
			m.err = "alias is required"
			return m, nil
		}
		return m, cmdEdit(m.accounts, m.cursor, alias)
	}

	var cmd tea.Cmd
	m.inputs[m.focused], cmd = m.inputs[m.focused].Update(msg)
	return m, cmd
}

func (m model) handleConfirm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q", "n", "esc":
		m.screen = screenList
		return m, nil
	case "y", "enter":
		return m, cmdDelete(m.accounts, m.cursor)
	}
	return m, nil
}

func (m model) View() string {
	switch m.screen {
	case screenList:
		return m.renderList()
	case screenAdd:
		return m.renderAdd()
	case screenKey:
		return m.renderKey()
	case screenConfirm:
		return m.renderConfirm()
	case screenEdit:
		return m.renderEdit()
	}
	return ""
}

func (m model) renderList() string {
	var b strings.Builder
	b.WriteString("\n  " + sTitle.Render("gssh") + "  " + sDim.Render("github account manager") + "\n\n")

	if len(m.accounts) == 0 {
		b.WriteString("  " + sDim.Render("no accounts — press a to add one") + "\n")
	}

	for i, a := range m.accounts {
		cur := "  "
		if i == m.cursor {
			cur = sAccent.Render("› ")
		}

		dot := sDim.Render("○ ")
		if a.Active {
			dot = sGreen.Render("● ")
		}

		var aliasStr string
		w15 := lipgloss.NewStyle().Width(15)
		w18 := lipgloss.NewStyle().Width(18)
		switch {
		case a.Active:
			aliasStr = sGreen.Width(15).Render(a.Alias)
		case i == m.cursor:
			aliasStr = sBold.Width(15).Render(a.Alias)
		default:
			aliasStr = w15.Render(a.Alias)
		}

		b.WriteString(cur + dot + aliasStr + w18.Render(a.Username) + sDim.Render(a.Email) + "\n")
	}

	if m.err != "" {
		b.WriteString("\n  " + sErr.Render(m.err) + "\n")
	}

	b.WriteString("\n  " + sHelp.Render("[↑↓] nav  [enter] activate  [a] add  [e] edit  [d] delete  [k] key  [q] quit") + "\n\n")
	return b.String()
}

func (m model) renderAdd() string {
	var b strings.Builder
	b.WriteString("\n  " + sTitle.Render("gssh") + "  " + sDim.Render("add account") + "\n\n")

	labels := []string{"alias    ", "username ", "email    "}
	for i, inp := range m.inputs {
		marker := "  "
		if i == m.focused {
			marker = sAccent.Render("› ")
		}
		b.WriteString(marker + sDim.Render(labels[i]+"  ") + inp.View() + "\n")
	}

	if m.err != "" {
		b.WriteString("\n  " + sErr.Render(m.err) + "\n")
	}

	b.WriteString("\n  " + sHelp.Render("[tab] next  [enter] confirm  [esc] back") + "\n\n")
	return b.String()
}

func (m model) renderEdit() string {
	var b strings.Builder
	b.WriteString("\n  " + sTitle.Render("gssh") + "  " + sDim.Render("edit alias") + "\n\n")

	marker := sAccent.Render("› ")
	b.WriteString(marker + sDim.Render("alias    "+"  ") + m.inputs[0].View() + "\n")

	if m.err != "" {
		b.WriteString("\n  " + sErr.Render(m.err) + "\n")
	}

	b.WriteString("\n  " + sHelp.Render("[enter] confirm  [esc] back") + "\n\n")
	return b.String()
}

func (m model) renderConfirm() string {
	var b strings.Builder
	alias := m.accounts[m.cursor].Alias
	b.WriteString("\n  " + sTitle.Render("gssh") + "  " + sDim.Render("delete account") + "\n\n")
	b.WriteString("  " + sErr.Render("delete "+alias+"?") + "\n\n")
	b.WriteString("\n  " + sHelp.Render("[y/enter] yes  [n/esc] no") + "\n\n")
	return b.String()
}

func (m model) renderKey() string {
	alias := ""
	if m.cursor < len(m.accounts) {
		alias = m.accounts[m.cursor].Alias
	}

	var b strings.Builder
	b.WriteString("\n  " + sTitle.Render("gssh") + "  " + sDim.Render("public key → "+alias) + "\n\n")
	b.WriteString("  " + sDim.Render("↓ add this at github.com/settings/keys") + "\n\n")
	b.WriteString("  " + sKey.Render(strings.TrimSpace(m.pubKey)) + "\n")
	b.WriteString("\n  " + sHelp.Render("[any key] back") + "\n\n")
	return b.String()
}

func cmdActivate(accounts []Account, idx int) tea.Cmd {
	return func() tea.Msg {
		accs, err := activate(accounts, idx)
		if err != nil {
			return msgErr{err}
		}
		return msgUpdated{accs}
	}
}

func cmdDelete(accounts []Account, idx int) tea.Cmd {
	return func() tea.Msg {
		accs, err := deleteAcc(accounts, idx)
		if err != nil {
			return msgErr{err}
		}
		return msgUpdated{accs}
	}
}

func cmdAdd(accounts []Account, alias, username, email string) tea.Cmd {
	return func() tea.Msg {
		accs, pubKey, err := createAccount(accounts, alias, username, email)
		if err != nil {
			return msgErr{err}
		}
		return msgKey{accs, pubKey}
	}
}

func cmdEdit(accounts []Account, idx int, newAlias string) tea.Cmd {
	return func() tea.Msg {
		accs, err := editAlias(accounts, idx, newAlias)
		if err != nil {
			return msgErr{err}
		}
		return msgUpdated{accs}
	}
}

func main() {
	p := tea.NewProgram(initial(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
