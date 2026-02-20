package ui

import (
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/sahilm/fuzzy"
)

type hostPickerCancelMsg struct{}

type hostPickerDoneMsg struct {
	hosts []string
}

type pickerRow struct {
	host     string
	selected bool
	hasCfg   bool
}

func (i pickerRow) Title() string       { return i.host }
func (i pickerRow) Description() string { return "" }
func (i pickerRow) FilterValue() string { return i.host }

type pickerDelegate struct{}

func (d pickerDelegate) Height() int                             { return 1 }
func (d pickerDelegate) Spacing() int                            { return 0 }
func (d pickerDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }

func (d pickerDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	row, ok := item.(pickerRow)
	if !ok {
		fmt.Fprint(w, item.FilterValue())
		return
	}
	fmt.Fprint(w, renderHostLikeRow(m.Width(), index == m.Index(), row.selected, row.host, row.hasCfg, false))
}

type hostPickerModel struct {
	opts        Options
	parentCrumb string

	width  int
	height int

	allHosts []string
	filtered []string
	selected map[string]bool

	list   list.Model
	search textinput.Model
	focus  focusState

	keymap      keyMap
	help        help.Model
	showHelp    bool
	toast       toast
	confirmQuit bool

	prevSearch string
}

func newHostPickerModel(opts Options, _groupIndex int) *hostPickerModel {
	all := append([]string(nil), opts.Hosts...)
	items := make([]list.Item, 0, len(all))
	for _, h := range all {
		_, ok := hostConfigFor(opts.Config, h)
		items = append(items, pickerRow{host: h, hasCfg: ok})
	}

	l := list.New(items, pickerDelegate{}, 0, 0)
	l.Title = "Add hosts"
	configureList(&l)

	search := textinput.New()
	search.Placeholder = "search"
	search.Prompt = "/ "
	search.CharLimit = 256
	search.Width = 40
	configureSearch(&search)
	setSearchBarFocused(&search, false)

	m := &hostPickerModel{
		opts:     opts,
		allHosts: all,
		filtered: append([]string(nil), all...),
		selected: make(map[string]bool),
		list:     l,
		search:   search,
		focus:    focusList,
		keymap:   defaultKeyMap(),
		help:     help.New(),
		showHelp: false,
	}
	return m
}

func (m *hostPickerModel) Init() tea.Cmd { return nil }

func (m *hostPickerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		w := msg.Width
		h := msg.Height
		m.width = w
		m.height = h
		innerW, innerH := frameInnerSize(w, h)
		m.list.SetSize(innerW, max(1, innerH-5))
		m.search.Width = max(10, innerW-len(m.search.Prompt))
		return m, nil
	case tea.KeyMsg:
		if m.showHelp {
			if key.Matches(msg, m.keymap.Help) || msg.String() == "esc" {
				m.showHelp = false
				return m, nil
			}
			return m, nil
		}

		if m.confirmQuit {
			s := msg.String()
			switch s {
			case "y", "Y", "enter":
				return m, tea.Quit
			case "n", "N", "esc":
				m.confirmQuit = false
				m.toast = toast{}
				return m, nil
			default:
				return m, nil
			}
		}
		if key.Matches(msg, m.keymap.Quit) {
			if !m.opts.Config.Defaults.ConfirmQuit {
				return m, tea.Quit
			}
			m.confirmQuit = true
			m.toast = toast{text: "quit? (y/n)", level: toastWarn}
			return m, nil
		}
		if key.Matches(msg, m.keymap.Help) {
			m.showHelp = !m.showHelp
			return m, nil
		}
		if key.Matches(msg, m.keymap.Esc) {
			if m.focus == focusSearch {
				if m.search.Value() != "" {
					m.search.SetValue("")
					m.applyFilter("")
					m.prevSearch = ""
					return m, nil
				}
				m.focus = focusList
				m.search.Blur()
				setSearchBarFocused(&m.search, false)
				return m, nil
			}
			if m.search.Value() != "" {
				m.search.SetValue("")
				m.applyFilter("")
				m.prevSearch = ""
				return m, nil
			}
			return m, func() tea.Msg { return hostPickerCancelMsg{} }
		}
		if key.Matches(msg, m.keymap.FocusSearch) {
			m.focus = focusSearch
			m.search.Focus()
			setSearchBarFocused(&m.search, true)
			return m, nil
		}
		if key.Matches(msg, m.keymap.ToggleFocus) {
			if m.focus == focusSearch {
				m.focus = focusList
				m.search.Blur()
				setSearchBarFocused(&m.search, false)
			} else {
				m.focus = focusSearch
				m.search.Focus()
				setSearchBarFocused(&m.search, true)
			}
			return m, nil
		}
		if key.Matches(msg, m.keymap.ToggleSel) && m.focus == focusList {
			m.toggleCurrentSelection()
			return m, nil
		}
		if key.Matches(msg, m.keymap.SelectAll) && m.focus == focusList {
			for _, h := range m.filtered {
				m.selected[h] = true
			}
			m.refreshVisibleSelection()
			return m, nil
		}
		if key.Matches(msg, m.keymap.ClearSel) && m.focus == focusList {
			m.selected = make(map[string]bool)
			m.refreshVisibleSelection()
			return m, nil
		}
		if key.Matches(msg, m.keymap.Connect) {
			if m.focus == focusSearch {
				m.focus = focusList
				m.search.Blur()
				setSearchBarFocused(&m.search, false)
				return m, nil
			}
			picked := m.selectedHosts()
			if len(picked) == 0 {
				row, ok := m.list.SelectedItem().(pickerRow)
				if ok && row.host != "" {
					picked = []string{row.host}
				}
			}
			return m, func() tea.Msg { return hostPickerDoneMsg{hosts: picked} }
		}
	}

	var cmd tea.Cmd
	if m.focus == focusSearch {
		m.search, cmd = m.search.Update(msg)
		cur := m.search.Value()
		if cur != m.prevSearch {
			m.applyFilter(cur)
			m.prevSearch = cur
		}
		return m, cmd
	}

	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m *hostPickerModel) View() string {
	if m.showHelp {
		return renderHelpModal(m.width, m.height, "Add Hosts", m.help, m.helpKeys())
	}
	if m.confirmQuit {
		return renderQuitConfirm(m.width, m.height)
	}
	innerW, _ := frameInnerSize(m.width, m.height)
	sep := dim.Render(strings.Repeat("─", innerW))
	searchLine := m.search.View()
	listView := strings.TrimRight(m.list.View(), "\n")
	body := strings.TrimRight(searchLine+"\n"+sep+"\n"+listView+"\n"+sep, "\n")
	return renderFrame(m.width, m.height, breadcrumbTitle(m.parentCrumb, "Add hosts"), "", body, m.statusLine())
}

