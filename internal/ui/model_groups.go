package ui

import (
	"fmt"
	"io"
	"os/exec"
	"strings"

	"github.com/al-bashkir/ssh-tui/internal/config"
	"github.com/al-bashkir/ssh-tui/internal/sshcmd"
	tmx "github.com/al-bashkir/ssh-tui/internal/tmux"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/sahilm/fuzzy"
)

type groupsDelegate struct{}

func (d groupsDelegate) Height() int                             { return 1 }
func (d groupsDelegate) Spacing() int                            { return 0 }
func (d groupsDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }

func (d groupsDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	row, ok := item.(groupRow)
	if !ok {
		fmt.Fprint(w, item.FilterValue())
		return
	}
	fmt.Fprint(w, renderGroupRow(m.Width(), index == m.Index(), row.name, row.hostCount, row.hasCfg))
}

type groupRow struct {
	index     int
	name      string
	hostCount int
	hasCfg    bool
}

func (i groupRow) Title() string       { return i.name }
func (i groupRow) Description() string { return "" }
func (i groupRow) FilterValue() string { return i.name }

type groupsModel struct {
	opts Options

	width  int
	height int

	allRows []groupRow
	rows    []groupRow

	list   list.Model
	search textinput.Model
	focus  focusState

	keymap    keyMap
	help      help.Model
	showHelp  bool
	helpVP    viewport.Model
	cmdPrompt bool
	cmdInput  textinput.Model
	toast     toast

	confirmQuit         bool
	confirmDelete       bool
	deleteIndex         int
	confirmConnect      bool
	confirmConnectCount int
	confirmConnectHosts []string
	pendingConnectFn    func() tea.Cmd

	prevSearch string

	quitting bool
	execCmd  []string
}

func newGroupsModel(opts Options) *groupsModel {
	rows := groupsRows(opts.Inventory)
	items := make([]list.Item, 0, len(rows))
	for _, r := range rows {
		items = append(items, r)
	}

	l := list.New(items, list.NewDefaultDelegate(), 0, 0)
	l.SetDelegate(groupsDelegate{})
	l.Title = "Groups"
	configureList(&l)

	search := textinput.New()
	search.Placeholder = "search"
	search.Prompt = "/ "
	search.CharLimit = 256
	search.Width = 40
	configureSearch(&search)
	setSearchBarFocused(&search, false)

	m := &groupsModel{
		opts:     opts,
		allRows:  rows,
		rows:     rows,
		list:     l,
		search:   search,
		focus:    focusList,
		keymap:   defaultKeyMap(),
		help:     help.New(),
		showHelp: false,
	}
	return m
}

func groupsRows(inv config.Inventory) []groupRow {
	rows := make([]groupRow, 0, len(inv.Groups))
	for i, g := range inv.Groups {
		rows = append(rows, groupRow{index: i, name: g.Name, hostCount: len(g.Hosts), hasCfg: groupHasCfg(g)})
	}
	return rows
}

func groupHasCfg(g config.Group) bool {
	return strings.TrimSpace(g.User) != "" ||
		g.Port != 0 ||
		strings.TrimSpace(g.IdentityFile) != "" ||
		len(g.ExtraArgs) > 0 ||
		strings.TrimSpace(g.RemoteCommand) != "" ||
		strings.TrimSpace(g.Tmux) != "" ||
		strings.TrimSpace(g.OpenMode) != "" ||
		strings.TrimSpace(g.PaneSplit) != "" ||
		strings.TrimSpace(g.PaneLayout) != "" ||
		strings.TrimSpace(g.PaneSync) != "" ||
		strings.TrimSpace(g.PaneBorderFmt) != "" ||
		strings.TrimSpace(g.PaneBorderPos) != ""
}

func (m *groupsModel) setRows(rows []groupRow) {
	m.rows = rows
	items := make([]list.Item, 0, len(rows))
	for _, r := range rows {
		items = append(items, r)
	}
	m.list.SetItems(items)
	if len(items) > 0 {
		m.list.Select(0)
	}
}

func (m *groupsModel) Refresh(inv config.Inventory) {
	m.opts.Inventory = inv
	m.allRows = groupsRows(inv)
	m.applyFilter(m.search.Value())
}

