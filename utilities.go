package gailwind

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/ofershap/gailwind/theme"
)

// matcher compiles a base (variantless) utility to declarations.
type matcher struct {
	family string
	fn     func(base string) ([]decl, bool)
}

var (
	matchers    []matcher
	classSeq    = map[string]int{} // definition order for static utilities
	familyIndex = map[string]int{}
	// familyGroups gives prefix groups for scale families: classes in an
	// earlier group sort first; within a group, lexicographic (v3 behavior).
	familyGroups = map[string][][]string{}
)

func register(family string, groups [][]string, fn func(base string) ([]decl, bool)) {
	familyIndex[family] = len(matchers)
	if groups != nil {
		familyGroups[family] = groups
	}
	matchers = append(matchers, matcher{family: family, fn: fn})
}

// statics registers a fixed class->declarations table; the literal pair
// order defines output order within the family (matching v3 plugin source).
func statics(pairs ...string) func(string) ([]decl, bool) {
	return staticsOrdered(true, pairs...)
}

// staticsLex is statics without definition-order pinning: classes sort
// lexicographically within the family (v3's theme-driven utility order).
func staticsLex(pairs ...string) func(string) ([]decl, bool) {
	return staticsOrdered(false, pairs...)
}

func staticsOrdered(pin bool, pairs ...string) func(string) ([]decl, bool) {
	table := map[string]string{}
	for i := 0; i+1 < len(pairs); i += 2 {
		table[pairs[i]] = pairs[i+1]
		if pin {
			if _, ok := classSeq[pairs[i]]; !ok {
				classSeq[pairs[i]] = len(classSeq)
			}
		}
	}
	return func(base string) ([]decl, bool) {
		v, ok := table[base]
		if !ok {
			return nil, false
		}
		var out []decl
		for _, pair := range strings.Split(v, ";") {
			kv := strings.SplitN(strings.TrimSpace(pair), ":", 2)
			if len(kv) == 2 {
				out = append(out, decl{strings.TrimSpace(kv[0]), strings.TrimSpace(kv[1])})
			}
		}
		return out, true
	}
}

// compileUtility tries each matcher in family registration order.
func compileUtility(base string) ([]decl, int, bool) {
	for _, m := range matchers {
		if d, ok := m.fn(base); ok {
			return d, familyIndex[m.family], true
		}
	}
	return nil, 0, false
}

// sortGroup returns the prefix-group index of base within its family.
func sortGroup(familyIdx int, base string) int {
	if familyIdx < 0 || familyIdx >= len(matchers) {
		return 0
	}
	groups := familyGroups[matchers[familyIdx].family]
	b := strings.TrimPrefix(base, "-")
	for gi, g := range groups {
		for _, p := range g {
			if b == p || strings.HasPrefix(b, p+"-") || strings.HasPrefix(b, p+"[") {
				return gi
			}
		}
	}
	return 0
}

// ---- helpers ----

func d(prop, value string) []decl { return []decl{{prop, value}} }

func twSpacing(v string) (string, bool) {
	if v == "px" {
		return "1px", true
	}
	if s, ok := theme0.Spacing[v]; ok {
		return s, true
	}
	return "", false
}

// arbitrary parses "prefix-[value]" (underscores become spaces, per v3).
func arbitrary(base, prefix string) (string, bool) {
	if strings.HasPrefix(base, prefix+"-[") && strings.HasSuffix(base, "]") {
		return strings.ReplaceAll(base[len(prefix)+2:len(base)-1], "_", " "), true
	}
	return "", false
}

// variantCtx carries variant-dependent compile behavior into utilities.
type variantCtx struct{ visited bool }

var ctx0 variantCtx

func hexToRGB(hex string) (string, bool) {
	h := strings.TrimPrefix(hex, "#")
	if len(h) == 3 {
		h = string([]byte{h[0], h[0], h[1], h[1], h[2], h[2]})
	}
	if len(h) != 6 {
		return "", false
	}
	var r, g, b int
	if _, err := fmt.Sscanf(h, "%02x%02x%02x", &r, &g, &b); err != nil {
		return "", false
	}
	return fmt.Sprintf("%d %d %d", r, g, b), true
}

