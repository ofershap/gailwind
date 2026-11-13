// Package theme holds the Tailwind CSS v3 default theme, extracted verbatim
// from tailwindcss v3.4.19 (see theme.json header provenance in the repo).
package theme

import (
	_ "embed"
	"encoding/json"
)

//go:embed theme.json
var raw []byte

// Theme is the subset of the v3 default theme gailwind generates from.
type Theme struct {
	Screens       map[string]string `json:"screens"`
	Spacing       map[string]string `json:"spacing"`
	Colors        map[string]any    `json:"colors"` // nested: {"blue":{"500":"#3b82f6"}}
	FontSize      map[string]any    `json:"fontSize"`
	FontWeight    map[string]string `json:"fontWeight"`
	BorderRadius  map[string]string `json:"borderRadius"`
	BoxShadow     map[string]string `json:"boxShadow"`
	MaxWidth      map[string]string `json:"maxWidth"`
	ZIndex        map[string]string `json:"zIndex"`
	Opacity       map[string]string `json:"opacity"`
	LineHeight    map[string]string `json:"lineHeight"`
	LetterSpacing map[string]string `json:"letterSpacing"`
}

// Default is the pinned v3.4.19 default theme.
func Default() Theme {
	var t Theme
	if err := json.Unmarshal(raw, &t); err != nil {
		panic(err)
	}
	return t
}

// FlatColors flattens the nested color map to "blue-500" -> "#3b82f6".
func (t Theme) FlatColors() map[string]string {
	out := map[string]string{}
	var walk func(m map[string]any, prefix string)
	walk = func(m map[string]any, prefix string) {
		for k, v := range m {
			switch vv := v.(type) {
			case string:
				out[prefix+k] = vv
			case map[string]any:
				walk(vv, prefix+k+"-")
			}
		}
	}
	walk(t.Colors, "")
	return out
}

// FontSizes flattens fontSize to "sm" -> (size, lineHeight).
func (t Theme) FontSizes() map[string][2]string {
	out := map[string][2]string{}
	for k, v := range t.FontSize {
		switch vv := v.(type) {
		case string:
			out[k] = [2]string{vv, ""}
		case []any:
			if len(vv) >= 1 {
				size, _ := vv[0].(string)
				lh := ""
				if len(vv) >= 2 {
					if opts, ok := vv[1].(map[string]any); ok {
						lh, _ = opts["lineHeight"].(string)
					}
				}
				out[k] = [2]string{size, lh}
			}
		}
	}
	return out
}
