package collector

import (
	"strings"
	"testing"
)

func TestBuildPartsTopQueryWithoutFilters(t *testing.T) {
	q, args := buildPartsTopQuery(nil, nil, 20)
	if len(args) != 1 {
		t.Fatalf("unexpected args len: %d", len(args))
	}
	if got := args[0].(int); got != 20 {
		t.Fatalf("unexpected limit arg: got %d, want 20", got)
	}
	if strings.Contains(q, "has(?, database)") {
		t.Fatal("query must not include allowlist filter")
	}
	if strings.Contains(q, "NOT has(?, database)") {
		t.Fatal("query must not include denylist filter")
	}
}

func TestBuildPartsTopQueryWithFilters(t *testing.T) {
	allow := []string{"analytics", "default"}
	deny := []string{"system"}
	q, args := buildPartsTopQuery(allow, deny, 5)
	if !strings.Contains(q, "AND has(?, database)") {
		t.Fatal("allowlist filter is missing")
	}
	if !strings.Contains(q, "AND NOT has(?, database)") {
		t.Fatal("denylist filter is missing")
	}
	if len(args) != 3 {
		t.Fatalf("unexpected args len: %d", len(args))
	}
	if got := args[2].(int); got != 5 {
		t.Fatalf("unexpected limit arg: got %d, want 5", got)
	}
}