func (m *groupsModel) Init() tea.Cmd { return nil }

func (m *groupsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		w := msg.Width
		h := msg.Height
		m.width = w
		m.height = h
		innerW := max(0, w-2)
		innerH := max(0, h-2)
		// tabs + sep + header + sep + footer sep + footer
		m.list.SetSize(innerW, max(1, innerH-6))
		promptW := len(m.search.Prompt)
		reserve := 18
		m.search.Width = max(10, innerW-reserve-promptW)
		return m, nil
	case tea.KeyMsg:
		if m.showHelp {
			if key.Matches(msg, m.keymap.Help) || msg.String() == "esc" {
				m.showHelp = false
				return m, nil
			}
			updateHelpViewport(&m.helpVP, msg)
			return m, nil
		}

		if m.cmdPrompt {
			s := msg.String()
			switch s {
			case "esc":
				m.cmdPrompt = false
				m.cmdInput.Blur()
				return m, nil
			case "enter":
				cmd := strings.TrimSpace(m.cmdInput.Value())
				m.cmdPrompt = false
				m.cmdInput.Blur()
				m.cmdInput.SetValue("")
				if cmd == "" {
					return m, nil
				}
				m.toast = toast{}
				return m, m.connectAllCmd(false, cmd)
			default:
				var cmdTea tea.Cmd
				m.cmdInput, cmdTea = m.cmdInput.Update(msg)
				return m, cmdTea
			}
		}

		if m.confirmQuit {
			s := msg.String()
			switch s {
			case "y", "Y", "enter":
				m.quitting = true
				return m, tea.Quit
			case "n", "N", "esc":
				m.confirmQuit = false
				m.toast = toast{}
				return m, nil
			default:
				return m, nil
			}
		}
		if m.confirmDelete {
			s := msg.String()
			switch s {
			case "y", "Y", "enter":
				idx := m.deleteIndex
				m.confirmDelete = false
				return m, func() tea.Msg { return deleteGroupMsg{index: idx} }
			case "n", "N", "esc":
				m.confirmDelete = false
				m.toast = toast{}
				return m, nil
			default:
				return m, nil
			}
		}
		if m.confirmConnect {
			s := msg.String()
			switch s {
			case "y", "Y", "enter":
				m.confirmConnect = false
				fn := m.pendingConnectFn
				m.pendingConnectFn = nil
				if fn != nil {
					return m, fn()
				}
				return m, nil
			case "n", "N", "esc":
				m.confirmConnect = false
				m.pendingConnectFn = nil
				m.toast = toast{}
				return m, nil
			default:
				return m, nil
			}
		}

		if key.Matches(msg, m.keymap.Quit) {
			if !m.opts.Config.Defaults.ConfirmQuit {
				m.quitting = true
				return m, tea.Quit
			}
			m.confirmQuit = true
			m.toast = toast{text: "quit? (y/n)", level: toastWarn}
			return m, nil
		}
		if key.Matches(msg, m.keymap.Help) {
			m.showHelp = !m.showHelp
			if m.showHelp && m.width > 0 && m.height > 0 {
				m.helpVP = initHelpViewport(m.width, m.height, "Groups", m.help, m.helpKeys())
			}
			return m, nil
		}
		if key.Matches(msg, m.keymap.Settings) && m.focus == focusList {
			return m, func() tea.Msg { return openDefaultsFormMsg{returnTo: screenGroups} }
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
		if key.Matches(msg, m.keymap.SwitchTab) && m.focus != focusSearch {
			return m, func() tea.Msg { return switchScreenMsg{to: screenHosts} }
		}
		if key.Matches(msg, m.keymap.Esc) {
			if m.focus == focusSearch && m.search.Value() == "" {
				m.focus = focusList
				m.search.Blur()
				setSearchBarFocused(&m.search, false)
				return m, nil
			}
			if m.search.Value() != "" {
				m.search.SetValue("")
				m.applyFilter("")
				m.prevSearch = ""
				if m.focus == focusSearch {
					m.focus = focusList
					m.search.Blur()
					setSearchBarFocused(&m.search, false)
				}
				return m, nil
			}
			return m, nil
		}
		if key.Matches(msg, m.keymap.NewGroup) && m.focus == focusList {
			return m, func() tea.Msg { return openGroupFormMsg{index: -1} }
		}
		if key.Matches(msg, m.keymap.EditGroup) && m.focus == focusList {
			row, ok := m.list.SelectedItem().(groupRow)
			if !ok {
				return m, nil
			}
			return m, func() tea.Msg { return openGroupFormMsg{index: row.index} }
		}
		if key.Matches(msg, m.keymap.AddHosts) && m.focus == focusList {
			row, ok := m.list.SelectedItem().(groupRow)
			if !ok {
				return m, nil
			}
			return m, func() tea.Msg { return openHostPickerMsg{groupIndex: row.index, returnTo: screenGroups} }
		}
		if key.Matches(msg, m.keymap.CustomHost) && m.focus == focusList {
			row, ok := m.list.SelectedItem().(groupRow)
			if !ok {
				return m, nil
			}
			return m, func() tea.Msg { return openCustomHostMsg{returnTo: screenGroups, groupIndex: row.index} }
		}
		if key.Matches(msg, m.keymap.DeleteGroup) && m.focus == focusList {
			row, ok := m.list.SelectedItem().(groupRow)
			if !ok {
				return m, nil
			}
			m.confirmDelete = true
			m.deleteIndex = row.index
			m.toast = toast{text: "delete? (y/n)", level: toastWarn}
			return m, nil
		}
		if key.Matches(msg, m.keymap.Connect) {
			if m.focus == focusSearch {
				if len(m.list.Items()) == 0 && m.search.Value() != "" {
					m.search.SetValue("")
					m.applyFilter("")
					m.prevSearch = ""
				}
				m.focus = focusList
				m.search.Blur()
				setSearchBarFocused(&m.search, false)
				return m, nil
			}
			row, ok := m.list.SelectedItem().(groupRow)
			if !ok {
				return m, nil
			}
			return m, func() tea.Msg { return openGroupHostsMsg{index: row.index} }
		}
		if key.Matches(msg, m.keymap.ConnectCmd) && m.focus == focusList {
			in := textinput.New()
			in.CharLimit = 512
			in.Prompt = "cmd: "
			in.Placeholder = "run on remote, keep session open"
			mw, mh := modalSize(m.width, m.height, 88, 9, 6, 10)
			innerW, _ := frameInnerSize(mw, mh)
			avail := innerW - len(in.Prompt)
			if avail < 1 {
				avail = 1
			}
			in.Width = min(70, avail)
			in.Focus()
			configureSearch(&in)
			setSearchFocused(&in, true)
			m.cmdInput = in
			m.cmdPrompt = true
			return m, nil
		}
		if key.Matches(msg, m.keymap.ConnectAll) && m.focus == focusList {
			m.toast = toast{}
			return m, m.connectAllCmd(false, "")
		}
		if key.Matches(msg, m.keymap.Copy) && m.focus == focusList {
			row, ok := m.list.SelectedItem().(groupRow)
			if !ok || row.index < 0 || row.index >= len(m.opts.Inventory.Groups) {
				m.toast = toast{text: "no group selected", level: toastWarn}
				return m, nil
			}
			g := m.opts.Inventory.Groups[row.index]
			g.Name = suggestCopyGroupName(m.opts.Inventory, g.Name)
			return m, func() tea.Msg { return openGroupFormPrefillMsg{group: g} }
		}
		if key.Matches(msg, m.keymap.OneWindow) && m.focus == focusList {
			m.toast = toast{}
			return m, m.connectAllCmd(true, "")
		}
	case toastMsg:
		m.toast = toast(msg)
		return m, nil
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

func (m *groupsModel) View() string {
	if m.showHelp {
		return renderHelpModalWithVP(m.width, m.height, "Groups", m.help, m.helpKeys(), &m.helpVP)
	}
	if m.cmdPrompt {
		mw, mh := modalSize(m.width, m.height, 88, 9, 6, 10)
		var b strings.Builder
		b.WriteString("Connect and run a remote command for all hosts (keeps sessions open).\n\n")
		b.WriteString(m.cmdInput.View())
		b.WriteString("\n")
		b.WriteString(footerStyle.Render("Enter connect  Esc cancel"))
		box := renderFrame(mw, mh, breadcrumbTitle("Groups", "Command"), "", strings.TrimRight(b.String(), "\n"), "")
		return placeCentered(m.width, m.height, box)
	}
	if m.confirmQuit {
		return renderQuitConfirm(m.width, m.height)
	}
	if m.confirmConnect {
		modal := connectConfirmBox(max(0, m.width-4), m.confirmConnectCount, m.confirmConnectHosts)
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, modal)
	}
	if m.confirmDelete {
		name := ""
		hostCount := 0
		if m.deleteIndex >= 0 && m.deleteIndex < len(m.opts.Inventory.Groups) {
			name = m.opts.Inventory.Groups[m.deleteIndex].Name
			hostCount = len(m.opts.Inventory.Groups[m.deleteIndex].Hosts)
		}
		innerW := max(0, m.width-2)
		innerH := max(0, m.height-2)
		contentH := max(0, innerH-4)
		modal := deleteGroupConfirmBox(innerW, name, hostCount)
		placed := lipgloss.Place(innerW, contentH, lipgloss.Center, lipgloss.Center, modal)
		right := statusDot(true, false) + "   " + dim.Render(fmt.Sprintf("%d groups", len(m.allRows)))
		return renderMainTabBox(m.width, m.height, 1, m.search.View(), right, placed)
	}

	right := ""
	if !m.toast.empty() {
		right = renderToast(m.toast)
	} else {
		right = statusDot(true, false) + "   " + dim.Render(fmt.Sprintf("%d groups", len(m.allRows)))
	}
	var footer string
	if m.width < 60 {
		footer = styledFooter("\u21b5 open  C connect  ? help")
	} else {
		footer = styledFooter("\u21b5 open  C connect  ·  o panes  Ctrl+o cmd  ·  n new")
		if m.height >= 20 {
			footer += "\n" + styledFooter("e edit  d delete  y copy  a add hosts  c custom  ·  g hosts  tab search  ? help")
		}
	}

	listContent := m.list.View()
	if len(m.list.Items()) == 0 {
		listContent = m.emptyStateView()
	}
	return renderMainTabBoxWithFooter(m.width, m.height, 1, m.search.View(), right, listContent, footer)
}

func (m *groupsModel) helpKeys() helpMap {
	openGroup := key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "open group"),
	)

	return helpMap{
		short: []key.Binding{
			m.list.KeyMap.CursorUp,
			m.list.KeyMap.CursorDown,
			m.keymap.ToggleFocus,
			openGroup,
			m.keymap.ConnectAll,
			m.keymap.ConnectCmd,
			m.keymap.OneWindow,
			m.keymap.CustomHost,
			m.keymap.NewGroup,
			m.keymap.EditGroup,
			m.keymap.Copy,
			m.keymap.DeleteGroup,
			m.keymap.AddHosts,
			m.keymap.SwitchTab,
			m.keymap.Settings,
			m.keymap.Help,
			m.keymap.Quit,
		},
		full: [][]key.Binding{{
			m.list.KeyMap.CursorUp,
			m.list.KeyMap.CursorDown,
			m.list.KeyMap.PrevPage,
			m.list.KeyMap.NextPage,
		}, {
			m.keymap.ToggleFocus,
			m.keymap.FocusSearch,
			m.keymap.SwitchTab,
			m.keymap.Esc,
			m.keymap.NewGroup,
			m.keymap.EditGroup,
			m.keymap.Copy,
			m.keymap.DeleteGroup,
		}, {
			openGroup,
			m.keymap.ConnectAll,
			m.keymap.ConnectCmd,
			m.keymap.OneWindow,
			m.keymap.CustomHost,
			m.keymap.AddHosts,
			m.keymap.Settings,
			m.keymap.Help,
			m.keymap.Quit,
		}},
	}
}

