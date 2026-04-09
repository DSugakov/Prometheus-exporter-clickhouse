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


func TestTimeoutInvalid(t *testing.T) {
	c := Default()
	c.Address = "x:9000"
	c.QueryTimeout = 0
	if err := c.Validate(); err == nil {
		t.Fatal("expected error for zero query timeout")
	}
}

func TestPartsTopNHardLimit(t *testing.T) {
	c := Default()
	c.Address = "x:9000"
	c.PartsTopN = AggressiveHardMaxPartsTopN + 1
	if err := c.Validate(); err == nil {
		t.Fatal("expected error for parts_top_n hard limit")
	}
}
