package ui

import (
	"fmt"
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
)

type groupHostsModel struct {
	hostSelectList
	opts Options

	width  int
	height int

	groupIndex int
	group      config.Group

	list   list.Model
	search textinput.Model
	focus  focusState

	keymap         keyMap
	help           help.Model
	showHelp       bool
	helpVP         viewport.Model
	cmdPrompt      bool
	cmdPromptCrumb string
	cmdInput       textinput.Model
	toast          toast

	confirmRemove       bool
	removeList          []string
	confirmConnect      bool
	confirmConnectCount int
	confirmConnectHosts []string
	pendingConnectFn    func() tea.Cmd
	quitting            bool
	execCmd             []string

	prevSearch  string
	navPendingG bool
}

func newGroupHostsModel(opts Options, groupIndex int) *groupHostsModel {
	g := config.Group{}
	if groupIndex >= 0 && groupIndex < len(opts.Inventory.Groups) {
		g = opts.Inventory.Groups[groupIndex]
	}

	items := make([]list.Item, 0, len(g.Hosts))
	for _, h := range g.Hosts {
		_, ok := hostConfigFor(opts.Inventory, h)
		items = append(items, hostRow{host: h, hasCfg: ok})
	}

	l := list.New(items, hostDelegate{}, 0, 0)
	l.Title = "Group: " + g.Name
	configureList(&l)

	search := newSearchInput()

	m := &groupHostsModel{
		hostSelectList: hostSelectList{
			allHosts: append([]string(nil), g.Hosts...),
			filtered: append([]string(nil), g.Hosts...),
			selected: make(map[string]bool),
		},
		opts:       opts,
		groupIndex: groupIndex,
		group:      g,
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
		m.list.SetSize(innerW, tabBoxListContentHeight(w, h))
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
		if handled, cmd := handleConfirmConnect(msg, &m.confirmConnect, &m.pendingConnectFn, &m.toast); handled {
			return m, cmd
		}

		if m.focus == focusList {
			if handleVimListNav(msg, &m.list, &m.navPendingG) {
				return m, nil
			}
		} else {
			m.navPendingG = false
		}

		if key.Matches(msg, m.keymap.HostsTab) {
			return m, func() tea.Msg { return switchScreenMsg{to: screenHosts} }
		}
		if key.Matches(msg, m.keymap.GroupsTab) {
			return m, func() tea.Msg { return switchScreenMsg{to: screenGroups} }
		}
		if key.Matches(msg, m.keymap.Settings) {
			return m, func() tea.Msg { return openDefaultsFormMsg{returnTo: screenGroupHosts} }
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
				row, ok := m.list.SelectedItem().(hostRow)
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
		if key.Matches(msg, m.keymap.Esc) || (key.Matches(msg, m.keymap.Quit) && m.focus != focusSearch) {
			// Esc/q priority: blur search → clear selection → clear search → back.
			if m.focus == focusSearch && m.search.Value() == "" {
				m.focus = focusList
				m.search.Blur()
				setSearchBarFocused(&m.search, false)
				return m, nil
			}
			if len(m.selected) > 0 {
				m.selected = make(map[string]bool)
				m.refreshVisibleSelection()
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
			targets := m.ghHostsToOpen()
			if len(targets) == 0 {
				return m, nil
			}
			prefix := "Groups > " + m.group.Name
			switch len(targets) {
			case 1:
				m.cmdPromptCrumb = prefix + " > " + targets[0]
			default:
				m.cmdPromptCrumb = fmt.Sprintf("%s > %d selected", prefix, len(targets))
			}
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
			row, ok := m.list.SelectedItem().(hostRow)
			if !ok || strings.TrimSpace(row.host) == "" {
				m.toast = toast{text: "no host selected", level: toastWarn}
				return m, nil
			}
			return m, func() tea.Msg { return openHostFormMsg{host: row.host, returnTo: screenGroupHosts} }
		}
		if key.Matches(msg, m.keymap.Copy) && m.focus == focusList {
			row, ok := m.list.SelectedItem().(hostRow)
			if !ok || strings.TrimSpace(row.host) == "" {
				m.toast = toast{text: "no host selected", level: toastWarn}
				return m, nil
			}
			hc, ok := hostConfigFor(m.opts.Inventory, row.host)
			if !ok {
				m.toast = toast{text: "no host config", level: toastWarn}
				return m, nil
			}
			hc.Host = suggestCopyHostKey(m.opts.Inventory, hc.Host)
			return m, func() tea.Msg { return openHostFormPrefillMsg{host: hc, returnTo: screenGroupHosts} }
		}

	case toastMsg:
		m.toast = toast(msg)
		if m.opts.Popup && msg.level != toastErr {
			m.quitting = true
			return m, tea.Quit
		}
		return m, nil
	}

	return m, updateSearchOrList(m.focus, &m.search, &m.list, &m.prevSearch, msg, m.applyFilter)
}

func (m *groupHostsModel) View() string {
	if m.showHelp {
		return renderHelpModalWithVP(m.width, m.height, "Group Hosts", m.help, m.helpKeys(), &m.helpVP)
	}
	if m.cmdPrompt {
		return renderCmdPromptModal(m.width, m.height, m.cmdPromptCrumb,
			"Connect and run a remote command (keeps session open).", m.cmdInput)
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
	if m.width < compactWidthThreshold {
		footer = styledFooter("\u21b5 connect  \u2423 select  esc back  ? help")
	} else {
		footer = styledFooter("\u21b5 connect  O pane  ·  \u2423 select  o panes  ·  Ctrl+o cmd  a add")
		if m.height >= twoLineFooterMinHeight {
			footer += "\n" + styledFooter("e config  c custom  d remove  y copy  ·  Alt+1 hosts  Alt+3 settings  esc back  ? help")
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
			m.keymap.HostsTab,
			m.keymap.GroupsTab,
			m.keymap.Settings,
			esc,
			m.keymap.Help,
		},
		full: [][]key.Binding{{
			m.list.KeyMap.CursorUp,
			m.list.KeyMap.CursorDown,
			m.list.KeyMap.PrevPage,
			m.list.KeyMap.NextPage,
			m.list.KeyMap.GoToStart,
			m.list.KeyMap.GoToEnd,
			m.keymap.HostsTab,
			m.keymap.GroupsTab,
			m.keymap.Settings,
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
		}},
		sections: []helpSection{
			{title: "Navigation", keys: []key.Binding{
				m.list.KeyMap.CursorUp,
				m.list.KeyMap.CursorDown,
				m.list.KeyMap.PrevPage,
				m.list.KeyMap.NextPage,
				m.list.KeyMap.GoToStart,
				m.list.KeyMap.GoToEnd,
				m.keymap.HostsTab,
				m.keymap.GroupsTab,
				m.keymap.Settings,
				m.keymap.FocusSearch,
				m.keymap.ToggleFocus,
				esc,
			}},
			{title: "Selection", keys: []key.Binding{
				m.keymap.ToggleSel,
				m.keymap.SelectAll,
				m.keymap.ClearSel,
			}},
			{title: "Connection", keys: []key.Binding{
				m.keymap.Connect,
				m.keymap.ConnectSame,
				m.keymap.ConnectCmd,
				m.keymap.OneWindow,
			}},
			{title: "Editing", keys: []key.Binding{
				m.keymap.AddHosts,
				m.keymap.CustomHost,
				m.keymap.HostConfig,
				m.keymap.Copy,
				remove,
			}},
			{title: "General", keys: []key.Binding{
				m.keymap.Help,
			}},
		},
	}
}

func (m *groupHostsModel) emptyStateView() string {
	return renderListEmptyState(m.width, m.height, m.search.Value(),
		dim.Render("No hosts in this group.")+"\n"+dim.Render("a \u2014 add hosts"))
}

func (m *groupHostsModel) applyFilter(query string) {
	m.hostSelectList.applyFilter(&m.list, query, m.opts.Inventory, nil, nil)
}

func (m *groupHostsModel) refreshVisibleSelection() {
	m.hostSelectList.refreshVisibleSelection(&m.list)
}

func (m *groupHostsModel) refreshVisibleBadges() {
	m.hostSelectList.refreshVisibleBadges(&m.list, m.opts.Inventory)
}

func (m *groupHostsModel) toggleCurrentSelection() {
	m.hostSelectList.toggleCurrentSelection(&m.list)
}

func (m *groupHostsModel) ghHostsToOpen() []string {
	if sel := m.selectedHosts(); len(sel) > 0 {
		return sel
	}
	row, ok := m.list.SelectedItem().(hostRow)
	if ok && row.host != "" {
		return []string{row.host}
	}
	return nil
}

func (m *groupHostsModel) buildGroupSSHCmds(hosts []string, modifySettings func(*sshcmd.Settings)) [][]string {
	return buildSSHCommands(hosts, m.opts.Config.Defaults, m.opts.Inventory, &m.group, modifySettings)
}

func (m *groupHostsModel) handleConnect() tea.Cmd {
	hosts := m.ghHostsToOpen()
	if len(hosts) == 0 {
		m.toast = toast{text: "no host selected", level: toastWarn}
		return nil
	}

	doConnect := func() tea.Cmd {
		mode, inTmux := resolveConnectMode(m.opts.Config.Defaults, &m.group)
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
		mode, inTmux := resolveConnectMode(m.opts.Config.Defaults, &m.group)
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
			ps := tmx.ResolvePaneSettings(defaults, &group, len(sshCmds))
			name := tmx.GroupWindowName(hosts, &group)
			err := tmx.OpenOneWindow(sshCmds, tmx.OneWindowOpts{
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
