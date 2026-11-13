// Package gailwind generates Tailwind-v3-compatible utility CSS from source
// files, in pure Go: no Node, no external binary. It is a subset engine for
// the v0.1 core (default theme, core utilities and variants); classes it
// cannot generate are reported, never silently dropped.
package gailwind

import (
	"regexp"
	"sort"
	"strings"

	"github.com/ofershap/gailwind/theme"
)

var theme0 = theme.Default()

// Report describes what Generate did with the extracted candidates.
type Report struct {
	Generated int      `json:"generated"`
	Unknown   []string `json:"unknown"` // candidates gailwind cannot compile
}

// extractor approximates Tailwind's own candidate regex: longest runs of
// characters that can appear in a utility class.
var candidateRe = regexp.MustCompile(`-?[A-Za-z0-9][A-Za-z0-9-_./%!@#&()\[\]'":,;\\]*[A-Za-z0-9)\]]|-?[A-Za-z0-9]`)

// Extract returns unique candidate class tokens from source text.
func Extract(sources ...string) []string {
	seen := map[string]bool{}
	var out []string
	for _, src := range sources {
		for _, tok := range candidateRe.FindAllString(src, -1) {
			tok = strings.Trim(tok, `"'`)
			if tok == "" || seen[tok] {
				continue
			}
			seen[tok] = true
			out = append(out, tok)
		}
	}
	sort.Strings(out)
	return out
}

var classAttrRe = regexp.MustCompile("(?i)class(?:Name)?\\s*=\\s*[\"'{]([^\"'`}]+)")

// declaredClasses returns the tokens appearing inside class="..." attributes.
func declaredClasses(sources []string) map[string]bool {
	out := map[string]bool{}
	for _, src := range sources {
		for _, m := range classAttrRe.FindAllStringSubmatch(src, -1) {
			for _, tok := range strings.Fields(m[1]) {
				out[tok] = true
			}
		}
	}
	return out
}

// Generate compiles sources to CSS plus a report of unknown candidates.
func Generate(sources ...string) (string, Report) {
	candidates := Extract(sources...)
	declared := declaredClasses(sources)
	var rules []rule
	var rep Report
	for _, c := range candidates {
		r, ok := compileClass(c)
		if !ok {
			// Report only tokens the author wrote in a class attribute;
			// other extraction noise (SVG path data, prose) is not a class.
			// For sources without class attributes, fall back to a
			// utility-shape heuristic.
			if len(declared) > 0 {
				if declared[c] {
					rep.Unknown = append(rep.Unknown, c)
				}
			} else if strings.ContainsAny(c, "-[:/") {
				rep.Unknown = append(rep.Unknown, c)
			}
			continue
		}
		rules = append(rules, r)
		rep.Generated++
	}
	sortRules(rules)
	return emit(withKeyframes(rules)), rep
}

var keyframesCSS = map[string]string{
	"animate-spin": `@keyframes spin {
to { transform: rotate(360deg) }
}`,
	"animate-ping": `@keyframes ping {
75%, 100% { transform: scale(2); opacity: 0 }
}`,
	"animate-pulse": `@keyframes pulse {
50% { opacity: .5 }
}`,
	"animate-bounce": `@keyframes bounce {
0%, 100% { transform: translateY(-25%); animation-timing-function: cubic-bezier(0.8,0,1,1) }
50% { transform: none; animation-timing-function: cubic-bezier(0,0,0.2,1) }
}`,
}

// withKeyframes inserts @keyframes blocks before the first animate-* rule of
// each used animation, matching where v3 emits them.
func withKeyframes(rules []rule) []rule {
	used := map[string]bool{}
	var out []rule
	for _, r := range rules {
		if kf, ok := keyframesCSS[r.cls]; ok && !used[r.cls] {
			used[r.cls] = true
			out = append(out, rule{raw: kf, cls: "@keyframes:" + r.cls, family: r.family})
		}
		out = append(out, r)
	}
	return out
}

