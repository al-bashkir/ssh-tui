package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// ---------------------------------------------------------------------------
// Navigation item
// ---------------------------------------------------------------------------

// formNavItem is a single navigable item (always a field) in the form.
type formNavItem struct {
	sectionIdx int
	fieldIdx   int
}

// ---------------------------------------------------------------------------
// Text input entry
// ---------------------------------------------------------------------------

// formTextInput pairs a field key with its textinput.Model.
type formTextInput struct {
	key   string
	input textinput.Model
}

// ---------------------------------------------------------------------------
// formModel
// ---------------------------------------------------------------------------

// formModel manages the generic state of a form with vim-style navigation
// and declarative field definitions.
//
// It is not a tea.Model itself — specific form models (defaults, group, host)
// embed it and delegate to its methods.
type formModel struct {
	schema    formSchema
	values    map[string]string // current values by field key
	originals map[string]string // snapshot at creation (for dirty check)

	items    []formNavItem // navigable field items
	focusIdx int           // index into items
	editing  bool          // true when in insert mode

	inputs []formTextInput // text inputs for text/number fields

	width  int
	height int

	labelWidth  int // label column width (chars)
	fieldIndent int // indent for fields under section headers
}

// newFormModel creates a formModel from a schema and initial values.
func newFormModel(schema formSchema, values map[string]string, labelWidth int) formModel {
	originals := make(map[string]string, len(values))
	for k, v := range values {
		originals[k] = v
	}

	m := formModel{
		schema:      schema,
		values:      values,
		originals:   originals,
		labelWidth:  labelWidth,
		fieldIndent: 2,
	}

	// Create text inputs for text/number fields.
	for si := range schema.Sections {
		for fi := range schema.Sections[si].Fields {
			fd := &schema.Sections[si].Fields[fi]
			if !fd.isTextField() {
				continue
			}
			in := textinput.New()
			limit := fd.CharLimit
			if limit <= 0 {
				limit = 256
			}
			in.CharLimit = limit
			in.Prompt = ""
			in.SetValue(values[fd.Key])
			in.Placeholder = fd.Placeholder
			configureSearch(&in)
			m.inputs = append(m.inputs, formTextInput{key: fd.Key, input: in})
		}
	}

	m.rebuildItems()

	// Focus the first field.
	if len(m.items) > 0 {
		m.focusIdx = 0
		m.applyFocusStyles()
	}

	return m
}

// ---------------------------------------------------------------------------
// Value access
// ---------------------------------------------------------------------------

// input returns a pointer to the textinput.Model for the given field key.
func (m *formModel) input(key string) *textinput.Model {
	for i := range m.inputs {
		if m.inputs[i].key == key {
			return &m.inputs[i].input
		}
	}
	return nil
}

// value returns the current form value for a field key.
func (m *formModel) value(key string) string {
	return m.values[key]
}

// setValue updates the current form value for a field key.
func (m *formModel) setValue(key, val string) {
	m.values[key] = val
}

// isDirty returns true if any value differs from the original snapshot.
func (m *formModel) isDirty() bool {
	for key, val := range m.values {
		if m.originals[key] != val {
			return true
		}
	}
	return false
}

// resetOriginals snapshots the current values as the new baseline,
// clearing the dirty state.
func (m *formModel) resetOriginals() {
	for k, v := range m.values {
		m.originals[k] = v
	}
}

// ---------------------------------------------------------------------------
// Navigation items
// ---------------------------------------------------------------------------

// rebuildItems constructs the flat navigation list of fields.
func (m *formModel) rebuildItems() {
	m.items = m.items[:0]
	for si, sec := range m.schema.Sections {
		for fi := range sec.Fields {
			m.items = append(m.items, formNavItem{sectionIdx: si, fieldIdx: fi})
		}
	}
}

// ---------------------------------------------------------------------------
// Focus queries
// ---------------------------------------------------------------------------

// focusedItem returns the currently focused navigation item.
func (m *formModel) focusedItem() formNavItem {
	if m.focusIdx >= 0 && m.focusIdx < len(m.items) {
		return m.items[m.focusIdx]
	}
	return formNavItem{}
}

