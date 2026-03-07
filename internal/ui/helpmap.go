package ui

import "github.com/charmbracelet/bubbles/key"

// helpSection is a named group of key bindings shown in the help modal.
type helpSection struct {
	title string
	keys  []key.Binding
}

type helpMap struct {
	short    []key.Binding
	full     [][]key.Binding
	sections []helpSection // named sections for grouped rendering
}

func (h helpMap) ShortHelp() []key.Binding  { return h.short }
func (h helpMap) FullHelp() [][]key.Binding { return h.full }