func (m *groupsModel) emptyStateView() string {
	innerW := max(0, m.width-2)
	innerH := max(0, m.height-2)
	contentH := max(0, innerH-6)

	q := strings.TrimSpace(m.search.Value())
	dots := dim.Render("·  ·  ·")
	var msg string
	if q != "" {
		msg = dots + "\n\n" + dim.Render(fmt.Sprintf("No matches for %q", q)) + "\n" + dim.Render("Esc to clear search")
	} else {
		msg = dots + "\n\n" + dim.Render("No groups yet.") + "\n" + dim.Render("n \u2014 create a new group")
	}

	return lipgloss.Place(innerW, contentH, lipgloss.Center, lipgloss.Center, msg)
}

func (m *groupsModel) statusLine() string {
	shown := len(m.list.Items())
	total := len(m.allRows)
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

	pos := ""
	if shown > 0 {
		pos = fmt.Sprintf("  %d of %d", m.list.Index()+1, shown)
	}
	left := fmt.Sprintf("groups: %d/%d", shown, total) + dim.Render(pos)
	if pg != "" {
		left += "  " + dim.Render(pg)
	}
	if !m.toast.empty() {
		left += "  " + renderToast(m.toast)
	}
	return left + "  " + statusOK.Render(searchInfo)
}

