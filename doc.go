// Copyright (c) 2026 the go-ruby-opentype/opentype authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

// Package opentype is the pure-Go, Ruby-runtime-independent core of the Ruby
// `opentype` gem: a text stack — font parsing, sized faces, complex-script
// shaping, the Unicode Bidirectional Algorithm and a registry of legible
// fonts — shaped so that github.com/go-embedded-ruby/ruby (rbgo) can bind it as
// `require "opentype"`.
//
// It is a thin adapter over the typed libraries of the go-opentype stack —
// github.com/go-opentype/opentype (parse + raster), .../shape (HarfBuzz-lite
// shaper), .../bidi (UAX #9) and .../fonts (bundled families). It exposes them
// through Ruby-facing handles (Module, Font, Face) whose methods return
// Ruby-shaped values: a Hash (map[string]any), an Array ([]any) or a scalar.
// A single dynamic entry point, Call, dispatches a Ruby-style snake_case method
// name to the matching handle method and coerces the arguments, which is exactly
// what an rbgo binding drives from method_missing. Nothing here imports the Ruby
// runtime, so the package is equally usable as a standalone Go library — a
// sibling of go-ruby-regexp/regexp, go-ruby-erb/erb and go-ruby-dimail/dimail.
//
// # Handles
//
//   - Module is the package-level receiver: OpenFont/Parse a font, Load a
//     bundled family, list Families, run VisualOrder/ResolveLevels over text and
//     Shape a run against a Face.
//   - Font is a parsed font: NumGlyphs, GlyphIndex, Axes, NamedInstances and
//     Face(px) to size it.
//   - Face is a sized font: Measure, Advance, Kern, Metrics, GlyphInfo (a
//     GlyphMask-style Hash), SetHinting and SetVariation.
//
// # Usage from Go
//
//	m := opentype.NewModule()
//	font, err := m.OpenFont(opentype.MostLegible())
//	if err != nil {
//		return err
//	}
//	face := font.Face(24)
//	adv := face.Measure("Hello")               // an Int
//	run := m.Shape(face, "بيت", nil)            // an Array of Hashes
//	order := m.VisualOrder("aب1", "auto")      // reordered String
//
// # Usage from Ruby
//
// Under rbgo, `require "opentype"` gives an Opentype module whose snake_case
// methods are these operations, returning Ruby Hashes, Arrays and scalars:
//
//	require "opentype"
//
//	font  = Opentype.open_font(Opentype.most_legible)
//	face  = font.face(24)
//	face.measure("Hello")                      # => Integer
//	Opentype.shape(face, "بيت")                # => Array<Hash>
//	Opentype.visual_order("aب1", "auto")       # => String
//
// The `require "opentype"` binding lives in rbgo (a thin method_missing shim
// over Call); it is pending in that repo.
package opentype
