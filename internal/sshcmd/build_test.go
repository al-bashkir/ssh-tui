package sshcmd

import (
	"reflect"
	"testing"

	"github.com/al-bashkir/ssh-tui/internal/config"
)

func TestMerge(t *testing.T) {
	d := config.Defaults{User: "", Port: 22, IdentityFile: "", ExtraArgs: []string{"-o", "A=B"}}
	g := config.Group{User: "root", Port: 0, IdentityFile: "id", ExtraArgs: nil, RemoteCommand: ""}

	got := Merge(d, g)
	want := Settings{User: "root", Port: 22, IdentityFile: "id", ExtraArgs: []string{"-o", "A=B"}, RemoteCommand: ""}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got=%#v, want %#v", got, want)
	}
}

func TestBuildCommandRemoteCommand(t *testing.T) {
	cmd, err := BuildCommand("example.com", Settings{RemoteCommand: "echo hi"})
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	want := []string{"ssh", "example.com", "sh -c 'echo hi'"}
	if !reflect.DeepEqual(cmd, want) {
		t.Fatalf("cmd=%v, want %v", cmd, want)
	}
}

func TestBuildCommandPlainHost(t *testing.T) {
	cmd, err := BuildCommand("example.com", Settings{User: "me", Port: 2222, IdentityFile: "~/.ssh/id", ExtraArgs: []string{"-o", "StrictHostKeyChecking=no"}})
	if err != nil {
		t.Fatalf("error: %v", err)
	}

	want := []string{"ssh", "-i", "~/.ssh/id", "-p", "2222", "-o", "StrictHostKeyChecking=no", "me@example.com"}
	if !reflect.DeepEqual(cmd, want) {
		t.Fatalf("cmd=%v, want %v", cmd, want)
	}
}

func TestBuildCommandBracketHostForcesPort(t *testing.T) {
	cmd, err := BuildCommand("[10.10.10.10]:2201", Settings{User: "", Port: 2222})
	if err != nil {
		t.Fatalf("error: %v", err)
	}

	want := []string{"ssh", "-p", "2201", "10.10.10.10"}
	if !reflect.DeepEqual(cmd, want) {
		t.Fatalf("cmd=%v, want %v", cmd, want)
	}
}

func TestBuildCommandPort22Omitted(t *testing.T) {
	cmd, err := BuildCommand("example.com", Settings{Port: 22})
	if err != nil {
		t.Fatalf("error: %v", err)
	}

	want := []string{"ssh", "example.com"}
	if !reflect.DeepEqual(cmd, want) {
		t.Fatalf("cmd=%v, want %v", cmd, want)
	}
}