func (m *groupsModel) applyFilter(query string) {
	query = strings.TrimSpace(query)
	if query == "" {
		m.setRows(append([]groupRow(nil), m.allRows...))
		return
	}

	names := make([]string, 0, len(m.allRows))
	for _, r := range m.allRows {
		names = append(names, r.name)
	}
	matches := fuzzy.Find(query, names)
	rows := make([]groupRow, 0, len(matches))
	for _, mt := range matches {
		rows = append(rows, m.allRows[mt.Index])
	}
	m.setRows(rows)
}

func (m *groupsModel) connectAllCmd(oneWindow bool, remoteCmd string) tea.Cmd {
	row, ok := m.list.SelectedItem().(groupRow)
	if !ok {
		m.toast = toast{text: "no group selected", level: toastWarn}
		return nil
	}
	if row.index < 0 || row.index >= len(m.opts.Inventory.Groups) {
		m.toast = toast{text: "invalid group", level: toastErr}
		return nil
	}
	g := m.opts.Inventory.Groups[row.index]
	if len(g.Hosts) == 0 {
		m.toast = toast{text: "group has no hosts", level: toastWarn}
		return nil
	}
	if oneWindow && !tmx.InTmux() {
		m.toast = toast{text: "requires an active tmux session", level: toastWarn}
		return nil
	}

	if len(g.Hosts) > connectThreshold(m.opts.Config.Defaults) {
		m.confirmConnect = true
		m.confirmConnectCount = len(g.Hosts)
		m.confirmConnectHosts = append([]string(nil), g.Hosts...)
		m.pendingConnectFn = func() tea.Cmd {
			return m.doConnectAll(g, oneWindow, remoteCmd)
		}
		return nil
	}
	return m.doConnectAll(g, oneWindow, remoteCmd)
}

