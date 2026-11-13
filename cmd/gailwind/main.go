// gailwind generates Tailwind-v3-compatible utility CSS from source files,
// in pure Go. It covers the v3 default-theme core utility set; classes it
// cannot compile are listed on stderr and (with -strict) fail the build.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ofershap/gailwind"
)

func main() {
	out := flag.String("o", "", "output CSS file (default stdout)")
	minify := flag.Bool("minify", false, "minify output")
	strict := flag.Bool("strict", false, "exit 1 if any candidate class is unknown")
	flag.Parse()
	files := flag.Args()
	if len(files) == 0 {
		fmt.Fprintln(os.Stderr, "usage: gailwind [-o out.css] [-minify] [-strict] <content files...>")
		os.Exit(2)
	}
	var sources []string
	for _, f := range files {
		matches, err := filepath.Glob(f)
		if err != nil || len(matches) == 0 {
			matches = []string{f}
		}
		for _, m := range matches {
			b, err := os.ReadFile(m)
			if err != nil {
				fmt.Fprintln(os.Stderr, "gailwind: "+err.Error())
				os.Exit(2)
			}
			sources = append(sources, string(b))
		}
	}
	css, rep := gailwind.Generate(sources...)
	if *minify {
		css = minifyCSS(css)
	}
	for _, u := range rep.Unknown {
		fmt.Fprintf(os.Stderr, "gailwind: unknown class %q\n", u)
	}
	if len(rep.Unknown) > 0 {
		fmt.Fprintf(os.Stderr, "gailwind: %d of %d candidates not generated (subset engine; see README)\n", len(rep.Unknown), rep.Generated+len(rep.Unknown))
	}
	if *out == "" {
		fmt.Print(css)
	} else if err := os.WriteFile(*out, []byte(css), 0644); err != nil {
		fmt.Fprintln(os.Stderr, "gailwind: "+err.Error())
		os.Exit(2)
	}
	if *strict && len(rep.Unknown) > 0 {
		os.Exit(1)
	}
}

func minifyCSS(css string) string {
	css = strings.ReplaceAll(css, ": ", ":")
	css = strings.ReplaceAll(css, "; ", ";")
	css = strings.ReplaceAll(css, " { ", "{")
	css = strings.ReplaceAll(css, " }\n", "}")
	css = strings.ReplaceAll(css, " {\n", "{")
	css = strings.ReplaceAll(css, "}\n", "}")
	return css
}