// colorDecls emits v3.4's rgb(R G B / var(--tw-*-opacity, 1)) form.
func colorDecls(base, prefix, prop, opacityVar string) ([]decl, bool) {
	name, ok := strings.CutPrefix(base, prefix)
	if !ok {
		return nil, false
	}
	return colorValue(name, prop, opacityVar)
}

func colorValue(name, prop, opacityVar string) ([]decl, bool) {
	alpha := ""
	if i := strings.Index(name, "/"); i >= 0 {
		alpha = name[i+1:]
		name = name[:i]
	}
	hex, ok := theme0.FlatColors()[name]
	if !ok {
		return nil, false
	}
	rgb, isHex := hexToRGB(hex)
	if !isHex {
		return d(prop, hex), true // transparent, currentColor, inherit
	}
	if alpha != "" {
		a, ok := theme0.Opacity[alpha]
		if !ok {
			return nil, false
		}
		return d(prop, fmt.Sprintf("rgb(%s / %s)", rgb, a)), true
	}
	if ctx0.visited {
		// v3 quirk: visited colors drop the opacity variable.
		return d(prop, fmt.Sprintf("rgb(%s )", rgb)), true
	}
	return []decl{
		{opacityVar, "1"},
		{prop, fmt.Sprintf("rgb(%s / var(%s, 1))", rgb, opacityVar)},
	}, true
}

// pctFraction formats v3's 6-decimal fraction percentages.
func pctFraction(n string) (string, bool) {
	parts := strings.SplitN(n, "/", 2)
	if len(parts) != 2 {
		return "", false
	}
	var num, den float64
	if _, err := fmt.Sscanf(parts[0], "%g", &num); err != nil {
		return "", false
	}
	if _, err := fmt.Sscanf(parts[1], "%g", &den); err != nil || den == 0 {
		return "", false
	}
	s := fmt.Sprintf("%.6f", num/den*100)
	s = strings.TrimRight(strings.TrimRight(s, "0"), ".")
	return s + "%", true
}

func negateValue(v string) string {
	if strings.HasPrefix(v, "-") {
		return v[1:]
	}
	return "-" + v
}

var rgbFuncRe = regexp.MustCompile(`rgb\([^)]*\)`)

func looksLikeSize(v string) bool {
	if strings.HasPrefix(v, "#") || strings.HasPrefix(v, "rgb") || strings.HasPrefix(v, "hsl") {
		return false
	}
	for _, u := range []string{"px", "rem", "em", "%", "vw", "vh", "pt", "ch", "ex", "lh", "vmin", "vmax"} {
		if strings.HasSuffix(v, u) {
			return true
		}
	}
	return false
}

// arbitraryColor compiles an arbitrary color value to v3's rgb form.
func arbitraryColor(v, prop, opacityVar string) ([]decl, bool) {
	if rgb, ok := hexToRGB(v); ok {
		if ctx0.visited {
			return d(prop, "rgb("+rgb+" )"), true
		}
		return []decl{{opacityVar, "1"}, {prop, "rgb(" + rgb + " / var(" + opacityVar + ", 1))"}}, true
	}
	return d(prop, v), true
}

func scaleLookup(base, prefix string, scale map[string]string, prop string) ([]decl, bool) {
	name, ok := strings.CutPrefix(base, prefix)
	if !ok {
		return nil, false
	}
	v, ok := scale[name]
	if !ok {
		return nil, false
	}
	return d(prop, v), true
}

var directionMap = map[string][]string{
	"":  {""},
	"x": {"left", "right"},
	"y": {"top", "bottom"},
	"t": {"top"},
	"b": {"bottom"},
	"l": {"left"},
	"r": {"right"},
	"s": {"inline-start"},
	"e": {"inline-end"},
}

func spacingBox(base, prefix, prop string) ([]decl, bool) {
	sign, b, _ := neg(base)
	name, ok := strings.CutPrefix(b, prefix)
	if !ok {
		return nil, false
	}
	dir := ""
	rest := name
	if i := strings.Index(name, "-"); i >= 0 {
		dir = name[:i]
		rest = name[i+1:]
	}
	var val string
	if strings.HasPrefix(rest, "[") && strings.HasSuffix(rest, "]") {
		val = strings.ReplaceAll(rest[1:len(rest)-1], "_", " ")
	} else {
		var vok bool
		val, vok = twSpacing(rest)
		if !vok {
			return nil, false
		}
	}
	sides, ok := directionMap[dir]
	if !ok {
		return nil, false
	}
	var out []decl
	for _, s := range sides {
		if s == "" {
			out = append(out, decl{prop, sign + val})
		} else {
			out = append(out, decl{prop + "-" + s, sign + val})
		}
	}
	return out, true
}

