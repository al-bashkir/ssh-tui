package ui

import (
	"fmt"

	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/sahilm/fuzzy"
)

// groupPickerDelegate renders groupRows as plain names.
var groupPickerDelegate = rowDelegate(func(width int, active bool, item list.Item) string {
	row := item.(groupRow)
	return renderSimpleRow(width, active, row.name, row.matchedIndexes)
})

type groupPickerModel struct {
	opts        Options
	parentCrumb string

	// hosts being added — shown as a summary line in the picker.
	hosts             []string
	cachedAddingLines []string // set on resize, reused by View

	width  int
	height int

	all      []groupRow
	names    []string // cached for fuzzy search, parallel to all
	keymap   keyMap
	showHelp bool
	toast    toast

	list       list.Model
	search     textinput.Model
	focus      focusState
	prevSearch string
}

func newGroupPickerModel(opts Options) *groupPickerModel {
	l := list.New(nil, groupPickerDelegate, 0, 0)
	configureList(&l)

	all := groupsRows(opts.Inventory)
	m := &groupPickerModel{
		opts:   opts,
		all:    all,
		names:  groupNames(all),
		keymap: defaultKeyMap(),
		list:   l,
		search: newSearchInput(),
		focus:  focusList,
	}
	m.applyFilter("")
	return m
}

func (m *groupPickerModel) Init() tea.Cmd { return nil }

func (m *groupPickerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		innerW, innerH := frameInnerSize(msg.Width, msg.Height)
		m.cachedAddingLines = m.addingLines(innerW)
		m.list.SetSize(innerW, max(1, innerH-5-len(m.cachedAddingLines)))
		m.search.Width = max(10, innerW-len(m.search.Prompt))
		return m, nil
	case tea.KeyMsg:
		if m.showHelp {
			if key.Matches(msg, m.keymap.Help) || msg.String() == "esc" {
				m.showHelp = false
			}
			return m, nil
		}

		if key.Matches(msg, m.keymap.Help) {
			m.showHelp = true
			return m, nil
		}
		if handleFocusKeys(msg, m.keymap, &m.focus, &m.search) {
			return m, nil
		}
		if key.Matches(msg, m.keymap.Esc) {
			if handlePickerEsc(&m.focus, &m.search, &m.prevSearch, m.applyFilter) {
				return m, nil
			}
			return m, func() tea.Msg { return groupPickerCancelMsg{} }
		}
		if key.Matches(msg, m.keymap.Connect) {
			if m.focus == focusSearch {
				setFocus(&m.focus, &m.search, focusList)
				return m, nil
			}
			row, ok := m.list.SelectedItem().(groupRow)
			if !ok {
				return m, nil
			}
			return m, func() tea.Msg { return groupPickerDoneMsg{groupIndex: row.index} }
		}
	}

	return m, updateSearchOrList(m.focus, &m.search, &m.list, &m.prevSearch, msg, m.applyFilter)
}

