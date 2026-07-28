# go-ruby-opentype

[![CI](https://github.com/go-ruby-opentype/opentype/actions/workflows/ci.yml/badge.svg)](https://github.com/go-ruby-opentype/opentype/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/go-ruby-opentype/opentype.svg)](https://pkg.go.dev/github.com/go-ruby-opentype/opentype)
[![Go Report Card](https://goreportcard.com/badge/github.com/go-ruby-opentype/opentype)](https://goreportcard.com/report/github.com/go-ruby-opentype/opentype)

The pure-Go, Ruby-runtime-independent core of the Ruby **`opentype`** gem — a
complete text stack (font parsing, sized faces, complex-script shaping, the
Unicode Bidirectional Algorithm and a registry of legible fonts) — shaped so
that [go-embedded-ruby](https://github.com/go-embedded-ruby/ruby) (`rbgo`) can
bind it as `require "opentype"`.

It is a thin adapter over the typed libraries of the
[go-opentype](https://github.com/go-opentype) stack:

| Library | Role |
| --- | --- |
| [`go-opentype/opentype`](https://github.com/go-opentype/opentype) | TrueType/OpenType parser + anti-aliased rasteriser. |
| [`go-opentype/shape`](https://github.com/go-opentype/shape) | HarfBuzz-lite complex-text shaper (Arabic, Indic, USE, ...). |
| [`go-opentype/bidi`](https://github.com/go-opentype/bidi) | The Unicode Bidirectional Algorithm (UAX #9). |
| [`go-opentype/fonts`](https://github.com/go-opentype/fonts) | A registry of legible, permissively-licensed families. |

It exposes them through Ruby-facing handles — `Module`, `Font`, `Face` — whose
methods return **Ruby-shaped values**: a **Hash** (`map[string]any`), an
**Array** (`[]any`) or a scalar. A single dynamic entry point, `Call`,
dispatches a Ruby-style snake_case method name to the matching handle method and
coerces the arguments, which is exactly what an rbgo binding drives from
`method_missing`. Nothing here depends on the Ruby runtime, so it is equally
usable as a standalone Go library — a sibling of `go-ruby-regexp/regexp`,
`go-ruby-erb/erb` and `go-ruby-dimail/dimail`.

- **CGO-free**, builds and tests identically on `amd64`, `arm64`, `riscv64`,
  `loong64`, `ppc64le`, `s390x`, plus `js/wasm`.
- **100 % test coverage**, race-clean, enforced in CI.

## The Ruby-facing surface

**`Module`** — the package-level receiver (the `Opentype` module under rbgo):

| Method | Returns |
| --- | --- |
| `open_font(ttf)` / `parse(ttf)` | a `Font` handle |
| `most_legible` | the bytes of the bundled Atkinson Hyperlegible family |
| `default_font` | a `Font` of the most-legible family |
| `load(name)` | the bytes of a bundled family, or the most-legible default |
| `families` | Array of Hashes `{name, kind, license, import_path}` |
| `visual_order(text, base)` | the String reordered to visual order |
| `resolve_levels(text, base)` | Array of the bidi embedding level of each rune |
| `shape(face, text, opts)` | Array of glyph Hashes (see below) |

**`Font`** — a parsed font: `num_glyphs`, `glyph_index(rune)` (an Int or nil),
`axes`, `named_instances`, and `face(px)`.

**`Face`** — a sized font: `measure(text)`, `advance(rune)`, `kern(a, b)`,
`metrics`, `glyph_info(rune, x, y)` (a `GlyphMask`-style Hash), `set_hinting(on)`
and `set_variation(coords)`.

`shape` returns an Array of Hashes, each:

```
{ "gid"=>, "cluster"=>, "x_advance"=>, "y_advance"=>,
  "x_offset"=>, "y_offset"=>, "scale"=> }
```

and its `opts` Hash carries `"direction"` (`"ltr"`/`"rtl"`/`"auto"`),
`"script"` (e.g. `"arab"`), `"features"` (an Array of tags) and `"vertical"`.

## Usage from Ruby

Under rbgo, `require "opentype"` gives an `Opentype` module whose snake_case
methods are these operations, returning Ruby Hashes, Arrays and scalars:

```ruby
require "opentype"

font  = Opentype.open_font(Opentype.most_legible)
puts font.num_glyphs                     # => Integer

face  = font.face(24)
puts face.measure("Hello")               # => Integer

Opentype.shape(face, "بيت").each do |g|  # => Array<Hash>
  # draw glyph g["gid"]; advance the pen by g["x_advance"], etc.
end

puts Opentype.visual_order("aب1", "auto")   # reordered String
```

The `require "opentype"` binding lives in rbgo (a thin `method_missing` shim
over `Call`); it is pending in that repo.

## Install (Go)

```sh
go get github.com/go-ruby-opentype/opentype
```

## Usage from Go

```go
package main

import (
	"fmt"
	"log"

	"github.com/go-ruby-opentype/opentype"
)

func main() {
	m := opentype.NewModule()

	font, err := m.OpenFont(m.MostLegible())
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("glyphs:", font.NumGlyphs())

	face := font.Face(24)
	fmt.Println("width:", face.Measure("Hello"))

	// A shaped run comes back as an Array of Hashes.
	for _, g := range m.Shape(face, "AV", nil) {
		h := g.(map[string]any)
		fmt.Println(h["gid"], h["x_advance"])
	}

	// Mixed-direction text reordered to visual order.
	fmt.Println(m.VisualOrder("aب1", "auto"))
}
```

`Methods(recv)` lists every snake_case name `Call` accepts for a handle, and
`Call(recv, name, args...)` is the uniform dynamic entry point rbgo binds.

## License

BSD-3-Clause. See [LICENSE](LICENSE). Each bundled font keeps its own upstream
license (see `go-opentype/fonts`); this repository's Go code is BSD-3-Clause.