func neg(base string) (string, string, bool) {
	if strings.HasPrefix(base, "-") {
		return "-", base[1:], true
	}
	return "", base, false
}

func negatable(base, prefix string, scale map[string]string, prop string) ([]decl, bool) {
	_, b, isNeg := neg(base)
	out, ok := scaleLookup(b, prefix, scale, prop)
	if ok && isNeg {
		for i := range out {
			out[i].value = negateValue(out[i].value)
		}
	}
	return out, ok
}

var twTransform = "translate(var(--tw-translate-x), var(--tw-translate-y)) rotate(var(--tw-rotate)) skewX(var(--tw-skew-x)) skewY(var(--tw-skew-y)) scaleX(var(--tw-scale-x)) scaleY(var(--tw-scale-y))"

// ---- registry (family order follows tailwindcss v3.4 core plugin order) ----

func init() {
	t := theme.Default()

	register("accessibility", nil, statics(
		"sr-only", "position: absolute; width: 1px; height: 1px; padding: 0; margin: -1px; overflow: hidden; clip: rect(0, 0, 0, 0); white-space: nowrap; border-width: 0",
	))
	register("pointerEvents", nil, statics(
		"pointer-events-none", "pointer-events: none",
		"pointer-events-auto", "pointer-events: auto",
	))
	register("visibility", nil, statics(
		"visible", "visibility: visible",
		"invisible", "visibility: hidden",
	))
	register("position", nil, statics(
		"static", "position: static",
		"fixed", "position: fixed",
		"absolute", "position: absolute",
		"relative", "position: relative",
		"sticky", "position: sticky",
	))
	register("inset", [][]string{{"inset"}, {"inset-x", "inset-y"}, {"top", "right", "bottom", "left"}}, func(base string) ([]decl, bool) {
		sign, b, _ := neg(base)
		val := func(n string) (string, bool) {
			if strings.HasPrefix(n, "[") && strings.HasSuffix(n, "]") {
				return strings.ReplaceAll(n[1:len(n)-1], "_", " "), true
			}
			return twSpacing(n)
		}
		for _, p := range []string{"inset-x", "inset-y", "inset"} {
			if n, ok := strings.CutPrefix(b, p+"-"); ok {
				if v, ok := val(n); ok {
					switch p {
					case "inset":
						return d("inset", sign+v), true
					case "inset-x":
						return []decl{{"left", sign + v}, {"right", sign + v}}, true
					case "inset-y":
						return []decl{{"top", sign + v}, {"bottom", sign + v}}, true
					}
				}
			}
		}
		for _, side := range []string{"top", "right", "bottom", "left"} {
			if n, ok := strings.CutPrefix(b, side+"-"); ok {
				if v, ok := val(n); ok {
					return d(side, sign+v), true
				}
			}
		}
		return nil, false
	})
	register("zIndex", nil, func(base string) ([]decl, bool) {
		if v, ok := arbitrary(base, "z"); ok {
			return d("z-index", v), true
		}
		return negatable(base, "z-", t.ZIndex, "z-index")
	})
	register("gridColumnSpan", nil, staticsLex(
		"col-span-1", "grid-column: span 1 / span 1",
		"col-span-2", "grid-column: span 2 / span 2",
		"col-span-3", "grid-column: span 3 / span 3",
		"col-span-4", "grid-column: span 4 / span 4",
		"col-span-5", "grid-column: span 5 / span 5",
		"col-span-6", "grid-column: span 6 / span 6",
		"col-span-7", "grid-column: span 7 / span 7",
		"col-span-8", "grid-column: span 8 / span 8",
		"col-span-9", "grid-column: span 9 / span 9",
		"col-span-10", "grid-column: span 10 / span 10",
		"col-span-11", "grid-column: span 11 / span 11",
		"col-span-12", "grid-column: span 12 / span 12",
		"col-span-full", "grid-column: 1 / -1",
	))
	register("margin", [][]string{{"m"}, {"mx", "my"}, {"ms", "me", "mt", "mr", "mb", "ml"}}, func(base string) ([]decl, bool) {
		if base == "m-auto" {
			return d("margin", "auto"), true
		}
		for _, p := range []string{"mx-", "my-", "mt-", "mr-", "mb-", "ml-", "ms-", "me-"} {
			if base == p+"auto" {
				out, _ := spacingBox(base[:len(base)-4]+"px", "m", "margin")
				for i := range out {
					out[i].value = "auto"
				}
				return out, true
			}
		}
		return spacingBox(base, "m", "margin")
	})
	register("boxSizing", nil, statics(
		"box-border", "box-sizing: border-box",
		"box-content", "box-sizing: content-box",
	))
	register("display", nil, statics(
		"block", "display: block",
		"inline-block", "display: inline-block",
		"inline", "display: inline",
		"flex", "display: flex",
		"inline-flex", "display: inline-flex",
		"table", "display: table",
		"flow-root", "display: flow-root",
		"grid", "display: grid",
		"inline-grid", "display: inline-grid",
		"contents", "display: contents",
		"hidden", "display: none",
	))
	register("height", nil, func(base string) ([]decl, bool) {
		if v, ok := arbitrary(base, "h"); ok {
			return d("height", v), true
		}
		if n, ok := strings.CutPrefix(base, "h-"); ok {
			if v, ok := twSpacing(n); ok {
				return d("height", v), true
			}
			if strings.Contains(n, "/") {
				if pct, ok := pctFraction(n); ok {
					return d("height", pct), true
				}
			}
			switch n {
			case "auto":
				return d("height", "auto"), true
			case "full":
				return d("height", "100%"), true
			case "screen":
				return d("height", "100vh"), true
			case "min":
				return d("height", "min-content"), true
			case "max":
				return d("height", "max-content"), true
			case "fit":
				return d("height", "fit-content"), true
			}
		}
		return nil, false
	})
	register("maxHeight", nil, statics(
		"max-h-full", "max-height: 100%",
		"max-h-screen", "max-height: 100vh",
	))
	register("minHeight", nil, statics(
		"min-h-0", "min-height: 0px",
		"min-h-full", "min-height: 100%",
		"min-h-screen", "min-height: 100vh",
	))
	register("width", nil, func(base string) ([]decl, bool) {
		if v, ok := arbitrary(base, "w"); ok {
			return d("width", v), true
		}
		if n, ok := strings.CutPrefix(base, "w-"); ok {
			if v, ok := twSpacing(n); ok {
				return d("width", v), true
			}
			if strings.Contains(n, "/") {
				if pct, ok := pctFraction(n); ok {
					return d("width", pct), true
				}
			}
			switch n {
			case "auto":
				return d("width", "auto"), true
			case "full":
				return d("width", "100%"), true
			case "screen":
				return d("width", "100vw"), true
			case "min":
				return []decl{{"width", "-moz-min-content"}, {"width", "min-content"}}, true
			case "max":
				return []decl{{"width", "-moz-max-content"}, {"width", "max-content"}}, true
			case "fit":
				return []decl{{"width", "-moz-fit-content"}, {"width", "fit-content"}}, true
			}
		}
		return nil, false
	})
	register("minWidth", nil, statics(
		"min-w-0", "min-width: 0px",
		"min-w-full", "min-width: 100%",
		"min-w-min", "min-width: min-content",
		"min-w-max", "min-width: max-content",
		"min-w-fit", "min-width: fit-content",
	))
	register("maxWidth", nil, func(base string) ([]decl, bool) {
		if v, ok := arbitrary(base, "max-w"); ok {
			return d("max-width", v), true
		}
		return scaleLookup(base, "max-w-", t.MaxWidth, "max-width")
	})
	register("flex", nil, statics(
		"flex-1", "flex: 1 1 0%",
		"flex-auto", "flex: 1 1 auto",
		"flex-initial", "flex: 0 1 auto",
		"flex-none", "flex: none",
	))
	register("flexShrink", nil, statics(
		"shrink", "flex-shrink: 1",
		"shrink-0", "flex-shrink: 0",
	))
	register("flexGrow", nil, statics(
		"grow", "flex-grow: 1",
		"grow-0", "flex-grow: 0",
	))
	register("scale", nil, func(base string) ([]decl, bool) {
		scaleVals := map[string]string{"0": "0", "50": ".5", "75": ".75", "90": ".9", "95": ".95", "100": "1", "105": "1.05", "110": "1.1", "125": "1.25", "150": "1.5"}
		for _, p := range []string{"scale-x-", "scale-y-", "scale-"} {
			if n, ok := strings.CutPrefix(base, p); ok {
				if v, ok := scaleVals[n]; ok {
					switch p {
					case "scale-":
						return []decl{{"--tw-scale-x", v}, {"--tw-scale-y", v}, {"transform", twTransform}}, true
					case "scale-x-":
						return []decl{{"--tw-scale-x", v}, {"transform", twTransform}}, true
					case "scale-y-":
						return []decl{{"--tw-scale-y", v}, {"transform", twTransform}}, true
					}
				}
			}
		}
		return nil, false
	})
	register("transform", nil, statics(
		"transform", "transform: "+twTransform,
		"transform-none", "transform: none",
	))
	register("animation", nil, staticsLex(
		"animate-none", "animation: none",
		"animate-spin", "animation: spin 1s linear infinite",
		"animate-ping", "animation: ping 1s cubic-bezier(0, 0, 0.2, 1) infinite",
		"animate-pulse", "animation: pulse 2s cubic-bezier(0.4, 0, 0.6, 1) infinite",
		"animate-bounce", "animation: bounce 1s infinite",
	))
	register("cursor", nil, staticsLex(
		"cursor-auto", "cursor: auto",
		"cursor-default", "cursor: default",
		"cursor-pointer", "cursor: pointer",
		"cursor-wait", "cursor: wait",
		"cursor-move", "cursor: move",
		"cursor-not-allowed", "cursor: not-allowed",
	))
	register("userSelect", nil, staticsLex(
		"select-none", "-webkit-user-select: none; -moz-user-select: none; user-select: none",
		"select-text", "-webkit-user-select: text; -moz-user-select: text; user-select: text",
		"select-all", "-webkit-user-select: all; -moz-user-select: all; user-select: all",
		"select-auto", "-webkit-user-select: auto; -moz-user-select: auto; user-select: auto",
	))
	register("listStyleType", nil, staticsLex(
		"list-none", "list-style-type: none",
		"list-disc", "list-style-type: disc",
		"list-decimal", "list-style-type: decimal",
	))
	register("gridTemplateColumns", nil, staticsLex(
		"grid-cols-1", "grid-template-columns: repeat(1, minmax(0, 1fr))",
		"grid-cols-2", "grid-template-columns: repeat(2, minmax(0, 1fr))",
		"grid-cols-3", "grid-template-columns: repeat(3, minmax(0, 1fr))",
		"grid-cols-4", "grid-template-columns: repeat(4, minmax(0, 1fr))",
		"grid-cols-5", "grid-template-columns: repeat(5, minmax(0, 1fr))",
		"grid-cols-6", "grid-template-columns: repeat(6, minmax(0, 1fr))",
		"grid-cols-7", "grid-template-columns: repeat(7, minmax(0, 1fr))",
		"grid-cols-8", "grid-template-columns: repeat(8, minmax(0, 1fr))",
		"grid-cols-9", "grid-template-columns: repeat(9, minmax(0, 1fr))",
		"grid-cols-10", "grid-template-columns: repeat(10, minmax(0, 1fr))",
		"grid-cols-11", "grid-template-columns: repeat(11, minmax(0, 1fr))",
		"grid-cols-12", "grid-template-columns: repeat(12, minmax(0, 1fr))",
	))
	register("flexDirection", nil, statics(
		"flex-row", "flex-direction: row",
		"flex-row-reverse", "flex-direction: row-reverse",
		"flex-col", "flex-direction: column",
		"flex-col-reverse", "flex-direction: column-reverse",
	))
	register("flexWrap", nil, statics(
		"flex-wrap", "flex-wrap: wrap",
		"flex-wrap-reverse", "flex-wrap: wrap-reverse",
		"flex-nowrap", "flex-wrap: nowrap",
	))
	register("alignItems", nil, statics(
		"items-start", "align-items: flex-start",
		"items-end", "align-items: flex-end",
		"items-center", "align-items: center",
		"items-baseline", "align-items: baseline",
		"items-stretch", "align-items: stretch",
	))
	register("justifyContent", nil, statics(
		"justify-normal", "justify-content: normal",
		"justify-start", "justify-content: flex-start",
		"justify-end", "justify-content: flex-end",
		"justify-center", "justify-content: center",
		"justify-between", "justify-content: space-between",
		"justify-around", "justify-content: space-around",
		"justify-evenly", "justify-content: space-evenly",
		"justify-stretch", "justify-content: stretch",
	))
	register("gap", nil, func(base string) ([]decl, bool) {
		if v, ok := arbitrary(base, "gap"); ok {
			return d("gap", v), true
		}
		if n, ok := strings.CutPrefix(base, "gap-x-"); ok {
			if v, ok := twSpacing(n); ok {
				return []decl{{"-moz-column-gap", v}, {"column-gap", v}}, true
			}
		}
		if n, ok := strings.CutPrefix(base, "gap-y-"); ok {
			if v, ok := twSpacing(n); ok {
				return d("row-gap", v), true
			}
		}
		if n, ok := strings.CutPrefix(base, "gap-"); ok {
			if v, ok := twSpacing(n); ok {
				return d("gap", v), true
			}
		}
		return nil, false
	})
	register("alignSelf", nil, statics(
		"self-auto", "align-self: auto",
		"self-start", "align-self: flex-start",
		"self-end", "align-self: flex-end",
		"self-center", "align-self: center",
		"self-stretch", "align-self: stretch",
		"self-baseline", "align-self: baseline",
	))
	register("overflow", nil, statics(
		"overflow-auto", "overflow: auto",
		"overflow-hidden", "overflow: hidden",
		"overflow-clip", "overflow: clip",
		"overflow-visible", "overflow: visible",
		"overflow-scroll", "overflow: scroll",
		"overflow-x-auto", "overflow-x: auto",
		"overflow-y-auto", "overflow-y: auto",
		"overflow-x-hidden", "overflow-x: hidden",
		"overflow-y-hidden", "overflow-y: hidden",
		"overflow-x-scroll", "overflow-x: scroll",
		"overflow-y-scroll", "overflow-y: scroll",
	))
	register("textOverflow", nil, statics(
		"truncate", "overflow: hidden; text-overflow: ellipsis; white-space: nowrap",
		"text-ellipsis", "text-overflow: ellipsis",
		"text-clip", "text-overflow: clip",
	))
	register("whitespace", nil, statics(
		"whitespace-normal", "white-space: normal",
		"whitespace-nowrap", "white-space: nowrap",
		"whitespace-pre", "white-space: pre",
		"whitespace-pre-line", "white-space: pre-line",
		"whitespace-pre-wrap", "white-space: pre-wrap",
		"whitespace-break-spaces", "white-space: break-spaces",
	))
	register("wordBreak", nil, statics(
		"break-normal", "overflow-wrap: normal; word-break: normal",
		"break-words", "overflow-wrap: break-word",
		"break-all", "word-break: break-all",
	))
	register("borderRadius", nil, func(base string) ([]decl, bool) {
		if base == "rounded" {
			return d("border-radius", t.BorderRadius["DEFAULT"]), true
		}
		if v, ok := arbitrary(base, "rounded"); ok {
			return d("border-radius", v), true
		}
		return scaleLookup(base, "rounded-", t.BorderRadius, "border-radius")
	})
	register("borderWidth", nil, func(base string) ([]decl, bool) {
		widths := map[string]string{"": "1px", "0": "0px", "2": "2px", "4": "4px", "8": "8px"}
		if base == "border" {
			return d("border-width", "1px"), true
		}
		for _, side := range []string{"t", "b", "l", "r"} {
			if base == "border-"+side {
				return d("border-"+map[string]string{"t": "top", "b": "bottom", "l": "left", "r": "right"}[side]+"-width", "1px"), true
			}
		}
		if n, ok := strings.CutPrefix(base, "border-"); ok {
			if w, ok := widths[n]; ok {
				return d("border-width", w), true
			}
		}
		return nil, false
	})
	register("borderStyle", nil, statics(
		"border-solid", "border-style: solid",
		"border-dashed", "border-style: dashed",
		"border-dotted", "border-style: dotted",
		"border-double", "border-style: double",
		"border-hidden", "border-style: hidden",
		"border-none", "border-style: none",
	))
	register("borderColor", nil, func(base string) ([]decl, bool) {
		return colorDecls(base, "border-", "border-color", "--tw-border-opacity")
	})
	register("backgroundColor", nil, func(base string) ([]decl, bool) {
		if v, ok := arbitrary(base, "bg"); ok {
			if strings.HasPrefix(v, "url(") {
				return d("background-image", v), true
			}
			return arbitraryColor(v, "background-color", "--tw-bg-opacity")
		}
		return colorDecls(base, "bg-", "background-color", "--tw-bg-opacity")
	})
	register("padding", [][]string{{"p"}, {"px", "py"}, {"ps", "pe", "pt", "pr", "pb", "pl"}}, func(base string) ([]decl, bool) {
		return spacingBox(base, "p", "padding")
	})
	register("textAlign", nil, statics(
		"text-left", "text-align: left",
		"text-center", "text-align: center",
		"text-right", "text-align: right",
		"text-justify", "text-align: justify",
	))
	register("fontSize", nil, func(base string) ([]decl, bool) {
		if v, ok := arbitrary(base, "text"); ok && looksLikeSize(v) {
			return d("font-size", v), true
		}
		if n, ok := strings.CutPrefix(base, "text-"); ok {
			if fs, ok := t.FontSizes()[n]; ok {
				if fs[1] != "" {
					return []decl{{"font-size", fs[0]}, {"line-height", fs[1]}}, true
				}
				return d("font-size", fs[0]), true
			}
		}
		return nil, false
	})
	register("fontWeight", nil, func(base string) ([]decl, bool) {
		return scaleLookup(base, "font-", t.FontWeight, "font-weight")
	})
	register("textTransform", nil, statics(
		"uppercase", "text-transform: uppercase",
		"lowercase", "text-transform: lowercase",
		"capitalize", "text-transform: capitalize",
		"normal-case", "text-transform: none",
	))
	register("fontStyle", nil, statics(
		"italic", "font-style: italic",
		"not-italic", "font-style: normal",
	))
	register("lineHeight", nil, func(base string) ([]decl, bool) {
		if v, ok := arbitrary(base, "leading"); ok {
			return d("line-height", v), true
		}
		return scaleLookup(base, "leading-", t.LineHeight, "line-height")
	})
	register("letterSpacing", nil, func(base string) ([]decl, bool) {
		if v, ok := arbitrary(base, "tracking"); ok {
			return d("letter-spacing", v), true
		}
		return negatable(base, "tracking-", t.LetterSpacing, "letter-spacing")
	})
	register("textColor", nil, func(base string) ([]decl, bool) {
		if v, ok := arbitrary(base, "text"); ok && !looksLikeSize(v) {
			return arbitraryColor(v, "color", "--tw-text-opacity")
		}
		return colorDecls(base, "text-", "color", "--tw-text-opacity")
	})
	register("textDecorationLine", nil, statics(
		"underline", "text-decoration-line: underline",
		"overline", "text-decoration-line: overline",
		"line-through", "text-decoration-line: line-through",
		"no-underline", "text-decoration-line: none",
	))
	register("fontSmoothing", nil, statics(
		"antialiased", "-webkit-font-smoothing: antialiased; -moz-osx-font-smoothing: grayscale",
		"subpixel-antialiased", "-webkit-font-smoothing: auto; -moz-osx-font-smoothing: auto",
	))
	register("opacity", nil, func(base string) ([]decl, bool) {
		if v, ok := arbitrary(base, "opacity"); ok {
			return d("opacity", v), true
		}
		return scaleLookup(base, "opacity-", t.Opacity, "opacity")
	})
	register("boxShadow", nil, func(base string) ([]decl, bool) {
		name := ""
		if base == "shadow" {
			name = "DEFAULT"
		} else if n, ok := strings.CutPrefix(base, "shadow-"); ok {
			name = n
		} else {
			return nil, false
		}
		v, ok := t.BoxShadow[name]
		if !ok {
			return nil, false
		}
		if v == "none" {
			v = "0 0 #0000"
		}
		colored := rgbFuncRe.ReplaceAllString(v, "var(--tw-shadow-color)")
		return []decl{
			{"--tw-shadow", v},
			{"--tw-shadow-colored", colored},
			{"box-shadow", "var(--tw-ring-offset-shadow, 0 0 #0000), var(--tw-ring-shadow, 0 0 #0000), var(--tw-shadow)"},
		}, true
	})
	register("outline", nil, statics(
		"outline-none", "outline: 2px solid transparent; outline-offset: 2px",
		"outline", "outline-style: solid",
	))
	register("ring", nil, func(base string) ([]decl, bool) {
		widths := map[string]string{"ring": "3px", "ring-0": "0px", "ring-1": "1px", "ring-2": "2px", "ring-4": "4px", "ring-8": "8px"}
		w, ok := widths[base]
		if !ok {
			return nil, false
		}
		return []decl{
			{"--tw-ring-offset-shadow", "var(--tw-ring-inset) 0 0 0 var(--tw-ring-offset-width) var(--tw-ring-offset-color)"},
			{"--tw-ring-shadow", "var(--tw-ring-inset) 0 0 0 calc(" + w + " + var(--tw-ring-offset-width)) var(--tw-ring-color)"},
			{"box-shadow", "var(--tw-ring-offset-shadow), var(--tw-ring-shadow), var(--tw-shadow, 0 0 #0000)"},
		}, true
	})
	register("transitionProperty", nil, staticsLex(
		"transition-none", "transition-property: none",
		"transition", "transition-property: color, background-color, border-color, text-decoration-color, fill, stroke, opacity, box-shadow, transform, filter, -webkit-backdrop-filter; transition-property: color, background-color, border-color, text-decoration-color, fill, stroke, opacity, box-shadow, transform, filter, backdrop-filter; transition-property: color, background-color, border-color, text-decoration-color, fill, stroke, opacity, box-shadow, transform, filter, backdrop-filter, -webkit-backdrop-filter; transition-timing-function: cubic-bezier(0.4, 0, 0.2, 1); transition-duration: 150ms",
		"transition-all", "transition-property: all; transition-timing-function: cubic-bezier(0.4, 0, 0.2, 1); transition-duration: 150ms",
		"transition-colors", "transition-property: color, background-color, border-color, text-decoration-color, fill, stroke; transition-timing-function: cubic-bezier(0.4, 0, 0.2, 1); transition-duration: 150ms",
		"transition-opacity", "transition-property: opacity; transition-timing-function: cubic-bezier(0.4, 0, 0.2, 1); transition-duration: 150ms",
		"transition-shadow", "transition-property: box-shadow; transition-timing-function: cubic-bezier(0.4, 0, 0.2, 1); transition-duration: 150ms",
		"transition-transform", "transition-property: transform; transition-timing-function: cubic-bezier(0.4, 0, 0.2, 1); transition-duration: 150ms",
	))
	register("transitionDuration", nil, staticsLex(
		"duration-0", "transition-duration: 0s",
		"duration-75", "transition-duration: 75ms",
		"duration-100", "transition-duration: 100ms",
		"duration-150", "transition-duration: 150ms",
		"duration-200", "transition-duration: 200ms",
		"duration-300", "transition-duration: 300ms",
		"duration-500", "transition-duration: 500ms",
		"duration-700", "transition-duration: 700ms",
		"duration-1000", "transition-duration: 1000ms",
	))
	register("transitionTimingFunction", nil, staticsLex(
		"ease-linear", "transition-timing-function: linear",
		"ease-in", "transition-timing-function: cubic-bezier(0.4, 0, 1, 1)",
		"ease-out", "transition-timing-function: cubic-bezier(0, 0, 0.2, 1)",
		"ease-in-out", "transition-timing-function: cubic-bezier(0.4, 0, 0.2, 1)",
	))
}
