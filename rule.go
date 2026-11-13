package gailwind

import (
	"fmt"
	"sort"
	"strings"
)

// rule is one compiled CSS rule.
type rule struct {
	selector string   // e.g. `.hover\:m-2:hover` or `@media`-wrapped
	decls    []decl   // declarations in order
	variants []string // variant names, outermost-first (e.g. ["md","hover"])
	family   int      // utility family order index (v3 core-plugin order)
	cls      string   // the original candidate class
	base     string   // variantless utility, for ordering
	raw      string   // pre-rendered block (keyframes); bypasses selector/decls
}

type decl struct {
	prop  string
	value string
}

func (r rule) key() string { return r.cls }

// variantRank orders variant layers to match v3's observed output:
// base < structural/state pseudos < group/peer < screens < dark.
func variantRank(v string) int {
	switch v {
	case "first":
		return 1
	case "last":
		return 2
	case "only":
		return 3
	case "odd":
		return 4
	case "even":
		return 5
	case "first-of-type":
		return 6
	case "last-of-type":
		return 7
	case "only-of-type":
		return 8
	case "visited":
		return 9
	case "target":
		return 10
	case "open":
		return 11
	case "default":
		return 12
	case "checked":
		return 13
	case "indeterminate":
		return 14
	case "placeholder-shown":
		return 15
	case "autofill":
		return 16
	case "required":
		return 17
	case "valid":
		return 18
	case "invalid":
		return 19
	case "in-range":
		return 20
	case "out-of-range":
		return 21
	case "read-only":
		return 22
	case "empty":
		return 23
	case "focus-within":
		return 24
	case "hover":
		return 25
	case "focus":
		return 26
	case "focus-visible":
		return 27
	case "active":
		return 28
	case "enabled":
		return 29
	case "disabled":
		return 30
	}
	if strings.HasPrefix(v, "group-hover") {
		return 35
	}
	if strings.HasPrefix(v, "group-focus") {
		return 36
	}
	if strings.HasPrefix(v, "peer-hover") {
		return 37
	}
	if strings.HasPrefix(v, "peer-focus") {
		return 38
	}
	switch v {
	case "motion-safe":
		return 39
	case "motion-reduce":
		return 40
	case "sm":
		return 41
	case "md":
		return 42
	case "lg":
		return 43
	case "xl":
		return 44
	case "2xl":
		return 45
	case "portrait":
		return 46
	case "landscape":
		return 47
	case "dark":
		return 50
	case "print":
		return 60
	}
	return 99
}

// variantKey returns variant ranks sorted descending; v3's variant offset
// comparison is equivalent to lexicographic comparison of that tuple, with
// shorter tuples (fewer variants) first on a common prefix.
func variantKey(variants []string) []int {
	out := make([]int, 0, len(variants))
	for _, v := range variants {
		out = append(out, variantRank(v))
	}
	sort.Sort(sort.Reverse(sort.IntSlice(out)))
	return out
}

func compareIntSlices(a, b []int) int {
	for i := 0; i < len(a) && i < len(b); i++ {
		if a[i] != b[i] {
			if a[i] < b[i] {
				return -1
			}
			return 1
		}
	}
	switch {
	case len(a) < len(b):
		return -1
	case len(a) > len(b):
		return 1
	}
	return 0
}

// sortRules orders rules to match v3: by outermost variant rank (last variant
// listed in the class is the outermost wrapper), then family order, then class.
func sortRules(rules []rule) {
	sort.SliceStable(rules, func(i, j int) bool {
		if c := compareIntSlices(variantKey(rules[i].variants), variantKey(rules[j].variants)); c != 0 {
			return c < 0
		}
		if rules[i].family != rules[j].family {
			return rules[i].family < rules[j].family
		}
		gi, gj := sortGroup(rules[i].family, rules[i].base), sortGroup(rules[j].family, rules[j].base)
		if gi != gj {
			return gi < gj
		}
		si, iok := classSeq[rules[i].base]
		sj, jok := classSeq[rules[j].base]
		if iok && jok {
			return si < sj
		}
		return rules[i].cls < rules[j].cls
	})
}

// emit renders rules as CSS, grouping consecutive rules under shared @media wrappers.
func emit(rules []rule) string {
	var b strings.Builder
	var openWrappers []string
	closeWrappers := func(n int) {
		for i := 0; i < n; i++ {
			b.WriteString("}\n")
		}
		openWrappers = nil
	}
	for _, r := range rules {
		wrappers := mediaWrappers(r.variants)
		if !equalStrings(wrappers, openWrappers) {
			closeWrappers(len(openWrappers))
			for _, w := range wrappers {
				b.WriteString(w + " {\n")
			}
			openWrappers = wrappers
		}
		if r.raw != "" {
			b.WriteString(r.raw + "\n")
			continue
		}
		b.WriteString(r.selector + " { ")
		for _, d := range r.decls {
			b.WriteString(d.prop + ": " + d.value + "; ")
		}
		b.WriteString("}\n")
	}
	closeWrappers(len(openWrappers))
	return b.String()
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// mediaWrappers returns the @media text for a rule's variants, outermost first.
func mediaWrappers(variants []string) []string {
	t := theme0
	var out []string
	for _, v := range variants {
		switch v {
		case "dark":
			out = append(out, "@media (prefers-color-scheme: dark)")
		case "portrait":
			out = append(out, "@media (orientation: portrait)")
		case "landscape":
			out = append(out, "@media (orientation: landscape)")
		case "motion-safe":
			out = append(out, "@media (prefers-reduced-motion: no-preference)")
		case "motion-reduce":
			out = append(out, "@media (prefers-reduced-motion: reduce)")
		case "print":
			out = append(out, "@media print")
		default:
			if w, ok := t.Screens[v]; ok {
				out = append(out, fmt.Sprintf("@media (min-width: %s)", w))
			}
		}
	}
	return out
}
