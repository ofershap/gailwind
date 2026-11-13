// Package preflight embeds Tailwind CSS v3.4.19's base reset ("preflight"),
// generated verbatim from the official CLI (`@tailwind base`) and covered by
// Tailwind Labs' MIT license (see licenses/LICENSE-tailwindcss).
package preflight

import _ "embed"

//go:embed preflight.css
var css string

// CSS is the verbatim v3.4.19 preflight stylesheet.
func CSS() string { return css }
