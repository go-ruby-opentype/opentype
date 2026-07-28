// Copyright (c) 2026 the go-ruby-opentype/opentype authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package opentype_test

import (
	"fmt"
	"log"

	"github.com/go-ruby-opentype/opentype"
)

// Example mirrors the README: parse the bundled most-legible font, size it,
// measure a string, shape a run and reorder mixed-direction text — every result
// a Ruby-shaped value.
func Example() {
	m := opentype.NewModule()

	font, err := m.OpenFont(m.MostLegible())
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("glyphs:", font.NumGlyphs())

	face := font.Face(24)
	fmt.Println("width:", face.Measure("Hello"))

	run := m.Shape(face, "AV", nil) // an Array of Hashes
	fmt.Println("shaped:", len(run))

	fmt.Println("order:", m.VisualOrder("abc", "ltr"))

	// Output:
	// glyphs: 369
	// width: 55
	// shaped: 2
	// order: abc
}
