package ui

// Layout values shared by several screens.
const (
	// tabBoxHeaderLines is the number of non-content lines consumed by the
	// tab-box header area: tabs + sep + header + sep.
	tabBoxHeaderLines = 4

	// compactWidthThreshold is the terminal width below which footers
	// switch to a compact single-line layout.
	compactWidthThreshold = 60

	// twoLineFooterMinHeight is the minimum terminal height for showing
	// a second footer line with additional key hints.
	twoLineFooterMinHeight = 20
)

func tabBoxListContentHeight(width, height int) int {
	innerH := max(0, height-2)
	footerLines := 1
	if width >= compactWidthThreshold && height >= twoLineFooterMinHeight {
		footerLines = 2
	}
	// tab/header lines + footer separator + footer lines.
	return max(1, innerH-tabBoxHeaderLines-1-footerLines)
}
