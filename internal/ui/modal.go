package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func placeCentered(fullW, fullH int, box string) string {
	box = strings.TrimRight(box, "\n")
	if fullW <= 0 || fullH <= 0 {
		return box
	}
	return lipgloss.Place(fullW, fullH, lipgloss.Center, lipgloss.Center, box)
}

// modalSpec sizes a modal: capped at maxW x maxH with margins kept free, and
// as tall as the terminal allows (up to maxH) once it would drop below minH.
type modalSpec struct{ maxW, maxH, marginW, marginH, minH int }

var (
	groupFormModal  = modalSpec{110, 26, 4, 1, 16}
	hostFormModal   = modalSpec{96, 20, 6, 4, 12}
	pickerModal     = modalSpec{90, 22, 6, 6, 10}
	customHostModal = modalSpec{76, 9, 6, 10, 8}
	cmdPromptModal  = modalSpec{88, 9, 6, 10, 0}
)

func (s modalSpec) size(fullW, fullH int) (w, h int) {
	w, h = modalDim(fullW, s.maxW, s.marginW), modalDim(fullH, s.maxH, s.marginH)
	if fullH > 0 && h < s.minH {
		h = min(s.maxH, fullH)
	}
	return w, h
}

// modalDim is full minus margin, capped at maxV. Unknown size (full <= 0)
// gives maxV; a margin that leaves nothing gives full.
func modalDim(full, maxV, margin int) int {
	if full <= 0 {
		return maxV
	}
	if v := min(maxV, full-margin); v > 0 {
		return v
	}
	return full
}