// focusedField returns the definition of the focused field, or nil.
func (m *formModel) focusedField() *fieldDef {
	item := m.focusedItem()
	if item.sectionIdx < len(m.schema.Sections) {
		sec := &m.schema.Sections[item.sectionIdx]
		if item.fieldIdx < len(sec.Fields) {
			return &sec.Fields[item.fieldIdx]
		}
	}
	return nil
}

// focusedFieldKey returns the key of the focused field, or "".
func (m *formModel) focusedFieldKey() string {
	if fd := m.focusedField(); fd != nil {
		return fd.Key
	}
	return ""
}

// ---------------------------------------------------------------------------
// Focus management
// ---------------------------------------------------------------------------

// applyFocusStyles updates text input accent styling to match current focus.
func (m *formModel) applyFocusStyles() {
	focusedKey := m.focusedFieldKey()
	for i := range m.inputs {
		setSearchFocused(&m.inputs[i].input, m.inputs[i].key == focusedKey)
	}
}

// refreshAccentStyles re-applies accent styles after a theme/accent change.
func (m *formModel) refreshAccentStyles() {
	m.applyFocusStyles()
}

// setFocusIdx moves focus to the given item index.
func (m *formModel) setFocusIdx(idx int) {
	if idx < 0 || idx >= len(m.items) {
		return
	}
	m.exitEdit()
	m.focusIdx = idx
	m.applyFocusStyles()
}

// blurAllInputs deactivates all text inputs.
func (m *formModel) blurAllInputs() {
	for i := range m.inputs {
		m.inputs[i].input.Blur()
	}
}

// moveFocus moves focus by delta items (+1 down, -1 up), wrapping at edges.
func (m *formModel) moveFocus(delta int) {
	n := len(m.items)
	if n == 0 {
		return
	}
	pos := m.focusIdx + delta
	if pos < 0 {
		pos = n - 1
	}
	if pos >= n {
		pos = 0
	}
	m.setFocusIdx(pos)
}

// ---------------------------------------------------------------------------
// Edit mode (insert)
// ---------------------------------------------------------------------------

// enterEdit activates insert mode for the focused text/number field.
func (m *formModel) enterEdit() {
	fd := m.focusedField()
	if fd == nil || !fd.isTextField() {
		return
	}
	m.editing = true
	if in := m.input(fd.Key); in != nil {
		_ = in.Focus()
	}
}

// exitEdit deactivates insert mode and syncs the text input value.
func (m *formModel) exitEdit() {
	if !m.editing {
		return
	}
	m.editing = false
	if fd := m.focusedField(); fd != nil && fd.isTextField() {
		if in := m.input(fd.Key); in != nil {
			m.values[fd.Key] = in.Value()
			in.Blur()
		}
	}
}

// updateFocusedInput routes a key message to the focused text input and
// syncs the value back.
func (m *formModel) updateFocusedInput(msg tea.Msg) tea.Cmd {
	fd := m.focusedField()
	if fd == nil || !fd.isTextField() {
		return nil
	}
	in := m.input(fd.Key)
	if in == nil {
		return nil
	}
	newIn, cmd := in.Update(msg)
	*in = newIn
	m.values[fd.Key] = in.Value()
	return cmd
}

// ---------------------------------------------------------------------------
// Value cycling (pickers / toggles)
// ---------------------------------------------------------------------------

// cycleValue cycles the focused picker/toggle value by delta (+1/-1).
func (m *formModel) cycleValue(delta int) {
	fd := m.focusedField()
	if fd == nil || len(fd.Options) == 0 {
		return
	}
	cur := strings.TrimSpace(m.values[fd.Key])
	idx := 0
	for i, opt := range fd.Options {
		if strings.TrimSpace(opt.Value) == cur {
			idx = i
			break
		}
	}
	idx += delta
	if idx < 0 {
		idx = len(fd.Options) - 1
	}
	if idx >= len(fd.Options) {
		idx = 0
	}
	m.values[fd.Key] = fd.Options[idx].Value
}

// ---------------------------------------------------------------------------
// Key handling
// ---------------------------------------------------------------------------

