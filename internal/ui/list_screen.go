package ui

import (
	"fmt"
	"io"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

type focusState int

const (
	focusList focusState = iota
	focusSearch
)

// rowDelegate renders one-line list rows with its function.
type rowDelegate func(width int, active bool, item list.Item) string

func (d rowDelegate) Height() int                         { return 1 }
func (d rowDelegate) Spacing() int                        { return 0 }
func (d rowDelegate) Update(tea.Msg, *list.Model) tea.Cmd { return nil }
func (d rowDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	_, _ = fmt.Fprint(w, d(m.Width(), index == m.Index(), item))
}

// listScreen is the state and behaviour shared by the full-screen lists
// (Hosts, Groups, Group Hosts). Screens embed it, so its fields are promoted.
type listScreen struct {
	connectActions
	opts Options

	width  int
	height int

	keymap      keyMap
	list        list.Model
	search      textinput.Model
	focus       focusState
	prevSearch  string
	navPendingG bool
	confirmQuit bool

	showHelp       bool
	helpVP         viewport.Model
	cmdPrompt      bool
	cmdPromptCrumb string
	cmdInput       textinput.Model
}

func newListScreen(opts Options, delegate list.ItemDelegate) listScreen {
	l := list.New(nil, delegate, 0, 0)
	configureList(&l)
	return listScreen{
		opts:   opts,
		keymap: defaultKeyMap(),
		list:   l,
		search: newSearchInput(),
		focus:  focusList,
	}
}

// resize fits the list and search bar to the window; searchReserve is the
// header width kept free for the right-hand status.
func (s *listScreen) resize(msg tea.WindowSizeMsg, searchReserve int) {
	s.width = msg.Width
	s.height = msg.Height
	innerW := max(0, msg.Width-2)
	s.list.SetSize(innerW, tabBoxListContentHeight(msg.Width, msg.Height))
	s.search.Width = max(10, innerW-searchReserve-len(s.search.Prompt))
}

// handleOverlayKey routes a key to the open overlay (help, command prompt,
// quit or connect confirm). It reports false when no overlay is open.
func (s *listScreen) handleOverlayKey(msg tea.KeyMsg, runCmd func(string) tea.Cmd) (bool, tea.Cmd) {
	if s.showHelp {
		if key.Matches(msg, s.keymap.Help) || msg.String() == "esc" {
			s.showHelp = false
		} else {
			updateHelpViewport(&s.helpVP, msg)
		}
		return true, nil
	}
	if s.cmdPrompt {
		return true, handleCmdPromptKey(msg, &s.cmdPrompt, &s.cmdInput, &s.toast, runCmd)
	}
	if handled, cmd := handleConfirmQuit(msg, &s.confirmQuit, &s.toast, &s.quitting); handled {
		return true, cmd
	}
	return handleConfirmConnect(msg, &s.confirmConnect, &s.pendingConnectFn, &s.toast)
}

// handleVimNav handles gg/G on the list.
func (s *listScreen) handleVimNav(msg tea.KeyMsg) bool {
	if s.focus != focusList {
		s.navPendingG = false
		return false
	}
	return handleVimListNav(msg, &s.list, &s.navPendingG)
}

// tabCmd switches to the tab its key names, other than the current one.
func (s *listScreen) tabCmd(msg tea.KeyMsg, current screen) tea.Cmd {
	to := current
	switch {
	case key.Matches(msg, s.keymap.HostsTab):
		to = screenHosts
	case key.Matches(msg, s.keymap.GroupsTab):
		to = screenGroups
	case key.Matches(msg, s.keymap.Settings):
		return func() tea.Msg { return openDefaultsFormMsg{returnTo: current} }
	}
	if to == current {
		return nil
	}
	return func() tea.Msg { return switchScreenMsg{to: to} }
}

// quit exits, or asks first when confirm_quit is set.
func (s *listScreen) quit() tea.Cmd {
	if !s.opts.Config.Defaults.ConfirmQuit {
		s.quitting = true
		return tea.Quit
	}
	s.confirmQuit = true
	s.toast = toast{text: "quit? (y/n)", level: toastWarn}
	return nil
}

func (s *listScreen) openHelp(title string, sections []helpSection) {
	s.showHelp = true
	if s.width > 0 && s.height > 0 {
		s.helpVP = initHelpViewport(s.width, s.height, title, sections)
	}
}

func (s *listScreen) openCmdPrompt(crumb string) {
	s.cmdPromptCrumb = crumb
	s.cmdInput = newCmdPromptInput(s.width, s.height)
	s.cmdPrompt = true
}

// showToast displays a toast from an async connect; popup mode quits after
// any successful tmux connect.
func (s *listScreen) showToast(t toast) tea.Cmd {
	s.toast = t
	if s.opts.Popup && t.level != toastErr {
		s.quitting = true
		return tea.Quit
	}
	return nil
}

// overlayView renders the open overlay, or "" when none is open.
func (s *listScreen) overlayView(title string, helpKeys func() []helpSection, cmdDesc string) string {
	switch {
	case s.showHelp:
		return renderHelpModal(s.width, s.height, title, helpKeys(), &s.helpVP)
	case s.cmdPrompt:
		return renderCmdPromptModal(s.width, s.height, s.cmdPromptCrumb, cmdDesc, s.cmdInput)
	case s.confirmQuit:
		return renderQuitConfirm(s.width, s.height)
	case s.confirmConnect:
		return placeCentered(s.width, s.height, connectConfirmBox(max(0, s.width-4), s.confirmConnectHosts))
	}
	return ""
}

func (s *listScreen) refreshAccentStyles() {
	setSearchBarFocused(&s.search, s.focus == focusSearch)
	if s.cmdPrompt {
		setSearchFocused(&s.cmdInput, true)
	}
}

// listNavKeys are the navigation bindings every list screen's help shows.
func (s *listScreen) listNavKeys() []key.Binding {
	return []key.Binding{
		s.list.KeyMap.CursorUp,
		s.list.KeyMap.CursorDown,
		s.list.KeyMap.PrevPage,
		s.list.KeyMap.NextPage,
		s.list.KeyMap.GoToStart,
		s.list.KeyMap.GoToEnd,
	}
}
