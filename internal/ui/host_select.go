package ui

import (
	"strings"

	"github.com/al-bashkir/ssh-tui/internal/config"
	"github.com/charmbracelet/bubbles/list"

	"github.com/sahilm/fuzzy"
)

// hostSelectList provides common host selection, filtering, and list management
// logic shared by hostsModel, groupHostsModel, and hostPickerModel.
//
// It is embedded (not composed) in each model so that fields like allHosts,
// filtered, selected, and matchIdxes are promoted to the outer struct.
// Each model keeps thin wrapper methods (applyFilter, refreshVisibleSelection,
// etc.) that delegate to these methods with the appropriate list.Model pointer
// and config.Inventory.
type hostSelectList struct {
	allHosts   []string
	filtered   []string
	selected   map[string]bool
	matchIdxes map[string][]int // fuzzy match indexes per host
}

// setListItems rebuilds list items from the current filtered hosts.
// hiddenFn, when non-nil, is called to set the display-only hidden flag.
func (s *hostSelectList) setListItems(
	l *list.Model,
	inv config.Inventory,
	hiddenFn func(config.Inventory, string) bool,
) {
	items := make([]list.Item, 0, len(s.filtered))
	for _, h := range s.filtered {
		_, hasCfg := hostConfigFor(inv, h)
		hidden := hiddenFn != nil && hiddenFn(inv, h)
		items = append(items, hostRow{
			host:           h,
			selected:       s.selected[h],
			hasCfg:         hasCfg,
			hidden:         hidden,
			matchedIndexes: s.matchIdxes[h],
		})
	}
	l.SetItems(items)
}

// refreshVisibleSelection updates the selected state on visible list items.
func (s *hostSelectList) refreshVisibleSelection(l *list.Model) {
	items := l.Items()
	for i := range items {
		row, ok := items[i].(hostRow)
		if !ok {
			continue
		}
		row.selected = s.selected[row.host]
		items[i] = row
	}
	l.SetItems(items)
}

// refreshVisibleBadges updates hasCfg badges on visible list items.
func (s *hostSelectList) refreshVisibleBadges(l *list.Model, inv config.Inventory) {
	idx := l.Index()
	items := l.Items()
	for i := range items {
		row, ok := items[i].(hostRow)
		if !ok {
			continue
		}
		_, ok = hostConfigFor(inv, row.host)
		row.hasCfg = ok
		items[i] = row
	}
	l.SetItems(items)
	if idx >= 0 && idx < len(items) {
		l.Select(idx)
	}
}

// toggleCurrentSelection toggles selection on the currently focused host.
func (s *hostSelectList) toggleCurrentSelection(l *list.Model) {
	row, ok := l.SelectedItem().(hostRow)
	if !ok || row.host == "" {
		return
	}
	if s.selected[row.host] {
		delete(s.selected, row.host)
	} else {
		s.selected[row.host] = true
	}
	s.refreshVisibleSelection(l)
}

// selectedHosts returns hosts that are currently selected, preserving
// the original allHosts order.
func (s *hostSelectList) selectedHosts() []string {
	if len(s.selected) == 0 {
		return nil
	}
	out := make([]string, 0, len(s.selected))
	for _, h := range s.allHosts {
		if s.selected[h] {
			out = append(out, h)
		}
	}
	return out
}

// applyFilter fuzzy-filters hosts, optionally excludes entries, and preserves
// cursor position. excludeFn removes entries from the result when it returns
// true. hiddenFn is forwarded to setListItems for display badges.
func (s *hostSelectList) applyFilter(
	l *list.Model,
	query string,
	inv config.Inventory,
	excludeFn func(config.Inventory, string) bool,
	hiddenFn func(config.Inventory, string) bool,
) {
	var prevHost string
	if row, ok := l.SelectedItem().(hostRow); ok {
		prevHost = row.host
	}

	query = strings.TrimSpace(query)
	s.matchIdxes = nil
	if query == "" {
		s.filtered = append([]string(nil), s.allHosts...)
	} else {
		matches := fuzzy.Find(query, s.allHosts)
		s.filtered = make([]string, 0, len(matches))
		s.matchIdxes = make(map[string][]int, len(matches))
		for _, match := range matches {
			s.filtered = append(s.filtered, match.Str)
			if len(match.MatchedIndexes) > 0 {
				s.matchIdxes[match.Str] = match.MatchedIndexes
			}
		}
	}

	if excludeFn != nil {
		visible := make([]string, 0, len(s.filtered))
		for _, h := range s.filtered {
			if !excludeFn(inv, h) {
				visible = append(visible, h)
			}
		}
		s.filtered = visible
	}

	s.setListItems(l, inv, hiddenFn)

	// Restore cursor: prefer same host; if gone, pick the next visible
	// neighbour based on allHosts order, then fall back to 0.
	if len(s.filtered) == 0 {
		return
	}
	if prevHost != "" {
		// Fast path: host is still in the list.
		for i, h := range s.filtered {
			if h == prevHost {
				l.Select(i)
				return
			}
		}
		// Walk allHosts forward from the previous host.
		filteredSet := make(map[string]int, len(s.filtered))
		for i, h := range s.filtered {
			filteredSet[h] = i
		}
		past := false
		for _, h := range s.allHosts {
			if h == prevHost {
				past = true
				continue
			}
			if past {
				if idx, ok := filteredSet[h]; ok {
					l.Select(idx)
					return
				}
			}
		}
		// Nothing after — try walking backward.
		for j := len(s.allHosts) - 1; j >= 0; j-- {
			if s.allHosts[j] == prevHost {
				break
			}
			if idx, ok := filteredSet[s.allHosts[j]]; ok {
				l.Select(idx)
				return
			}
		}
	}
	l.Select(0)
}
