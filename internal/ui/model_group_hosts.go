package ui

import (
	"fmt"
	"io"
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

type groupHostRow struct {
	host     string
	selected bool
	hasCfg   bool
}

func (i groupHostRow) Title() string       { return i.host }
func (i groupHostRow) Description() string { return "" }
func (i groupHostRow) FilterValue() string { return i.host }

type groupHostsDelegate struct{}

func (d groupHostsDelegate) Height() int                             { return 1 }
func (d groupHostsDelegate) Spacing() int                            { return 0 }
func (d groupHostsDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }

func (d groupHostsDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	row, ok := item.(groupHostRow)
	if !ok {
		fmt.Fprint(w, item.FilterValue())
		return
	}
	fmt.Fprint(w, renderHostLikeRow(m.Width(), index == m.Index(), row.selected, row.host, row.hasCfg, false))
}

type groupHostsModel struct {
	opts Options

	width  int
	height int

	groupIndex int
	group      config.Group

	allHosts []string
	filtered []string
	selected map[string]bool

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
	confirmRemove       bool
	removeList          []string
	confirmConnect      bool
	confirmConnectCount int
	confirmConnectHosts []string
	pendingConnectFn    func() tea.Cmd
	quitting            bool
	execCmd             []string

	prevSearch string
}

func newGroupHostsModel(opts Options, groupIndex int) *groupHostsModel {
	g := config.Group{}
	if groupIndex >= 0 && groupIndex < len(opts.Config.Groups) {
		g = opts.Config.Groups[groupIndex]
	}

	items := make([]list.Item, 0, len(g.Hosts))
	for _, h := range g.Hosts {
		_, ok := hostConfigFor(opts.Config, h)
		items = append(items, groupHostRow{host: h, hasCfg: ok})
	}

	l := list.New(items, groupHostsDelegate{}, 0, 0)
	l.Title = "Group: " + g.Name
	configureList(&l)

	search := textinput.New()
	search.Placeholder = "search"
	search.Prompt = "/ "
	search.CharLimit = 256
	search.Width = 40
	configureSearch(&search)
	setSearchBarFocused(&search, false)

	m := &groupHostsModel{
		opts:       opts,
		groupIndex: groupIndex,
		group:      g,
		allHosts:   append([]string(nil), g.Hosts...),
		filtered:   append([]string(nil), g.Hosts...),
		selected:   make(map[string]bool),
		list:       l,
		search:     search,
		focus:      focusList,
		keymap:     defaultKeyMap(),
		help:       help.New(),
		showHelp:   false,
	}
	return m
}

func (m *groupHostsModel) Init() tea.Cmd { return nil }

func (m *groupHostsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
		reserve := 24
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
				return m, m.handleConnectWithRemoteCommand(cmd)
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
		if m.confirmRemove {
			s := msg.String()
			switch s {
			case "y", "Y", "enter":
				toRemove := append([]string(nil), m.removeList...)
				m.confirmRemove = false
				m.removeList = nil
				return m, func() tea.Msg { return removeHostsMsg{groupIndex: m.groupIndex, hosts: toRemove} }
			case "n", "N", "esc":
				m.confirmRemove = false
				m.removeList = nil
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
				m.helpVP = initHelpViewport(m.width, m.height, "Group Hosts", m.help, m.helpKeys())
			}
			return m, nil
		}
		if key.Matches(msg, m.keymap.DeleteGroup) && m.focus == focusList {
			toRemove := m.selectedHosts()
			if len(toRemove) == 0 {
				row, ok := m.list.SelectedItem().(groupHostRow)
				if ok && row.host != "" {
					toRemove = []string{row.host}
				}
			}
			if len(toRemove) == 0 {
				m.toast = toast{text: "no host selected", level: toastWarn}
				return m, nil
			}
			m.confirmRemove = true
			m.removeList = toRemove
			m.toast = toast{text: fmt.Sprintf("remove %d? (y/n)", len(toRemove)), level: toastWarn}
			return m, nil
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
			return m, func() tea.Msg { return switchScreenMsg{to: screenGroups} }
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
			m.toast = toast{}
			return m, m.handleConnect()
		}
		if key.Matches(msg, m.keymap.OneWindow) && m.focus == focusList {
			m.toast = toast{}
			return m, m.openOneWindow()
		}
		if key.Matches(msg, m.keymap.ConnectSame) && m.focus == focusList {
			m.toast = toast{}
			return m, m.handleConnectSame()
		}
		if key.Matches(msg, m.keymap.AddHosts) && m.focus == focusList {
			return m, func() tea.Msg { return openHostPickerMsg{groupIndex: m.groupIndex, returnTo: screenGroupHosts} }
		}
		if key.Matches(msg, m.keymap.CustomHost) && m.focus == focusList {
			return m, func() tea.Msg { return openCustomHostMsg{returnTo: screenGroupHosts, groupIndex: m.groupIndex} }
		}
		if key.Matches(msg, m.keymap.HostConfig) && m.focus == focusList {
			row, ok := m.list.SelectedItem().(groupHostRow)
			if !ok || strings.TrimSpace(row.host) == "" {
				m.toast = toast{text: "no host selected", level: toastWarn}
				return m, nil
			}
			return m, func() tea.Msg { return openHostFormMsg{host: row.host, returnTo: screenGroupHosts} }
		}
		if key.Matches(msg, m.keymap.Copy) && m.focus == focusList {
			row, ok := m.list.SelectedItem().(groupHostRow)
			if !ok || strings.TrimSpace(row.host) == "" {
				m.toast = toast{text: "no host selected", level: toastWarn}
				return m, nil
			}
			hc, ok := hostConfigFor(m.opts.Config, row.host)
			if !ok {
				m.toast = toast{text: "no host config", level: toastWarn}
				return m, nil
			}
			hc.Host = suggestCopyHostKey(m.opts.Config, hc.Host)
			return m, func() tea.Msg { return openHostFormPrefillMsg{host: hc, returnTo: screenGroupHosts} }
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

func (m *groupHostsModel) View() string {
	if m.showHelp {
		return renderHelpModalWithVP(m.width, m.height, "Group Hosts", m.help, m.helpKeys(), &m.helpVP)
	}
	if m.cmdPrompt {
		mw, mh := modalSize(m.width, m.height, 88, 9, 6, 10)
		var b strings.Builder
		b.WriteString("Connect and run a remote command (keeps session open).\n\n")
		b.WriteString(m.cmdInput.View())
		b.WriteString("\n")
		b.WriteString(footerStyle.Render("Enter connect  Esc cancel"))
		breadcrumb := "Groups > " + m.group.Name
		box := renderFrame(mw, mh, breadcrumbTitle(breadcrumb, "Command"), "", strings.TrimRight(b.String(), "\n"), "")
		return placeCentered(m.width, m.height, box)
	}
	if m.confirmQuit {
		return renderQuitConfirm(m.width, m.height)
	}
	if m.confirmConnect {
		modal := connectConfirmBox(max(0, m.width-4), m.confirmConnectCount, m.confirmConnectHosts)
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, modal)
	}
	if m.confirmRemove {
		innerW := max(0, m.width-2)
		innerH := max(0, m.height-2)
		contentH := max(0, innerH-6)
		modal := removeHostsConfirmBox(innerW, m.removeList, m.group.Name)
		placed := lipgloss.Place(innerW, contentH, lipgloss.Center, lipgloss.Center, modal)
		breadcrumb := dim.Render("Groups >") + " " + headerStyle.Render(m.group.Name)
		right := statusDot(true, false)
		return renderBreadcrumbTabBox(m.width, m.height, breadcrumb, m.search.View(), right, placed, "")
	}

	right := ""
	if !m.toast.empty() {
		right = renderToast(m.toast)
	} else {
		right = statusDot(true, false)
		shown := len(m.list.Items())
		total := len(m.allHosts)
		q := strings.TrimSpace(m.search.Value())
		if q != "" {
			right += dim.Render(fmt.Sprintf(" %d / %d hosts", shown, total))
		} else {
			right += dim.Render(fmt.Sprintf(" %d hosts", total))
		}
		if selCount := len(m.selected); selCount > 0 {
			right += "   " + badgeSelStyle.Render(fmt.Sprintf("%d selected", selCount))
		}
	}
	var footer string
	if m.width < 60 {
		footer = styledFooter("\u21b5 connect  \u2423 select  esc back  ? help")
	} else {
		footer = styledFooter("\u21b5 connect  O pane  ·  \u2423 select  o panes  ·  Ctrl+o cmd  a add")
		if m.height >= 20 {
			footer += "\n" + styledFooter("e config  c custom  d remove  y copy  ·  tab search  esc back  ? help")
		}
	}

	listContent := m.list.View()
	if len(m.list.Items()) == 0 {
		listContent = m.emptyStateView()
	}
	breadcrumb := dim.Render("Groups >") + " " + headerStyle.Render(m.group.Name)
	return renderBreadcrumbTabBox(m.width, m.height, breadcrumb, m.search.View(), right, listContent, footer)
}

func (m *groupHostsModel) helpKeys() helpMap {
	esc := key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("esc", "back/clear"),
	)
	remove := key.NewBinding(
		key.WithKeys("d"),
		key.WithHelp("d", "remove"),
	)

	return helpMap{
		short: []key.Binding{
			m.list.KeyMap.CursorUp,
			m.list.KeyMap.CursorDown,
			m.keymap.ToggleFocus,
			m.keymap.ToggleSel,
			m.keymap.Connect,
			m.keymap.ConnectSame,
			m.keymap.ConnectCmd,
			m.keymap.OneWindow,
			m.keymap.AddHosts,
			m.keymap.CustomHost,
			m.keymap.HostConfig,
			m.keymap.Copy,
			remove,
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
			m.keymap.ToggleFocus,
			m.keymap.FocusSearch,
			esc,
		}, {
			m.keymap.ToggleSel,
			m.keymap.SelectAll,
			m.keymap.ClearSel,
			m.keymap.Connect,
			m.keymap.ConnectSame,
			m.keymap.ConnectCmd,
			m.keymap.OneWindow,
		}, {
			m.keymap.AddHosts,
			m.keymap.CustomHost,
			m.keymap.HostConfig,
			m.keymap.Copy,
			remove,
		}, {
			m.keymap.Help,
			m.keymap.Quit,
		}},
	}
}

