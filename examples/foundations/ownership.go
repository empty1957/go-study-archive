// Package foundations contains small, executable examples for the foundations
// learning section.
package foundations

// Checklist owns the items passed to it. Callers cannot mutate its state
// through either the constructor argument or the Items result.
//
// The zero value is a valid empty checklist.
type Checklist struct {
	items []string
}

// NewChecklist copies items because the checklist retains them after the call.
func NewChecklist(items []string) Checklist {
	return Checklist{items: append([]string(nil), items...)}
}

// Items returns a copy so callers cannot mutate the checklist's internal state.
func (c Checklist) Items() []string {
	return append([]string(nil), c.items...)
}
