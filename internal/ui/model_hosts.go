package ui

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/bashkir/ssh-tui/internal/config"
	"github.com/bashkir/ssh-tui/internal/hosts"
	"github.com/bashkir/ssh-tui/internal/sshcmd"
	tmx "github.com/bashkir/ssh-tui/internal/tmux"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/sahilm/fuzzy"
)

type focusState int

const (
	focusList focusState = iota
	focusSearch
)

type hostRow struct {
	host     string
	selected bool
	hasCfg   bool
	hidden   bool
}

func (i hostRow) Title() string       { return i.host }
func (i hostRow) Description() string { return "" }
func (i hostRow) FilterValue() string { return i.host }

type hostDelegate struct{}

func (d hostDelegate) Height() int                             { return 1 }
func (d hostDelegate) Spacing() int                            { return 0 }
func (d hostDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }

func (d hostDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	row, ok := item.(hostRow)
	if !ok {
		fmt.Fprint(w, item.FilterValue())
		return
	}
	fmt.Fprint(w, renderHostLikeRow(m.Width(), index == m.Index(), row.selected, row.host, row.hasCfg, row.hidden))
}

type knownHostsReloadMsg struct {
	res  hosts.LoadResult
	errs []hosts.PathError
}

type hostsModel struct {
	opts Options

	width  int
	height int

	allHosts []string
	filtered []string
	selected map[string]bool
	keymap   keyMap
	help     help.Model

	list   list.Model
	search textinput.Model
	focus  focusState

	reloading   bool
	showHidden  bool
	prevSearch  string
	toast       string
	confirmQuit bool

	confirmConnect      bool
	confirmConnectCount int
	confirmConnectHosts []string
	pendingConnectFn    func() tea.Cmd

	quitting  bool
	showHelp  bool
	helpVP    viewport.Model
	cmdPrompt bool
	cmdInput  textinput.Model
	execCmd   []string
}

func newHostsModel(opts Options) *hostsModel {
	items := make([]list.Item, 0, len(opts.Hosts))
	for _, h := range opts.Hosts {
		_, ok := hostConfigFor(opts.Config, h)
		items = append(items, hostRow{host: h, hasCfg: ok})
	}

	delegate := hostDelegate{}
	l := list.New(items, delegate, 0, 0)
	l.Title = "Hosts"
	configureList(&l)

	search := textinput.New()
	search.Placeholder = "search"
	search.Prompt = "/ "
	search.CharLimit = 256
	search.Width = 40
	configureSearch(&search)
	setSearchBarFocused(&search, false)

	m := &hostsModel{
		opts:     opts,
		allHosts: append([]string(nil), opts.Hosts...),
		filtered: append([]string(nil), opts.Hosts...),
		selected: make(map[string]bool),
		keymap:   defaultKeyMap(),
		help:     help.New(),
		list:     l,
		search:   search,
		focus:    focusList,
	}
	if !opts.Config.Defaults.LoadKnownHosts {
		m.keymap.Reload.SetEnabled(false)
	}
	m.applyFilter("")
	return m
}

func (m *hostsModel) Init() tea.Cmd {
	return nil
}

