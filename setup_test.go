package trapi

import (
	"testing"

	"github.com/mholt/caddy"
)

// TestSetup tests the various things that should be parsed by setup.
// Make sure you also test for parse errors.
func TestSetup(t *testing.T) {
	c := caddy.NewTestController("dns", `trapi 127.0.0.1:53080`) // IPv4
	if err := setupTrapi(c); err != nil {
		t.Fatalf("Expected no errors, but got: %v", err)
	}

	c = caddy.NewTestController("dns", `trapi 127.1:53080`) // shorthand IPv4
	if err := setupTrapi(c); err != nil {
		t.Fatalf("Expected no errors, but got: %v", err)
	}

	c = caddy.NewTestController("dns", `trapi [::1]:53080`) // IPv6
	if err := setupTrapi(c); err != nil {
		t.Fatalf("Expected no errors, but got: %v", err)
	}

	// TODO: handle invalid listen address in goroutine
	//c = caddy.NewTestController("dns", `trapi xyz`) // malformed listen address
	//if err := setupTrapi(c); err == nil {
	//	t.Fatalf("Expected errors, but got: %v", err)
	//}

	c = caddy.NewTestController("dns", `trapi`) // no arguments
	if err := setupTrapi(c); err == nil {
		t.Fatalf("Expected errors, but got: %v", err)
	}
}

// TODO: TestAPI
