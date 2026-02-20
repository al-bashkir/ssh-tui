package main

import (
	"fmt"

	"github.com/al-bashkir/ssh-tui/internal/config"
)

func runCompletion(args []string) {
	if len(args) == 0 {
		fatal(fmt.Errorf("completion requires a shell: bash or zsh\nUsage: ssh-tui completion bash|zsh"))
	}
	switch args[0] {
	case "bash":
		fmt.Print(bashCompletionScript)
	case "zsh":
		fmt.Print(zshCompletionScript)
	default:
		fatal(fmt.Errorf("unknown shell %q: use bash or zsh", args[0]))
	}
}

// runInternalComplete is called by shell completion scripts to get dynamic candidates.
// It prints one entry per line and is intentionally silent on errors.
func runInternalComplete(args []string, cfg config.Config, knownHosts []string) {
	if len(args) == 0 {
		return
	}
	switch args[0] {
	case "groups":
		for _, g := range cfg.Groups {
			fmt.Println(g.Name)
		}
	case "hosts":
		for _, h := range knownHosts {
			fmt.Println(h)
		}
	}
	// unknown token → print nothing (graceful for completion scripts)
}

const bashCompletionScript = `# bash completion for ssh-tui
# Usage:
#   source <(ssh-tui completion bash)
# Or add to ~/.bashrc:
#   eval "$(ssh-tui completion bash)"

_ssh_tui() {
  COMPREPLY=()
  local cur="${COMP_WORDS[COMP_CWORD]}"
  local cmd="${COMP_WORDS[1]}"
  local subcmd="${COMP_WORDS[2]}"

  case $COMP_CWORD in
    1)
      COMPREPLY=($(compgen -W "connect c list l completion" -- "$cur"))
      ;;
    2)
      case $cmd in
        connect|c)
          COMPREPLY=($(compgen -W "group g host h" -- "$cur"))
          ;;
        list|l)
          COMPREPLY=($(compgen -W "groups g hosts h" -- "$cur"))
          ;;
        completion)
          COMPREPLY=($(compgen -W "bash zsh" -- "$cur"))
          ;;
      esac
      ;;
    3)
      case $cmd in
        connect|c)
          case $subcmd in
            group|g)
              COMPREPLY=($(compgen -W "$(ssh-tui __complete groups 2>/dev/null)" -- "$cur"))
              ;;
            host|h)
              COMPREPLY=($(compgen -W "$(ssh-tui __complete hosts 2>/dev/null)" -- "$cur"))
              ;;
          esac
          ;;
      esac
      ;;
  esac
}

complete -F _ssh_tui ssh-tui
`

const zshCompletionScript = `#compdef ssh-tui
# zsh completion for ssh-tui
# Usage (one-time setup):
#   mkdir -p ~/.zfunc
#   ssh-tui completion zsh > ~/.zfunc/_ssh_tui
# Then add to ~/.zshrc (before compinit):
#   fpath=(~/.zfunc $fpath)
#   autoload -Uz compinit && compinit

_ssh_tui() {
  local cmd="${words[2]}"
  local subcmd="${words[3]}"

  case $CURRENT in
    2)
      local -a cmds
      cmds=(
        'connect:connect to a host or group'
        'c:alias for connect'
        'list:list groups or hosts'
        'l:alias for list'
        'completion:output shell completion script'
      )
      _describe 'command' cmds
      ;;
    3)
      case $cmd in
        connect|c)
          local -a sub
          sub=(
            'group:connect to all hosts in a group'
            'g:alias for group'
            'host:connect to a specific host'
            'h:alias for host'
          )
          _describe 'subcommand' sub
          ;;
        list|l)
          local -a sub
          sub=(
            'groups:list all groups'
            'g:alias for groups'
            'hosts:list all known hosts'
            'h:alias for hosts'
          )
          _describe 'subcommand' sub
          ;;
        completion)
          local -a shells
          shells=('bash:bash completion script' 'zsh:zsh completion script')
          _describe 'shell' shells
          ;;
      esac
      ;;
    4)
      case $cmd in
        connect|c)
          case $subcmd in
            group|g)
              local -a groups
              groups=(${(f)"$(ssh-tui __complete groups 2>/dev/null)"})
              _describe 'group' groups
              ;;
            host|h)
              local -a hosts
              hosts=(${(f)"$(ssh-tui __complete hosts 2>/dev/null)"})
              _describe 'host' hosts
              ;;
          esac
          ;;
      esac
      ;;
  esac
}

_ssh_tui "$@"
`
