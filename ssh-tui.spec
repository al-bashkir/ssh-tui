%global app_version 0.0.0

Name:           ssh-tui
Version:        %{app_version}
Release:        1%{?dist}
Summary:        Terminal UI for managing SSH connections

# Disable the empty debuginfo subpackage that Go stripped binaries produce.
%global debug_package %{nil}

License:        MIT
URL:            https://github.com/al-bashkir/ssh-tui

# Source tarball is prepared by .copr/Makefile, which runs go mod vendor
# before creating the archive, so the vendor/ directory is already present.
Source0:        %{name}-%{version}.tar.gz

BuildRequires:  golang >= 1.25.6
BuildRequires:  git-core

# ssh-tui shells out to the system ssh binary
Requires:       openssh-clients
# tmux is required only for multi-host sessions
Recommends:     tmux

%description
ssh-tui is a terminal UI for browsing and connecting to SSH hosts.
It reads your SSH config and known_hosts files and presents an
interactive host picker built with Bubble Tea. Single-host connections
exec ssh directly; multi-host sessions are managed through tmux.

%prep
%autosetup

%build
export CGO_ENABLED=0
go build \
    -trimpath \
    -mod=vendor \
    -ldflags "-s -w" \
    -o %{name} \
    ./cmd/ssh-tui

%install
install -Dpm 0755 %{name} %{buildroot}%{_bindir}/%{name}

# Shell completion scripts
./%{name} completion bash > %{name}.bash
install -Dpm 0644 %{name}.bash %{buildroot}%{_datadir}/bash-completion/completions/%{name}

./%{name} completion zsh > _ssh_tui
install -Dpm 0644 _ssh_tui %{buildroot}%{_datadir}/zsh/site-functions/_ssh_tui

%post
# Regenerate completion files so they always match the installed binary.
%{_bindir}/%{name} completion bash > %{_datadir}/bash-completion/completions/%{name} 2>/dev/null || :
%{_bindir}/%{name} completion zsh  > %{_datadir}/zsh/site-functions/_ssh_tui         2>/dev/null || :

%files
%license LICENSE
%doc README.md
%{_bindir}/%{name}
%{_datadir}/bash-completion/completions/%{name}
%{_datadir}/zsh/site-functions/_ssh_tui

%changelog
* Thu Aug 27 2026 Pavel Aksenov <41126916+al-bashkir@users.noreply.github.com> - 1.3.3-1
- Extract shared connect logic into internal/connect
- Remove dead sshcmd/ui wrappers and empty doc files
- Ship the MIT license file in the package
- Document Homebrew installation

* Tue May 12 2026 Pavel Aksenov <41126916+al-bashkir@users.noreply.github.com> - 1.3.2-1
- Do not quit the app when the search bar has focus
- Add MIT license

* Sat Apr 25 2026 Pavel Aksenov <41126916+al-bashkir@users.noreply.github.com> - 1.3.1-1
- Move tab and settings navigation to Alt shortcuts
- Add vim-style g/G list navigation
- Match list height to footer size to avoid clipped pages
- Update README and UI docs for current keybindings and usage

* Fri Mar 13 2026 Pavel Aksenov <41126916+al-bashkir@users.noreply.github.com> - 1.3.0-1
- Add named colorscheme support (dracula, nord, gruvbox, catppuccin, kanagawa)
- Preserve cursor position across filter updates and hide/toggle in all list views
- Highlight fuzzy search matches in accent/underline style
- Settings options use dropdown pickers; add show_field_help setting
- Groups sorted alphabetically; quit (q/Q) restricted to main tabs
- Inline field validation, disabled fields, grouped help modal sections

* Mon Mar 02 2026 Pavel Aksenov <41126916+al-bashkir@users.noreply.github.com> - 1.2.1-1
- Fix completion

* Sun Mar 01 2026 Pavel Aksenov <41126916+al-bashkir@users.noreply.github.com> - 1.2.0-1
- Host inventory (hosts, groups, hidden_hosts) moved to a separate
  hosts.toml file; first launch after upgrade migrates automatically
- Add -hosts flag to set the hosts.toml path independently of -config
- Add -popup flag: TUI quits after a successful tmux connect, intended
  for tmux popup use (e.g. bind-key f display-popup -E 'ssh-tui -popup')
- Shell completion extended to cover all flags including -hosts and -popup
- Badge column width is now stable as the cursor moves; cfg badge symbol
  changed from "cfg" text to ⚙
- Esc key in host/group-host views: blur search bar, then clear selection,
  then clear search text, then navigate back (was: search then navigate)
- Esc in the groups tab clears search text even when the list has focus
- Group picker shows pending host names above the search bar (up to 2 lines)
- Group picker status line shows filtered count vs total
- Groups help modal collapsed from 4 to 3 columns to avoid clipping on
  narrow terminals

* Fri Feb 20 2026 Pavel Aksenov <41126916+al-bashkir@users.noreply.github.com> - 1.1.0-1
- Toast notifications now carry severity levels (info/ok/warn/err) with
  distinct colors and auto-dismiss durations
- Modal titles show navigation breadcrumbs (e.g. Groups > prod > Add hosts)
- Remove-host confirmation dialog lists the affected host names
- Rename "Confirm at" setting label to "Confirm connect at"
- Rename module path to github.com/al-bashkir/ssh-tui

* Fri Feb 20 2026 Pavel Aksenov <41126916+al-bashkir@users.noreply.github.com> - 1.0.1-1
- Add zsh and bash completions

* Fri Feb 20 2026 Pavel Aksenov <41126916+al-bashkir@users.noreply.github.com> - 1.0.0-1
- Initial package
