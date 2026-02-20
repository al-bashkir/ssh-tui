Name:           ssh-tui
Version:        1.0.0
Release:        1%{?dist}
Summary:        Terminal UI for managing SSH connections

# Disable the empty debuginfo subpackage that Go stripped binaries produce.
%global debug_package %{nil}

# No LICENSE file exists yet; update this field when one is added.
License:        MIT
URL:            https://github.com/bashkir/ssh-tui

# Source tarball is prepared by .copr/Makefile, which runs go mod vendor
# before creating the archive, so the vendor/ directory is already present.
Source0:        %{name}-%{version}.tar.gz

BuildRequires:  golang >= 1.21
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

%files
%doc README.md
%{_bindir}/%{name}

%changelog
* Thu Feb 20 2026 Pavel Aksenov <41126916+al-bashkir@users.noreply.github.com> - 1.0.0-1
- Initial package
