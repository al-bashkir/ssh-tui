package ui

import "github.com/charmbracelet/bubbles/key"

type helpMap struct {
	short []key.Binding
	full  [][]key.Binding
}

func (h helpMap) ShortHelp() []key.Binding  { return h.short }
func (h helpMap) FullHelp() [][]key.Binding { return h.full }
