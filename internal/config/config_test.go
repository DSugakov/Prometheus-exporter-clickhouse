package config

import "testing"

func TestDefaultValidate(t *testing.T) {
	c := Default()
	c.Address = "localhost:9000"
	if err := c.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestProfileInvalid(t *testing.T) {
	c := Default()
	c.Address = "x:9000"
	c.Profile = "nope"
	if err := c.Validate(); err == nil {
		t.Fatal("expected error")
	}
}
