package collector

import "testing"

func TestNameFilterAllowed(t *testing.T) {
	f := newNameFilter([]string{"A", "B"}, []string{"B"})
	if !f.Allowed("A") {
		t.Fatal("A must be allowed")
	}
	if f.Allowed("B") {
		t.Fatal("B must be denied")
	}
	if f.Allowed("C") {
		t.Fatal("C must be blocked by allowlist")
	}
}

func TestNameFilterOnlyDeny(t *testing.T) {
	f := newNameFilter(nil, []string{"X"})
	if f.Allowed("X") {
		t.Fatal("X must be denied")
	}
	if !f.Allowed("Y") {
		t.Fatal("Y must be allowed")
	}
}
