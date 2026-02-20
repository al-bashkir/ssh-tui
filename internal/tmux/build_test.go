package tmux

import (
	"reflect"
	"testing"
)

func TestResolveOpenMode(t *testing.T) {
	tc := []struct {
		name     string
		tmux     string
		openMode string
		inTmux   bool
		want     OpenMode
	}{
		{"never forces current", "never", "tmux-window", true, OpenCurrent},
		{"force defaults to window", "force", "current", false, OpenWindow},
		{"force pane", "force", "tmux-pane", false, OpenPane},
		{"auto not in tmux defaults current", "auto", "auto", false, OpenCurrent},
		{"auto in tmux defaults window", "auto", "auto", true, OpenWindow},
		{"explicit window", "auto", "tmux-window", false, OpenWindow},
	}

	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			got := ResolveOpenMode(tt.tmux, tt.openMode, tt.inTmux)
			if got != tt.want {
				t.Fatalf("got=%q want=%q", got, tt.want)
			}
		})
	}
}

func TestCmdBuilders(t *testing.T) {
	ssh := []string{"ssh", "-p", "2222", "host"}

	if got, want := NewWindowCmd("h", ssh), []string{"tmux", "new-window", "-n", "h", "--", "ssh", "-p", "2222", "host"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("NewWindowCmd=%v want=%v", got, want)
	}
	if got, want := SplitPaneCmd(ssh), []string{"tmux", "split-window", "-h", "--", "ssh", "-p", "2222", "host"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("SplitPaneCmd=%v want=%v", got, want)
	}
	if got, want := SplitPaneCmdFlag("-v", ssh), []string{"tmux", "split-window", "-v", "--", "ssh", "-p", "2222", "host"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("SplitPaneCmdFlag=%v want=%v", got, want)
	}
	if got, want := NewSessionCmd("sess", ssh), []string{"tmux", "new-session", "-A", "-s", "sess", "--", "ssh", "-p", "2222", "host"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("NewSessionCmd=%v want=%v", got, want)
	}
}
