package ui

import "github.com/charmbracelet/bubbles/list"

func configureList(m *list.Model) {
	// Avoid default letter shortcuts that conflict with our app keys.
	km := list.DefaultKeyMap()
	// Keep vim-style page keys (h/l) but drop other letters (b/f, etc.).
	km.NextPage.SetKeys("right", "pgdown", "l")
	km.PrevPage.SetKeys("left", "pgup", "h")
	km.GoToStart.SetKeys("home")
	km.GoToStart.SetHelp("home", "go to start")
	km.GoToEnd.SetKeys("end")
	km.GoToEnd.SetHelp("end", "go to end")
	m.KeyMap = km

	// We render our own header/footer.
	m.SetShowTitle(false)
	m.SetShowPagination(false)
	m.SetShowHelp(false)
	m.SetShowStatusBar(false)
	m.SetFilteringEnabled(false)
	m.DisableQuitKeybindings()
}