// addingLines packs the pending host names into at most 2 lines, each fitting
// within innerW. The first line is prefixed with "adding: " or "adding N: ".
// If hosts overflow 2 lines, the last line ends with a dim "+N more" count.
// Returns nil when no hosts are set.
func (m *groupPickerModel) addingLines(innerW int) []string {
	if len(m.hosts) == 0 {
		return nil
	}

	prefix := "adding: "
	if len(m.hosts) > 1 {
		prefix = fmt.Sprintf("adding %d: ", len(m.hosts))
	}
	prefixW := lipgloss.Width(prefix)

	// Narrow-width fallback: just show count, no individual names.
	if innerW < prefixW+3 {
		return []string{dim.Render(fmt.Sprintf("adding %d", len(m.hosts)))}
	}

	// Two-line budgets: line 0 is narrowed by the prefix, line 1 is full width.
	budgets := [2]int{max(1, innerW-prefixW), max(1, innerW)}

	type lineBuf struct {
		hosts []string
		usedW int
	}
	bufs := [2]lineBuf{}
	li := 0

	for i, h := range m.hosts {
		sep := ""
		if len(bufs[li].hosts) > 0 {
			sep = ", "
		}
		sepW := lipgloss.Width(sep)

		// Truncate hostname if it would exceed the remaining budget.
		avail := budgets[li] - bufs[li].usedW - sepW
		if avail < 1 {
			avail = 1
		}
		h = truncateTail(h, avail)
		tokenW := sepW + lipgloss.Width(h)

		// On the last line, reserve room for a "+N more" trailer.
		isLastBudget := li == len(budgets)-1
		remaining := len(m.hosts) - i - 1
		trailerW := 0
		if isLastBudget && remaining > 0 {
			trailerW = lipgloss.Width(fmt.Sprintf(", +%d more", remaining))
			// If the long trailer doesn't fit, try a shorter form.
			if bufs[li].usedW+tokenW+trailerW > budgets[li] {
				trailerW = lipgloss.Width(fmt.Sprintf(", +%d", remaining))
			}
		}

		if bufs[li].usedW+tokenW+trailerW <= budgets[li] || len(bufs[li].hosts) == 0 {
			bufs[li].hosts = append(bufs[li].hosts, h)
			bufs[li].usedW += tokenW
		} else if li < len(budgets)-1 {
			// Wrap to the next line.
			li++
			bufs[li].hosts = append(bufs[li].hosts, h)
			bufs[li].usedW = lipgloss.Width(h)
		} else {
			// Last line is full — remaining hosts become the "+N more" count.
			break
		}
	}

	placed := len(bufs[0].hosts) + len(bufs[1].hosts)
	leftOver := len(m.hosts) - placed

	var out []string
	for i, buf := range bufs {
		if len(buf.hosts) == 0 {
			continue
		}
		var sb strings.Builder
		if i == 0 {
			sb.WriteString(dim.Render(prefix))
		}
		sb.WriteString(strings.Join(buf.hosts, ", "))
		if i == li && leftOver > 0 {
			sb.WriteString(dim.Render(fmt.Sprintf(", +%d more", leftOver)))
		}
		out = append(out, sb.String())
	}
	return out
}

func (m *groupPickerModel) View() string {
	if m.showHelp {
		return renderHelpModal(m.width, m.height, "Select Group", m.helpKeys(), nil)
	}
	innerW, _ := frameInnerSize(m.width, m.height)
	sep := dim.Render(strings.Repeat("─", innerW))
	searchLine := m.search.View()
	listView := strings.TrimRight(m.list.View(), "\n")

	bodyParts := make([]string, 0, 2)
	if len(m.cachedAddingLines) > 0 {
		bodyParts = append(bodyParts, strings.Join(m.cachedAddingLines, "\n"))
	}
	bodyParts = append(bodyParts, searchLine+"\n"+sep+"\n"+listView+"\n"+sep)
	body := strings.TrimRight(strings.Join(bodyParts, "\n"), "\n")

	return renderFocusedFrame(m.width, m.height, breadcrumbTitle(m.parentCrumb, "Select group"), "", body, m.statusLine())
}

func (m *groupPickerModel) helpKeys() []helpSection {
	esc := key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("esc", "back/clear"),
	)
	selectGroup := key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "select"),
	)

	return []helpSection{
		{title: "Navigation", keys: []key.Binding{
			m.list.KeyMap.CursorUp,
			m.list.KeyMap.CursorDown,
			m.list.KeyMap.PrevPage,
			m.list.KeyMap.NextPage,
			m.keymap.FocusSearch,
			m.keymap.ToggleFocus,
		}},
		{title: "Selection", keys: []key.Binding{
			selectGroup,
		}},
		{title: "General", keys: []key.Binding{
			esc,
			m.keymap.Help,
		}},
	}
}

func (m *groupPickerModel) statusLine() string {
	left := fmt.Sprintf("groups: %d/%d", len(m.list.Items()), len(m.all))
	return pickerStatusLine(left, &m.list, m.search.Value(), m.toast)
}

func (m *groupPickerModel) applyFilter(query string) {
	var items []list.Item
	if query = strings.TrimSpace(query); query == "" {
		for _, r := range m.all {
			items = append(items, r)
		}
	} else {
		for _, mt := range fuzzy.Find(query, m.names) {
			r := m.all[mt.Index]
			r.matchedIndexes = mt.MatchedIndexes
			items = append(items, r)
		}
	}
	m.list.SetItems(items)
	if len(items) > 0 {
		m.list.Select(0)
	}
}
