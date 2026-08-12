package foundations

import "testing"

func TestChecklistOwnsItems(t *testing.T) {
	input := []string{"read", "run"}
	checklist := NewChecklist(input)

	input[0] = "changed through input"
	got := checklist.Items()
	if got[0] != "read" {
		t.Fatalf("Items()[0] = %q after input mutation, want %q", got[0], "read")
	}

	got[1] = "changed through output"
	if checklist.Items()[1] != "run" {
		t.Fatalf("Items()[1] changed through returned slice, want %q", "run")
	}
}

func TestChecklistZeroValue(t *testing.T) {
	var checklist Checklist
	if got := checklist.Items(); got != nil {
		t.Fatalf("zero-value Items() = %#v, want nil", got)
	}
}
