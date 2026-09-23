package ui

import (
	"errors"

	"github.com/al-bashkir/ssh-tui/internal/config"

	tea "github.com/charmbracelet/bubbletea"
)

var ErrQuit = errors.New("quit")

type ExecRequest struct {
	Cmd []string
}

func (e *ExecRequest) Error() string { return "exec requested" }

type Options struct {
	ConfigPath    string
	Config        config.Config
	InventoryPath string
	Inventory     config.Inventory
	KnownHosts    []string
	Hosts         []string
	SkippedLines  int
	LoadErrors    []error
	Popup         bool // quit after any tmux connect (for tmux popup use)
}

func Run(opts Options) error {
	ApplyColorScheme(opts.Config.Defaults.Colorscheme, opts.Config.Defaults.AccentColor)

	m := newAppModel(opts)
	p := tea.NewProgram(m, tea.WithAltScreen())

	model, err := p.Run()
	if err != nil {
		return err
	}
	if st, ok := model.(*appModel); ok {
		if len(st.execCmd) != 0 {
			return &ExecRequest{Cmd: st.execCmd}
		}
		if st.quitting {
			return ErrQuit
		}
	}
	return nil
}
