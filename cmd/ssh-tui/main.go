package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"

	"github.com/al-bashkir/ssh-tui/internal/config"
	"github.com/al-bashkir/ssh-tui/internal/hosts"
	"github.com/al-bashkir/ssh-tui/internal/ui"
)

type multiFlag []string

func (m *multiFlag) String() string { return strings.Join(*m, ",") }
func (m *multiFlag) Set(v string) error {
	*m = append(*m, v)
	return nil
}

func main() {
	var configPath string
	var knownHosts multiFlag
	var noTmux bool
	var debug bool

	flag.StringVar(&configPath, "config", "", "path to config.toml (default: XDG config)")
	flag.Var(&knownHosts, "known-hosts", "known_hosts path (repeatable)")
	flag.BoolVar(&noTmux, "no-tmux", false, "disable tmux integration")
	flag.BoolVar(&debug, "debug", false, "enable debug logging")
	flag.Usage = usage
	flag.Parse()

	cfg, cfgPathUsed, err := config.Load(configPath)
	if err != nil {
		fatal(err)
	}
	if noTmux {
		cfg.Defaults.Tmux = "never"
	}

	knownPaths := []string(knownHosts)
	res := hosts.LoadResult{}
	var loadErrs []hosts.PathError
	if cfg.Defaults.LoadKnownHosts {
		res, loadErrs = hosts.LoadKnownHosts(knownPaths)
	} else {
		knownPaths = nil
		res.Hosts = config.ConfigHosts(cfg)
	}

	args := flag.Args()
	if len(args) == 0 {
		runTUI(ui.Options{
			ConfigPath:   cfgPathUsed,
			Config:       cfg,
			KnownHosts:   knownPaths,
			Hosts:        res.Hosts,
			SkippedLines: res.SkippedLines,
			LoadErrors:   loadErrs,
			Debug:        debug,
		})
		return
	}

	switch args[0] {
	case "connect", "c":
		runConnect(args[1:], cfg, noTmux)
	case "list", "l":
		runList(args[1:], cfg, res.Hosts)
	case "completion", "comp":
		runCompletion(args[1:])
	case "__complete":
		runInternalComplete(args[1:], cfg, res.Hosts)
	default:
		fatal(fmt.Errorf("unknown command %q\nUsage: ssh-tui [flags] [connect|list|completion] ...", args[0]))
	}
}

func runTUI(opts ui.Options) {
	if err := ui.Run(opts); err != nil {
		var req *ui.ExecRequest
		if errors.As(err, &req) {
			if err := execReplace(req.Cmd); err != nil {
				fatal(err)
			}
			return
		}
		if errors.Is(err, ui.ErrQuit) {
			return
		}
		fatal(err)
	}
}

func usage() {
	fmt.Fprintf(os.Stderr, `ssh-tui — terminal UI and CLI for SSH connections

Usage:
  ssh-tui [flags]                        launch interactive TUI
  ssh-tui [flags] connect host NAME      connect to a host
  ssh-tui [flags] connect group NAME     connect to all hosts in a group
  ssh-tui [flags] list hosts             print known hosts
  ssh-tui [flags] list groups            print configured groups
  ssh-tui completion bash|zsh            print shell completion script

Subcommand aliases:  connect=c  list=l  host=h  group=g  hosts=h  groups=g

Flags:
`)
	flag.PrintDefaults()
}

func fatal(err error) {
	_, _ = fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}

func execReplace(cmd []string) error {
	if len(cmd) == 0 {
		return errors.New("empty exec command")
	}
	path, err := exec.LookPath(cmd[0])
	if err != nil {
		return err
	}
	// #nosec G204 -- exec replaces current process with argv (no shell); cmd is constructed by the app.
	return syscall.Exec(path, cmd, os.Environ())
}
