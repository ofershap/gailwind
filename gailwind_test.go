package gailwind

import (
	"strings"
	"testing"
)

func compileOne(t *testing.T, cls string) string {
	t.Helper()
	css, rep := Generate(`<div class="` + cls + `"></div>`)
	if len(rep.Unknown) != 0 {
		t.Fatalf("%s reported unknown: %v", cls, rep.Unknown)
	}
	if !strings.Contains(css, escapeClass(cls)) {
		t.Fatalf("%s: selector missing in %q", cls, css)
	}
	return css
}

func TestCoreUtilities(t *testing.T) {
	cases := map[string]string{
		"m-4":             "margin: 1rem",
		"mx-auto":         "margin-left: auto",
		"-mt-2":           "margin-top: -0.5rem",
		"p-2":             "padding: 0.5rem",
		"w-1/2":           "width: 50%",
		"w-[17px]":        "width: 17px",
		"h-screen":        "height: 100vh",
		"flex":            "display: flex",
		"hidden":          "display: none",
		"items-center":    "align-items: center",
		"justify-between": "justify-content: space-between",
		"gap-4":           "gap: 1rem",
		"text-sm":         "font-size: 0.875rem",
		"text-red-500":    "color: rgb(239 68 68 / var(--tw-text-opacity, 1))",
		"bg-white":        "background-color: rgb(255 255 255 / var(--tw-bg-opacity, 1))",
		"bg-blue-500/50":  "background-color: rgb(59 130 246 / 0.5)",
		"font-bold":       "font-weight: 700",
		"rounded-lg":      "border-radius: 0.5rem",
		"rounded":         "border-radius: 0.25rem",
		"shadow-md":       "box-shadow",
		"opacity-50":      "opacity: 0.5",
		"z-10":            "z-index: 10",
		"grid-cols-3":     "repeat(3, minmax(0, 1fr))",
		"underline":       "text-decoration-line: underline",
		"cursor-pointer":  "cursor: pointer",
		"sr-only":         "clip: rect(0, 0, 0, 0)",
	}
	for cls, want := range cases {
		css := compileOne(t, cls)
		if !strings.Contains(css, want) {
			t.Errorf("%s: want %q in %q", cls, want, css)
		}
	}
}

func TestVariants(t *testing.T) {
	css := compileOne(t, "hover:bg-blue-600")
	if !strings.Contains(css, `.hover\:bg-blue-600:hover`) {
		t.Errorf("hover selector wrong: %q", css)
	}
	css = compileOne(t, "md:w-1/2")
	if !strings.Contains(css, "@media (min-width: 768px)") {
		t.Errorf("md media missing: %q", css)
	}
	css = compileOne(t, "dark:bg-gray-900")
	if !strings.Contains(css, "@media (prefers-color-scheme: dark)") {
		t.Errorf("dark media missing: %q", css)
	}
}

func TestUnknownReportedNotDropped(t *testing.T) {
	_, rep := Generate(`<div class="totally-made-up-thing animate-weirdness"></div>`)
	if len(rep.Unknown) == 0 {
		t.Fatal("unknown classes must be reported")
	}
}

func TestOrderingBaseBeforeHoverBeforeScreensBeforeDark(t *testing.T) {
	css, _ := Generate(`<div class="dark:m-9 xl:m-1 hover:m-2 m-3 first:m-7"></div>`)
	idx := map[string]int{}
	for _, sel := range []string{".m-3", ".first\\:m-7", ".hover\\:m-2", ".xl\\:m-1", ".dark\\:m-9"} {
		idx[sel] = strings.Index(css, sel)
		if idx[sel] < 0 {
			t.Fatalf("missing %s in %q", sel, css)
		}
	}
	if !(idx[".m-3"] < idx[".first\\:m-7"] && idx[".first\\:m-7"] < idx[".hover\\:m-2"] && idx[".hover\\:m-2"] < idx[".xl\\:m-1"] && idx[".xl\\:m-1"] < idx[".dark\\:m-9"]) {
		t.Fatalf("order wrong: %v\n%s", idx, css)
	}
}
