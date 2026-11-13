# gailwind

Tailwind v3 utility CSS from Go source files - as a Go library and a `go install`-able CLI, with no separate binary or runtime to vendor.

## What this is NOT

Read this first. gailwind is a **subset engine**, not a Tailwind replacement:

- **Not a full Tailwind port.** It implements the v3.4.19 **default-theme core utility set** (layout, spacing, sizing, flex/grid, typography, colors, borders, effects, transforms, interactivity, transitions/animations) with variants (pseudo-classes, `group`/`peer`, `sm`-`2xl`, `dark`, orientation, motion, print) and arbitrary values (`w-[17px]`, `bg-[#bada55]`).
- **No plugins, no custom themes/config files, no `@apply`.** Gradients, filters, `space-*`/`divide-*`, and `ring` color/offset sub-utilities are not implemented yet.
- **Not measured against all of Tailwind.** Coverage claims below refer to gailwind's own implemented-feature test corpus, not to a percentage of Tailwind.

Classes gailwind can't compile are **reported on stderr, never silently dropped** (and fail the build with `-strict`). If you hit one, it's either on the not-implemented list above or a bug - the report tells you which.

Preflight and `container` are included: gailwind embeds the real v3.4.19 base reset verbatim (Tailwind Labs, MIT - see `licenses/`) and prepends it by default (`-no-preflight` to opt out), so output matches `@tailwind base; @tailwind components; @tailwind utilities;`.

## Why

Official Tailwind already ships a standalone CLI. gailwind exists for a different reason: it's a **Go-native engine** -

- import it as a **library** (`github.com/ofershap/gailwind`) and generate CSS inside your Go build pipeline, templ handler, or code generator;
- or `go install` the CLI in a pure-Go toolchain where fetching and pinning a separate platform binary is unwanted friction.

## Install

```sh
go install github.com/ofershap/gailwind/cmd/gailwind@latest
```

## Use

```sh
gailwind -o app.css './templates/**/*.templ' './web/**/*.html'
gailwind -minify -strict -o app.css index.html
gailwind -no-preflight -o utilities-only.css index.html
```

As a library:

```go
css, report := gailwind.Generate(source1, source2)
// report.Unknown lists class-attribute tokens gailwind did not generate
```

Output CSS is ordered to match tailwindcss v3.4.19's own rule order, so cascade behavior matches what Tailwind produces for the same classes.

## Measured fidelity

Verified against the real `tailwindcss@3.4.19` CLI, not self-reported:

- **290/290 rules - 100% declaration match, 0 order inversions** on gailwind's implemented-feature corpus (`testdata/corpus.html`, ~200 classes across every implemented family, including variant stacks, negatives and arbitrary values). Reproduce with `parity/parity.py`.
- **Real-template check:** on the unmodified [tailwindtoolbox Landing-Page](https://github.com/tailwindtoolbox/Landing-Page), gailwind's full output (preflight + container + utilities) matches the real v3.4.19 build on **206/206 flattened rules - 100% declaration match**. Of 169 class-attribute candidates, 160 generated; the 9 reported unknowns are v2-era/nonexistent classes real v3.4.19 doesn't generate either, plus the template's own custom classes. Element-anchored side-by-side renders are visually identical.

## How it works

1. **Extract** candidate classes from source text (Tailwind-style candidate regex).
2. **Compile** each candidate against the real v3.4.19 default theme (embedded, dumped from `resolveConfig`) into the same declarations v3 emits - including its `rgb(R G B / var(--tw-*-opacity, 1))` color form, `--tw-shadow` triplets, vendor-prefixed fallbacks and keyframes.
3. **Order** rules the way v3's variant offsets order them (verified empirically against probe outputs).

## FAQ

<details><summary>Does it watch files?</summary><code>-watch</code> is planned; today you re-run the command (it's fast - pure Go, no startup tax).</details>
<details><summary>What happens with a class it doesn't know?</summary>It's printed to stderr and skipped; <code>-strict</code> exits 1. Unknown detection only flags tokens you actually wrote in a <code>class</code> attribute, so SVG path data and prose don't produce noise.</details>
<details><summary>Dark mode?</summary><code>dark:</code> compiles to <code>@media (prefers-color-scheme: dark)</code>, v3's default strategy. Class-based dark mode is not implemented.</details>
<details><summary>Can I use my own theme?</summary>Not yet - the v3.4.19 default theme is the contract. Custom themes are on the roadmap behind a config file.</details>