func (m *hostsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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

	case knownHostsReloadMsg:
		spinnerStop()
		m.reloading = false
		m.opts.Hosts = msg.res.Hosts
		m.opts.SkippedLines = msg.res.SkippedLines
		m.opts.LoadErrors = msg.errs
		m.allHosts = append([]string(nil), msg.res.Hosts...)
		present := make(map[string]struct{}, len(m.allHosts))
		for _, h := range m.allHosts {
			present[h] = struct{}{}
		}
		for h := range m.selected {
			if _, ok := present[h]; !ok {
				delete(m.selected, h)
			}
		}
		m.applyFilter(m.search.Value())
		m.toast = fmt.Sprintf("%d hosts loaded", len(m.allHosts))
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
				m.toast = ""
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
				m.toast = ""
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
				m.toast = ""
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
			m.toast = "quit? (y/n)"
			return m, nil
		}
		if key.Matches(msg, m.keymap.Help) {
			m.showHelp = !m.showHelp
			if m.showHelp && m.width > 0 && m.height > 0 {
				m.helpVP = initHelpViewport(m.width, m.height, "Hosts", m.help, m.helpKeys())
			}
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
		if key.Matches(msg, m.keymap.SwitchTab) && m.focus != focusSearch {
			return m, func() tea.Msg { return switchScreenMsg{to: screenGroups} }
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
		}

		if key.Matches(msg, m.keymap.Reload) && m.focus != focusSearch {
			if !m.opts.Config.Defaults.LoadKnownHosts {
				m.toast = "known_hosts disabled"
				return m, nil
			}
			m.toast = "reloading"
			m.reloading = true
			spinnerStart()
			return m, tea.Batch(
				reloadKnownHostsCmd(m.opts.KnownHosts),
				tea.Tick(spinnerTickInterval, func(time.Time) tea.Msg { return spinnerTickMsg{} }),
			)
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
			// In search focus, Enter should accept the query and go back to list.
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
			m.toast = ""
			return m, m.handleConnect()
		}
		if key.Matches(msg, m.keymap.OneWindow) && m.focus == focusList {
			m.toast = ""
			return m, m.openOneWindow()
		}
		if key.Matches(msg, m.keymap.ConnectSame) && m.focus == focusList {
			m.toast = ""
			return m, m.handleConnectSame()
		}
		if key.Matches(msg, m.keymap.AddHosts) && m.focus == focusList {
			hostsToAdd := m.selectedHosts()
			if len(hostsToAdd) == 0 {
				row, ok := m.list.SelectedItem().(hostRow)
				if ok && row.host != "" {
					hostsToAdd = []string{row.host}
				}
			}
			if len(hostsToAdd) == 0 {
				m.toast = "no host selected"
				return m, nil
			}
			return m, func() tea.Msg { return openGroupPickerMsg{hosts: hostsToAdd} }
		}
		if key.Matches(msg, m.keymap.CustomHost) && m.focus == focusList {
			return m, func() tea.Msg { return openCustomHostMsg{returnTo: screenHosts, groupIndex: -1} }
		}
		if key.Matches(msg, m.keymap.HostConfig) && m.focus == focusList {
			row, ok := m.list.SelectedItem().(hostRow)
			if !ok || strings.TrimSpace(row.host) == "" {
				m.toast = "no host selected"
				return m, nil
			}
			return m, func() tea.Msg { return openHostFormMsg{host: row.host, returnTo: screenHosts} }
		}
		if key.Matches(msg, m.keymap.Copy) && m.focus == focusList {
			row, ok := m.list.SelectedItem().(hostRow)
			if !ok || strings.TrimSpace(row.host) == "" {
				m.toast = "no host selected"
				return m, nil
			}
			hc, ok := hostConfigFor(m.opts.Config, row.host)
			if !ok {
				m.toast = "no host config"
				return m, nil
			}
			hc.Host = suggestCopyHostKey(m.opts.Config, hc.Host)
			return m, func() tea.Msg { return openHostFormPrefillMsg{host: hc, returnTo: screenHosts} }
		}
		if key.Matches(msg, m.keymap.Settings) && m.focus == focusList {
			return m, func() tea.Msg { return openDefaultsFormMsg{returnTo: screenHosts} }
		}
		if key.Matches(msg, m.keymap.HideHost) && m.focus == focusList {
			return m, m.toggleCurrentHidden()
		}
		if key.Matches(msg, m.keymap.ShowHidden) && m.focus == focusList {
			m.showHidden = !m.showHidden
			m.applyFilter(m.search.Value())
			return m, nil
		}

	case toastMsg:
		m.toast = string(msg)
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

func (m *hostsModel) applyFilter(query string) {
	query = strings.TrimSpace(query)
	var filtered []string
	if query == "" {
		filtered = append([]string(nil), m.allHosts...)
	} else {
		matches := fuzzy.Find(query, m.allHosts)
		filtered = make([]string, 0, len(matches))
		for _, match := range matches {
			filtered = append(filtered, match.Str)
		}
	}
	if !m.showHidden {
		visible := make([]string, 0, len(filtered))
		for _, h := range filtered {
			if isHostHidden(m.opts.Config, h) {
				continue
			}
			visible = append(visible, h)
		}
		filtered = visible
	}
	m.filtered = filtered
	m.setListItems(m.filtered)
}

func (m *hostsModel) setListItems(hosts []string) {
	items := make([]list.Item, 0, len(hosts))
	for _, h := range hosts {
		_, ok := hostConfigFor(m.opts.Config, h)
		hidden := isHostHidden(m.opts.Config, h)
		items = append(items, hostRow{host: h, selected: m.selected[h], hasCfg: ok, hidden: hidden})
	}
	m.list.SetItems(items)
	if len(items) > 0 {
		m.list.Select(0)
	}
}

type toastMsg string

func (m *hostsModel) refreshVisibleSelection() {
	items := m.list.Items()
	for i := range items {
		row, ok := items[i].(hostRow)
		if !ok {
			continue
		}
		row.selected = m.selected[row.host]
		items[i] = row
	}
	m.list.SetItems(items)
}

func (m *hostsModel) refreshVisibleBadges() {
	idx := m.list.Index()
	items := m.list.Items()
	for i := range items {
		row, ok := items[i].(hostRow)
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

func (m *hostsModel) toggleCurrentHidden() tea.Cmd {
	row, ok := m.list.SelectedItem().(hostRow)
	if !ok || row.host == "" {
		return nil
	}
	hide := !isHostHidden(m.opts.Config, row.host)
	return func() tea.Msg { return toggleHiddenHostMsg{host: row.host, hide: hide} }
}

func (m *hostsModel) reapplyFilter() {
	m.applyFilter(m.search.Value())
}

func (m *hostsModel) hiddenCount() int {
	count := 0
	for _, h := range m.allHosts {
		if isHostHidden(m.opts.Config, h) {
			count++
		}
	}
	return count
}

func (m *hostsModel) toggleCurrentSelection() {
	row, ok := m.list.SelectedItem().(hostRow)
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

func (m *hostsModel) selectedHosts() []string {
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

func (m *hostsModel) hostsToOpen() []string {
	if sel := m.selectedHosts(); len(sel) > 0 {
		return sel
	}
	row, ok := m.list.SelectedItem().(hostRow)
	if ok && row.host != "" {
		return []string{row.host}
	}
	return nil
}

func (m *hostsModel) buildSSHCmds(hosts []string, modifySettings func(*sshcmd.Settings)) [][]string {
	base := sshcmd.FromDefaults(m.opts.Config.Defaults)
	cmds := make([][]string, 0, len(hosts))
	for _, h := range hosts {
		s := base
		if hc, ok := hostConfigFor(m.opts.Config, h); ok {
			s = sshcmd.ApplyHost(s, hc)
		}
		if modifySettings != nil {
			modifySettings(&s)
		}
		cmd, _ := sshcmd.BuildCommand(h, s)
		cmds = append(cmds, cmd)
	}
	return cmds
}

func connectThreshold(d config.Defaults) int {
	if d.ConnectConfirmThreshold < 0 {
		return 5
	}
	return d.ConnectConfirmThreshold
}

func (m *hostsModel) handleConnect() tea.Cmd {
	hosts := m.hostsToOpen()
	if len(hosts) == 0 {
		m.toast = "no host selected"
		return nil
	}

	doConnect := func() tea.Cmd {
		defaults := m.opts.Config.Defaults
		inTmux := tmx.InTmux()
		mode := tmx.ResolveOpenMode(defaults.Tmux, defaults.OpenMode, inTmux)
		sshCmds := m.buildSSHCmds(hosts, nil)

		res, cmd := dispatchConnect(hosts, sshCmds, defaults, nil, mode, inTmux)
		if res.toast != "" {
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

func (m *hostsModel) handleConnectWithRemoteCommand(remoteCmd string) tea.Cmd {
	hosts := m.hostsToOpen()
	if len(hosts) == 0 {
		m.toast = "no host selected"
		return nil
	}

	remoteCmd = strings.TrimSpace(remoteCmd)
	if remoteCmd == "" {
		m.toast = "command required"
		return nil
	}

	doConnect := func() tea.Cmd {
		defaults := m.opts.Config.Defaults
		inTmux := tmx.InTmux()
		mode := tmx.ResolveOpenMode(defaults.Tmux, defaults.OpenMode, inTmux)
		sshCmds := m.buildSSHCmds(hosts, func(s *sshcmd.Settings) {
			s.ExtraArgs = ensureSSHForceTTY(s.ExtraArgs)
			s.RemoteCommand = keepSessionOpenRemoteCmd(remoteCmd)
		})

		res, cmd := dispatchConnect(hosts, sshCmds, defaults, nil, mode, inTmux)
		if res.toast != "" {
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

func (m *hostsModel) handleConnectSame() tea.Cmd {
	hosts := m.hostsToOpen()
	if len(hosts) == 0 {
		m.toast = "no host selected"
		return nil
	}
	if len(hosts) > 1 {
		m.toast = "select single host for same-window connect"
		return nil
	}
	sshCmds := m.buildSSHCmds(hosts, nil)
	m.execCmd = sshCmds[0]
	return tea.Quit
}

func (m *hostsModel) openOneWindow() tea.Cmd {
	hosts := m.hostsToOpen()
	if len(hosts) == 0 {
		m.toast = "no host selected"
		return nil
	}
	if !tmx.InTmux() {
		m.toast = "requires an active tmux session"
		return nil
	}

	doConnect := func() tea.Cmd {
		sshCmds := m.buildSSHCmds(hosts, nil)
		defaults := m.opts.Config.Defaults
		return func() tea.Msg {
			ps := resolvePaneSettings(defaults, nil, len(sshCmds))
			err := tmuxOpenOneWindow(sshCmds, tmuxOneWindowOpts{
				WindowName:       windowName(hosts[0]),
				PaneTitles:       hosts,
				SplitFlag:        ps.SplitFlag,
				Layout:           ps.Layout,
				SyncPanes:        ps.SyncPanes,
				PaneBorderFormat: ps.BorderFormat,
				PaneBorderStatus: ps.BorderStatus,
			})
			if err != nil {
				return toastMsg(err.Error())
			}
			return toastMsg(fmt.Sprintf("opened %d in one window", len(sshCmds)))
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

func windowName(host string) string {
	return tmx.WindowName(host)
}

func (m *hostsModel) helpKeys() helpMap {
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
			m.keymap.Settings,
			m.keymap.Reload,
			m.keymap.SwitchTab,
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
			m.keymap.ToggleSel,
			m.keymap.SelectAll,
			m.keymap.ClearSel,
		}, {
			m.keymap.Connect,
			m.keymap.ConnectSame,
			m.keymap.ConnectCmd,
			m.keymap.OneWindow,
			m.keymap.AddHosts,
			m.keymap.CustomHost,
			m.keymap.HostConfig,
			m.keymap.Copy,
			m.keymap.Settings,
			m.keymap.Reload,
			m.keymap.Help,
			m.keymap.Quit,
		}},
	}
}

func reloadKnownHostsCmd(paths []string) tea.Cmd {
	return func() tea.Msg {
		res, errs := hosts.LoadKnownHosts(paths)
		return knownHostsReloadMsg{res: res, errs: errs}
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