func (m *groupsModel) doConnectAll(g config.Group, oneWindow bool, remoteCmd string) tea.Cmd {
	defaults := m.opts.Config.Defaults
	base := sshcmd.FromDefaults(defaults)
	rc := strings.TrimSpace(remoteCmd)

	tmuxSetting := defaults.Tmux
	if strings.TrimSpace(g.Tmux) != "" {
		tmuxSetting = g.Tmux
	}
	openModeSetting := defaults.OpenMode
	if strings.TrimSpace(g.OpenMode) != "" {
		openModeSetting = g.OpenMode
	}

	inTmux := tmx.InTmux()
	mode := tmx.ResolveOpenMode(tmuxSetting, openModeSetting, inTmux)

	sshCmds := make([][]string, 0, len(g.Hosts))
	for _, h := range g.Hosts {
		s := base
		if hc, ok := hostConfigFor(m.opts.Inventory, h); ok {
			s = sshcmd.ApplyHost(s, hc)
		}
		s = sshcmd.ApplyGroup(s, g)
		if rc != "" {
			s.ExtraArgs = ensureSSHForceTTY(s.ExtraArgs)
			s.RemoteCommand = keepSessionOpenRemoteCmd(rc)
		}
		cmd, _ := sshcmd.BuildCommand(h, s)
		sshCmds = append(sshCmds, cmd)
	}

	if oneWindow {
		return func() tea.Msg {
			name := strings.TrimSpace(g.Name)
			if name == "" {
				name = windowName(g.Hosts[0])
			}
			psOne := resolvePaneSettings(defaults, &g, len(sshCmds))
			err := tmuxOpenOneWindow(sshCmds, tmuxOneWindowOpts{
				WindowName:       name,
				PaneTitles:       g.Hosts,
				SplitFlag:        psOne.SplitFlag,
				Layout:           psOne.Layout,
				SyncPanes:        psOne.SyncPanes,
				PaneBorderFormat: psOne.BorderFormat,
				PaneBorderStatus: psOne.BorderStatus,
			})
			if err != nil {
				return toastMsg{text: err.Error(), level: toastErr}
			}
			return toastMsg{text: fmt.Sprintf("opened %d in one window", len(sshCmds)), level: toastInfo}
		}
	}

	if mode == tmx.OpenCurrent {
		if len(sshCmds) > 1 {
			m.toast = toast{text: "multi-host requires tmux (window or pane mode)", level: toastWarn}
			return nil
		}
		m.execCmd = sshCmds[0]
		return tea.Quit
	}
	if !inTmux {
		if len(sshCmds) > 1 {
			m.toast = toast{text: "multi-host requires an active tmux session", level: toastWarn}
			return nil
		}
		m.execCmd = tmx.NewSessionCmd(defaults.TmuxSession, sshCmds[0])
		return tea.Quit
	}

	return func() tea.Msg {
		if mode == tmx.OpenPane {
			name := strings.TrimSpace(g.Name)
			if name == "" {
				name = windowName(g.Hosts[0])
			}
			ps := resolvePaneSettings(defaults, &g, len(sshCmds))
			err := tmuxOpenOneWindow(sshCmds, tmuxOneWindowOpts{
				WindowName:       name,
				PaneTitles:       g.Hosts,
				SplitFlag:        ps.SplitFlag,
				Layout:           ps.Layout,
				SyncPanes:        ps.SyncPanes,
				PaneBorderFormat: ps.BorderFormat,
				PaneBorderStatus: ps.BorderStatus,
			})
			if err != nil {
				return toastMsg{text: err.Error(), level: toastErr}
			}
			return toastMsg{text: fmt.Sprintf("opened %d in one window", len(sshCmds)), level: toastInfo}
		}
		if mode == tmx.OpenWindow && len(sshCmds) > 1 {
			name := strings.TrimSpace(g.Name)
			if name == "" {
				name = windowName(g.Hosts[0])
			}
			ps := resolvePaneSettings(defaults, &g, len(sshCmds))
			err := tmuxOpenOneWindow(sshCmds, tmuxOneWindowOpts{
				WindowName:       name,
				PaneTitles:       g.Hosts,
				SplitFlag:        ps.SplitFlag,
				Layout:           ps.Layout,
				SyncPanes:        ps.SyncPanes,
				PaneBorderFormat: ps.BorderFormat,
				PaneBorderStatus: ps.BorderStatus,
			})
			if err != nil {
				return toastMsg{text: err.Error(), level: toastErr}
			}
			return toastMsg{text: fmt.Sprintf("opened %d in one window", len(sshCmds)), level: toastInfo}
		}

		for i, sshCmd := range sshCmds {
			name := strings.TrimSpace(g.Name)
			if name == "" {
				name = windowName(g.Hosts[i])
			}
			tmuxCmd := tmx.NewWindowCmd(name, sshCmd)
			// #nosec G204 -- tmux argv is constructed (no shell) from known host/group settings.
			if err := exec.Command(tmuxCmd[0], tmuxCmd[1:]...).Run(); err != nil {
				return toastMsg{text: "tmux error: " + err.Error(), level: toastErr}
			}
		}
		return toastMsg{text: fmt.Sprintf("opened %d", len(sshCmds)), level: toastInfo}
	}
}

func (m *groupsModel) IsQuitting() bool  { return m.quitting }
func (m *groupsModel) ExecCmd() []string { return m.execCmd }