func (m *hostPickerModel) helpKeys() helpMap {
	esc := key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("esc", "back/clear"),
	)
	add := key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "add"),
	)

	return helpMap{
		short: []key.Binding{
			m.list.KeyMap.CursorUp,
			m.list.KeyMap.CursorDown,
			m.keymap.FocusSearch,
			m.keymap.ToggleSel,
			add,
			esc,
			m.keymap.Help,
			m.keymap.Quit,
		},
		full: [][]key.Binding{{
			m.list.KeyMap.CursorUp,
			m.list.KeyMap.CursorDown,
			m.list.KeyMap.PrevPage,
			m.list.KeyMap.NextPage,
		}, {
			m.keymap.FocusSearch,
			m.keymap.ToggleFocus,
			esc,
		}, {
			m.keymap.ToggleSel,
			m.keymap.SelectAll,
			m.keymap.ClearSel,
			add,
		}, {
			esc,
			m.keymap.Help,
			m.keymap.Quit,
		}},
	}
}

func (m *hostPickerModel) statusLine() string {
	shown := len(m.list.Items())
	total := len(m.allHosts)
	sel := len(m.selected)
	pg := ""
	if m.list.Paginator.TotalPages > 1 {
		pg = fmt.Sprintf("pg:%d/%d", m.list.Paginator.Page+1, m.list.Paginator.TotalPages)
	}

	q := strings.TrimSpace(m.search.Value())
	searchInfo := "search"
	if q != "" {
		if len(q) > 40 {
			q = q[:40] + "..."
		}
		searchInfo = "search: " + q
	}

	left := fmt.Sprintf("hosts: %d/%d  sel:%d", shown, total, sel)
	if pg != "" {
		left += "  " + dim.Render(pg)
	}
	if !m.toast.empty() {
		left += "  " + renderToast(m.toast)
	}
	return left + "  " + statusOK.Render(searchInfo)
}

func (m *hostPickerModel) applyFilter(query string) {
	query = strings.TrimSpace(query)
	if query == "" {
		m.filtered = append([]string(nil), m.allHosts...)
		m.setListItems(m.filtered)
		return
	}

	matches := fuzzy.Find(query, m.allHosts)
	filtered := make([]string, 0, len(matches))
	for _, match := range matches {
		filtered = append(filtered, match.Str)
	}
	m.filtered = filtered
	m.setListItems(m.filtered)
}

func (m *hostPickerModel) setListItems(hosts []string) {
	items := make([]list.Item, 0, len(hosts))
	for _, h := range hosts {
		_, ok := hostConfigFor(m.opts.Config, h)
		items = append(items, pickerRow{host: h, selected: m.selected[h], hasCfg: ok})
	}
	m.list.SetItems(items)
	if len(items) > 0 {
		m.list.Select(0)
	}
}

func (m *hostPickerModel) refreshVisibleSelection() {
	items := m.list.Items()
	for i := range items {
		row, ok := items[i].(pickerRow)
		if !ok {
			continue
		}
		row.selected = m.selected[row.host]
		items[i] = row
	}
	m.list.SetItems(items)
}

func (m *hostPickerModel) refreshVisibleBadges() {
	idx := m.list.Index()
	items := m.list.Items()
	for i := range items {
		row, ok := items[i].(pickerRow)
		if !ok {
			continue
		}
		_, ok = hostConfigFor(m.opts.Config, row.host)
		row.hasCfg = ok
		items[i] = row
	}
	m.list.SetItems(items)
	if idx >= 0 && idx < len(items) {
		m.list.Select(idx)
	}
}

func (m *hostPickerModel) toggleCurrentSelection() {
	row, ok := m.list.SelectedItem().(pickerRow)
	if !ok || row.host == "" {
		return
	}
	if m.selected[row.host] {
		delete(m.selected, row.host)
	} else {
		m.selected[row.host] = true
	}
	m.refreshVisibleSelection()
}

func (m *hostPickerModel) selectedHosts() []string {
	if len(m.selected) == 0 {
		return nil
	}
	out := make([]string, 0, len(m.selected))
	for _, h := range m.allHosts {
		if m.selected[h] {
			out = append(out, h)
		}
	}
	return out
}
