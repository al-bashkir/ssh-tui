package ui

import "testing"

func TestModalSpecSize(t *testing.T) {
	cases := []struct {
		spec         modalSpec
		fullW, fullH int
		w, h         int
	}{
		{groupFormModal, 200, 60, 110, 26}, // capped at max
		{groupFormModal, 100, 12, 96, 12},  // below minH: full height
		{customHostModal, 80, 14, 74, 9},   // below minH: capped at maxH, not full
		{customHostModal, 0, 0, 76, 9},     // unknown size: max
		{cmdPromptModal, 50, 12, 44, 2},    // no minH fallback
	}
	for _, c := range cases {
		if w, h := c.spec.size(c.fullW, c.fullH); w != c.w || h != c.h {
			t.Errorf("%+v.size(%d,%d) = %d,%d, want %d,%d", c.spec, c.fullW, c.fullH, w, h, c.w, c.h)
		}
	}
}

func TestOptionPickerMoveWraps(t *testing.T) {
	p := newOptionPickerPopup("k", "K", "", opts("a", "b", "c"))
	p.move(-1)
	if p.cursor != 2 {
		t.Fatalf("up from 0: cursor=%d, want 2", p.cursor)
	}
	p.move(1)
	if p.cursor != 0 {
		t.Fatalf("down from 2: cursor=%d, want 0", p.cursor)
	}
}
