package collector

import "testing"

func TestIsUnsupportedSchemaError(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{"unknown table", errString("Code: 60. DB::Exception: Unknown table system.replicas"), true},
		{"unknown column", errString("There is no column absolute_delay"), true},
		{"unknown identifier", errString("Unknown identifier: is_done"), true},
		{"other", errString("authentication failed"), false},
		{"nil", nil, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := isUnsupportedSchemaError(tc.err)
			if got != tc.want {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
		})
	}
}

type errString string

func (e errString) Error() string { return string(e) }