// splitVariants splits a candidate on unbracketed ':' separators.
// "dark:hover:bg-[url(a:b)]" -> ["dark","hover"], "bg-[url(a:b)]".
func splitVariants(cls string) ([]string, string) {
	var variants []string
	depth := 0
	last := 0
	for i := 0; i < len(cls); i++ {
		switch cls[i] {
		case '[':
			depth++
		case ']':
			depth--
		case ':':
			if depth == 0 {
				variants = append(variants, cls[last:i])
				last = i + 1
			}
		}
	}
	return variants, cls[last:]
}

var pseudoMap = map[string]string{
	"first": ":first-child", "last": ":last-child", "only": ":only-child",
	"odd": ":nth-child(odd)", "even": ":nth-child(even)",
	"first-of-type": ":first-of-type", "last-of-type": ":last-of-type", "only-of-type": ":only-of-type",
	"visited": ":visited", "target": ":target", "open": "[open]",
	"checked": ":checked", "indeterminate": ":indeterminate", "placeholder-shown": ":placeholder-shown",
	"autofill": ":autofill", "required": ":required", "valid": ":valid", "invalid": ":invalid",
	"in-range": ":in-range", "out-of-range": ":out-of-range", "read-only": ":read-only",
	"empty": ":empty", "focus-within": ":focus-within", "hover": ":hover", "focus": ":focus",
	"focus-visible": ":focus-visible", "active": ":active", "enabled": ":enabled", "disabled": ":disabled",
}

// compileClass compiles one candidate into a rule.
func compileClass(cls string) (rule, bool) {
	variants, base := splitVariants(cls)
	ctx0 = variantCtx{}
	for _, v := range variants {
		if v == "visited" {
			ctx0.visited = true
		}
	}
	decls, family, ok := compileUtility(base)
	if !ok {
		return rule{}, false
	}
	sel := "." + escapeClass(cls)
	prefix := ""
	var pseudos []string
	for _, v := range variants {
		switch {
		case pseudoMap[v] != "":
			pseudos = append(pseudos, v)
		case v == "group-hover":
			prefix += ".group:hover "
		case v == "peer-hover":
			prefix += ".peer:hover ~ "
		case v == "group-focus":
			prefix += ".group:focus "
		case v == "peer-focus":
			prefix += ".peer:focus ~ "
		case isMediaVariant(v):
		default:
			return rule{}, false // unknown variant: report, don't guess
		}
	}
	// v3 orders pseudo-classes in the selector by variant rank, not class order.
	sort.SliceStable(pseudos, func(i, j int) bool { return variantRank(pseudos[i]) < variantRank(pseudos[j]) })
	suffix := ""
	for _, p := range pseudos {
		suffix += pseudoMap[p]
	}
	return rule{
		selector: prefix + sel + suffix,
		decls:    decls,
		variants: variants,
		family:   family,
		cls:      cls,
		base:     base,
	}, true
}

func nonMediaVariants(vs []string) []string {
	var out []string
	for _, v := range vs {
		if !isMediaVariant(v) {
			out = append(out, v)
		}
	}
	return out
}

func isMediaVariant(v string) bool {
	switch v {
	case "dark", "sm", "md", "lg", "xl", "2xl", "portrait", "landscape", "motion-safe", "motion-reduce", "print":
		return true
	}
	return false
}

// escapeClass escapes a class for use in a CSS selector, matching Tailwind:
// backslash-escape special chars, hex-escape a leading digit.
func escapeClass(cls string) string {
	var b strings.Builder
	for i, r := range cls {
		switch {
		case r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r == '-' || r == '_':
			b.WriteRune(r)
		case r >= '0' && r <= '9':
			if i == 0 {
				b.WriteString("\\3")
				b.WriteRune(r)
				// A hex escape is terminated by a following non-hex char or
				// a consumed space; add the space only before hex digits.
				if i+1 < len(cls) {
					c := cls[i+1]
					if c >= '0' && c <= '9' || c >= 'a' && c <= 'f' || c >= 'A' && c <= 'F' {
						b.WriteString(" ")
					}
				}
			} else {
				b.WriteRune(r)
			}
		default:
			b.WriteString("\\")
			b.WriteRune(r)
		}
	}
	return b.String()
}