// handleKey processes a key message in the generic form.
// Returns true if the key was consumed by the form.
func (m *formModel) handleKey(msg tea.KeyMsg) (bool, tea.Cmd) {
	// Insert mode: route everything to the text input.
	if m.editing {
		switch msg.String() {
		case "esc":
			m.exitEdit()
			return true, nil
		case "enter":
			m.exitEdit()
			m.moveFocus(1)
			return true, nil
		default:
			return true, m.updateFocusedInput(msg)
		}
	}

	// Normal mode.
	s := msg.String()
	switch s {
	case "j", "down", "tab":
		m.moveFocus(1)
		return true, nil
	case "k", "up", "shift+tab":
		m.moveFocus(-1)
		return true, nil
	case "i":
		if fd := m.focusedField(); fd != nil && fd.isTextField() {
			m.enterEdit()
		}
		return true, nil
	case "enter":
		if fd := m.focusedField(); fd != nil {
			if fd.isTextField() {
				m.enterEdit()
				return true, nil
			}
			if fd.Kind == fieldSubModal {
				return false, nil // wrapper handles sub-modal activation
			}
		}
		m.moveFocus(1)
		return true, nil
	case " ":
		if fd := m.focusedField(); fd != nil {
			switch fd.Kind {
			case fieldPicker, fieldToggle:
				m.cycleValue(1)
				return true, nil
			case fieldSubModal:
				return false, nil
			}
		}
		return true, nil
	case "h", "left":
		if fd := m.focusedField(); fd != nil {
			if fd.Kind == fieldPicker || fd.Kind == fieldToggle {
				m.cycleValue(-1)
				return true, nil
			}
		}
		return true, nil
	case "l", "right":
		if fd := m.focusedField(); fd != nil {
			switch fd.Kind {
			case fieldPicker, fieldToggle:
				m.cycleValue(1)
				return true, nil
			case fieldSubModal:
				return false, nil
			}
		}
		return true, nil
	}

	return false, nil
}

// ---------------------------------------------------------------------------
// Resize
// ---------------------------------------------------------------------------

// handleResize updates form dimensions and text input widths.
func (m *formModel) handleResize(w, h int) {
	m.width = w
	m.height = h
	innerW := max(0, w-2)
	fieldW := max(10, innerW-m.fieldIndent-m.labelWidth-1)
	for i := range m.inputs {
		fd := m.schema.fieldByKey(m.inputs[i].key)
		if fd != nil && fd.Narrow {
			m.inputs[i].input.Width = min(12, fieldW)
		} else {
			m.inputs[i].input.Width = fieldW
		}
	}
}

// ---------------------------------------------------------------------------
// Footer
// ---------------------------------------------------------------------------

// positionInfo returns a human-readable position string for the footer.
func (m *formModel) positionInfo() string {
	if m.focusIdx < 0 || m.focusIdx >= len(m.items) {
		return ""
	}
	item := m.items[m.focusIdx]
	sec := &m.schema.Sections[item.sectionIdx]
	return fmt.Sprintf("%d/%d %s", item.fieldIdx+1, len(sec.Fields), sec.Label)
}

// footerHints returns context-sensitive key hints for the footer.
func (m *formModel) footerHints() string {
	if m.editing {
		return "Ctrl+S save   Esc done"
	}
	fd := m.focusedField()
	if fd == nil {
		return "Ctrl+S save   j/k nav   Esc back"
	}
	switch fd.Kind {
	case fieldText, fieldNumber:
		return "Ctrl+S save   j/k nav   i edit   Esc back"
	case fieldPicker, fieldToggle:
		return "Ctrl+S save   j/k nav   h/l option   Esc back"
	case fieldSubModal:
		return "Ctrl+S save   j/k nav   Enter select   Esc back"
	}
	return "Ctrl+S save   j/k nav   Esc back"
}

// renderFooter returns the styled footer line.
func (m *formModel) renderFooter() string {
	pos := m.positionInfo()
	if m.editing {
		return footerStyle.Render(pos) + "  " + headerStyle.Render("INSERT") + "  " + footerStyle.Render("Ctrl+S save   Esc done")
	}
	return footerStyle.Render(pos + "  " + m.footerHints())
}

// ---------------------------------------------------------------------------
// Validation
// ---------------------------------------------------------------------------

// validate runs all per-field validators and returns the first error.
func (m *formModel) validate() error {
	for _, sec := range m.schema.Sections {
		for _, fd := range sec.Fields {
			if fd.Validate == nil {
				continue
			}
			if err := fd.Validate(m.values[fd.Key]); err != nil {
				return fmt.Errorf("%s: %w", fd.Label, err)
			}
		}
	}
	return nil
}
