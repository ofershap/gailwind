package preflight

import (
	"strings"
	"testing"
)

// Pin the reset behaviors whose absence was observed as visible rendering
// differences against the real Tailwind build.
func TestPreflightPins(t *testing.T) {
	for _, want := range []string{
		"margin: 0",        // body and headings reset
		"list-style: none", // ul/ol markers off
		"text-decoration: inherit",
		"border-style: solid", // borders default
		"box-sizing: border-box",
	} {
		if !strings.Contains(CSS(), want) {
			t.Errorf("preflight missing %q", want)
		}
	}
}