func (m *groupHostsModel) emptyStateView() string {
	innerW := max(0, m.width-2)
	innerH := max(0, m.height-2)
	contentH := max(0, innerH-6)

	q := strings.TrimSpace(m.search.Value())
	dots := dim.Render("·  ·  ·")
	var msg string
	if q != "" {
		msg = dots + "\n\n" + dim.Render(fmt.Sprintf("No matches for %q", q)) + "\n" + dim.Render("Esc to clear search")
	} else {
		msg = dots + "\n\n" + dim.Render("No hosts in this group.") + "\n" + dim.Render("a \u2014 add hosts")
	}

	return lipgloss.Place(innerW, contentH, lipgloss.Center, lipgloss.Center, msg)
}

func (m *groupHostsModel) statusLine() string {
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

	pos := ""
	if shown > 0 {
		pos = fmt.Sprintf("  %d of %d", m.list.Index()+1, shown)
	}
	left := fmt.Sprintf("hosts: %d/%d  sel:%d", shown, total, sel) + dim.Render(pos)
	if pg != "" {
		left += "  " + dim.Render(pg)
	}
	if !m.toast.empty() {
		left += "  " + renderToast(m.toast)
	}
	return left + "  " + statusOK.Render(searchInfo)
}

func (m *groupHostsModel) applyFilter(query string) {
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

func (m *groupHostsModel) setListItems(hosts []string) {
	items := make([]list.Item, 0, len(hosts))
	for _, h := range hosts {
		_, ok := hostConfigFor(m.opts.Config, h)
		items = append(items, groupHostRow{host: h, selected: m.selected[h], hasCfg: ok})
	}
	m.list.SetItems(items)
	if len(items) > 0 {
		m.list.Select(0)
	}
}

func (m *groupHostsModel) refreshVisibleSelection() {
	items := m.list.Items()
	for i := range items {
		row, ok := items[i].(groupHostRow)
		if !ok {
			continue
		}
		row.selected = m.selected[row.host]
		items[i] = row
	}
	m.list.SetItems(items)
}

func (m *groupHostsModel) refreshVisibleBadges() {
	idx := m.list.Index()
	items := m.list.Items()
	for i := range items {
		row, ok := items[i].(groupHostRow)
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

func (m *groupHostsModel) toggleCurrentSelection() {
	row, ok := m.list.SelectedItem().(groupHostRow)
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

func (m *groupHostsModel) selectedHosts() []string {
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

func (m *groupHostsModel) ghHostsToOpen() []string {
	if sel := m.selectedHosts(); len(sel) > 0 {
		return sel
	}
	row, ok := m.list.SelectedItem().(groupHostRow)
	if ok && row.host != "" {
		return []string{row.host}
	}
	return nil
}

func (m *groupHostsModel) resolveGroupMode() (tmx.OpenMode, bool) {
	defaults := m.opts.Config.Defaults
	tmuxSetting := defaults.Tmux
	if strings.TrimSpace(m.group.Tmux) != "" {
		tmuxSetting = m.group.Tmux
	}
	openModeSetting := defaults.OpenMode
	if strings.TrimSpace(m.group.OpenMode) != "" {
		openModeSetting = m.group.OpenMode
	}
	inTmux := tmx.InTmux()
	return tmx.ResolveOpenMode(tmuxSetting, openModeSetting, inTmux), inTmux
}

func (m *groupHostsModel) buildGroupSSHCmds(hosts []string, modifySettings func(*sshcmd.Settings)) [][]string {
	base := sshcmd.FromDefaults(m.opts.Config.Defaults)
	cmds := make([][]string, 0, len(hosts))
	for _, h := range hosts {
		s := base
		if hc, ok := hostConfigFor(m.opts.Config, h); ok {
			s = sshcmd.ApplyHost(s, hc)
		}
		s = sshcmd.ApplyGroup(s, m.group)
		if modifySettings != nil {
			modifySettings(&s)
		}
		cmd, _ := sshcmd.BuildCommand(h, s)
		cmds = append(cmds, cmd)
	}
	return cmds
}

func (m *groupHostsModel) handleConnect() tea.Cmd {
	hosts := m.ghHostsToOpen()
	if len(hosts) == 0 {
		m.toast = toast{text: "no host selected", level: toastWarn}
		return nil
	}

	doConnect := func() tea.Cmd {
		mode, inTmux := m.resolveGroupMode()
		sshCmds := m.buildGroupSSHCmds(hosts, nil)

		res, cmd := dispatchConnect(hosts, sshCmds, m.opts.Config.Defaults, &m.group, mode, inTmux)
		if !res.toast.empty() {
			m.toast = res.toast
		}
		if res.quit {
			m.execCmd = res.execCmd
			return tea.Quit
		}
		return cmd
	}

	if len(hosts) > connectThreshold(m.opts.Config.Defaults) {
		m.confirmConnect = true
		m.confirmConnectCount = len(hosts)
		m.confirmConnectHosts = hosts
		m.pendingConnectFn = doConnect
		return nil
	}
	return doConnect()
}

func (m *groupHostsModel) handleConnectWithRemoteCommand(remoteCmd string) tea.Cmd {
	hosts := m.ghHostsToOpen()
	if len(hosts) == 0 {
		m.toast = toast{text: "no host selected", level: toastWarn}
		return nil
	}

	remoteCmd = strings.TrimSpace(remoteCmd)
	if remoteCmd == "" {
		m.toast = toast{text: "command required", level: toastWarn}
		return nil
	}

	doConnect := func() tea.Cmd {
		mode, inTmux := m.resolveGroupMode()
		sshCmds := m.buildGroupSSHCmds(hosts, func(s *sshcmd.Settings) {
			s.ExtraArgs = ensureSSHForceTTY(s.ExtraArgs)
			s.RemoteCommand = keepSessionOpenRemoteCmd(remoteCmd)
		})

		res, cmd := dispatchConnect(hosts, sshCmds, m.opts.Config.Defaults, &m.group, mode, inTmux)
		if !res.toast.empty() {
			m.toast = res.toast
		}
		if res.quit {
			m.execCmd = res.execCmd
			return tea.Quit
		}
		return cmd
	}

	if len(hosts) > connectThreshold(m.opts.Config.Defaults) {
		m.confirmConnect = true
		m.confirmConnectCount = len(hosts)
		m.confirmConnectHosts = hosts
		m.pendingConnectFn = doConnect
		return nil
	}
	return doConnect()
}

func (m *groupHostsModel) handleConnectSame() tea.Cmd {
	hosts := m.ghHostsToOpen()
	if len(hosts) == 0 {
		m.toast = toast{text: "no host selected", level: toastWarn}
		return nil
	}
	if len(hosts) > 1 {
		m.toast = toast{text: "select single host for same-window connect", level: toastWarn}
		return nil
	}
	sshCmds := m.buildGroupSSHCmds(hosts, nil)
	m.execCmd = sshCmds[0]
	return tea.Quit
}

func (m *groupHostsModel) openOneWindow() tea.Cmd {
	hosts := m.ghHostsToOpen()
	if len(hosts) == 0 {
		m.toast = toast{text: "no host selected", level: toastWarn}
		return nil
	}
	if !tmx.InTmux() {
		m.toast = toast{text: "requires an active tmux session", level: toastWarn}
		return nil
	}

	doConnect := func() tea.Cmd {
		sshCmds := m.buildGroupSSHCmds(hosts, nil)
		defaults := m.opts.Config.Defaults
		group := m.group
		return func() tea.Msg {
			ps := resolvePaneSettings(defaults, &group, len(sshCmds))
			name := tmuxWindowName(hosts, &group)
			err := tmuxOpenOneWindow(sshCmds, tmuxOneWindowOpts{
				WindowName:       name,
				PaneTitles:       hosts,
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
	}

	if len(hosts) > connectThreshold(m.opts.Config.Defaults) {
		m.confirmConnect = true
		m.confirmConnectCount = len(hosts)
		m.confirmConnectHosts = hosts
		m.pendingConnectFn = doConnect
		return nil
	}
	return doConnect()
}

func (m *groupHostsModel) IsQuitting() bool  { return m.quitting }
func (m *groupHostsModel) ExecCmd() []string { return m.execCmd }
