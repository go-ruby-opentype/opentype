// Copyright (c) 2026 the go-ruby-opentype/opentype authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package opentype

import (
	"reflect"
	"strings"
	"testing"

	"github.com/go-opentype/fonts/notosansarabic"
)

// arabicFont parses the bundled Noto Sans Arabic (a variable font with real
// axes and named instances), used to exercise the RTL/variation code paths.
func arabicFont(t *testing.T) *Font {
	t.Helper()
	f, err := OpenFont(notosansarabic.TTF)
	if err != nil {
		t.Fatalf("parse arabic: %v", err)
	}
	return f
}

func TestOpenFontAndParse(t *testing.T) {
	f, err := OpenFont(MostLegible())
	if err != nil {
		t.Fatal(err)
	}
	if f.NumGlyphs() != 369 {
		t.Fatalf("NumGlyphs = %d", f.NumGlyphs())
	}
	if _, err := Parse(MostLegible()); err != nil {
		t.Fatal(err)
	}
	if _, err := OpenFont([]byte("not a font")); err == nil {
		t.Fatal("want parse error")
	}
	if _, err := NewModule().DefaultFont(); err != nil {
		t.Fatal(err)
	}
	if _, err := DefaultFont(); err != nil {
		t.Fatal(err)
	}
}

func TestLoad(t *testing.T) {
	if Load("") == nil {
		t.Fatal("Load(\"\") should return default bytes")
	}
	if Load("Atkinson Hyperlegible") == nil {
		t.Fatal("Load(default name) should return bytes")
	}
	if Load("atkinson hyperlegible") == nil {
		t.Fatal("Load is case-insensitive")
	}
	if Load("Inter") != nil {
		t.Fatal("Load of a non-embedded family returns nil")
	}
}

func TestFamilies(t *testing.T) {
	fams := Families()
	if len(fams) == 0 {
		t.Fatal("no families")
	}
	first := fams[0].(map[string]any)
	for _, k := range []string{"name", "kind", "license", "import_path"} {
		if _, ok := first[k]; !ok {
			t.Fatalf("family hash missing %q: %#v", k, first)
		}
	}
	if first["name"] != "Atkinson Hyperlegible" {
		t.Fatalf("first family = %v", first["name"])
	}
}

func TestGlyphIndex(t *testing.T) {
	f, _ := OpenFont(MostLegible())
	gid := f.GlyphIndex('A')
	if _, ok := gid.(int); !ok {
		t.Fatalf("GlyphIndex('A') = %#v", gid)
	}
	if got := f.GlyphIndex(''); got != nil { // a private-use rune the font lacks
		t.Fatalf("absent glyph = %#v", got)
	}
}

func TestFaceMetricsAndMeasure(t *testing.T) {
	f, _ := OpenFont(MostLegible())
	face := f.Face(24)
	if face.Measure("Hello") != 55 {
		t.Fatalf("Measure = %d", face.Measure("Hello"))
	}
	if face.Advance('A') <= 0 {
		t.Fatalf("Advance('A') = %d", face.Advance('A'))
	}
	if face.Kern('A', 'V') != -2 {
		t.Fatalf("Kern(A,V) = %d", face.Kern('A', 'V'))
	}
	m := face.Metrics()
	for _, k := range []string{"ascent", "descent", "height", "scale"} {
		if _, ok := m[k]; !ok {
			t.Fatalf("metrics missing %q", k)
		}
	}
}

func TestGlyphInfo(t *testing.T) {
	f, _ := OpenFont(MostLegible())
	face := f.Face(24)
	info := face.GlyphInfo('A', 0, 32)
	if info["found"] != true {
		t.Fatalf("A not found: %#v", info)
	}
	if info["advance"].(int) <= 0 {
		t.Fatal("advance not positive")
	}
	b := info["bounds"].(map[string]any)
	if b["width"].(int) <= 0 || b["height"].(int) <= 0 {
		t.Fatalf("bounds = %#v", b)
	}
	if _, ok := info["origin"].(map[string]any); !ok {
		t.Fatal("missing origin")
	}
	mask := info["mask"].(map[string]any)
	if len(mask["pix"].([]byte)) == 0 {
		t.Fatal("empty mask pix")
	}
	// A space glyph has an empty outline: found, but no mask.
	sp := face.GlyphInfo(' ', 0, 32)
	if _, hasMask := sp["mask"]; hasMask {
		t.Fatalf("space should have no mask: %#v", sp)
	}
}

func TestSetHinting(t *testing.T) {
	f, _ := OpenFont(MostLegible())
	face := f.Face(24)
	face.SetHinting(true)
	face.SetHinting(false)
}

func TestSetVariation(t *testing.T) {
	f := arabicFont(t)
	face := f.Face(32)
	before := face.Measure("بيت")
	out := face.SetVariation(map[string]any{
		"wght": 700,          // int
		"wdth": float64(80),  // float64
		"unit": int64(400),   // int64
		"junk": "not-number", // ignored
	})
	if _, ok := out["junk"]; ok {
		t.Fatal("non-numeric coordinate should be dropped")
	}
	if out["wght"].(float64) != 700 {
		t.Fatalf("normalised wght = %v", out["wght"])
	}
	if after := face.Measure("بيت"); after == before {
		t.Fatalf("SetVariation had no effect: %d == %d", before, after)
	}
}

func TestAxesAndNamedInstances(t *testing.T) {
	f := arabicFont(t)
	axes := f.Axes()
	if len(axes) == 0 {
		t.Fatal("arabic font should report axes")
	}
	a := axes[0].(map[string]any)
	for _, k := range []string{"tag", "min", "default", "max", "flags", "name_id"} {
		if _, ok := a[k]; !ok {
			t.Fatalf("axis missing %q", k)
		}
	}
	nis := f.NamedInstances()
	if len(nis) == 0 {
		t.Fatal("arabic font should report named instances")
	}
	ni := nis[0].(map[string]any)
	if _, ok := ni["coordinates"].(map[string]any); !ok {
		t.Fatalf("named instance missing coordinates: %#v", ni)
	}
	// A non-variable font reports empty (but valid) slices.
	plain, _ := OpenFont(MostLegible())
	if len(plain.Axes()) != 0 || len(plain.NamedInstances()) != 0 {
		t.Fatal("Atkinson should have no axes/instances")
	}
}

func TestShapeLatinAndArabic(t *testing.T) {
	f, _ := OpenFont(MostLegible())
	face := f.Face(24)
	run := Shape(face, "AV", nil)
	if len(run) != 2 {
		t.Fatalf("latin run = %d glyphs", len(run))
	}
	g := run[0].(map[string]any)
	for _, k := range []string{"gid", "cluster", "x_advance", "y_advance", "x_offset", "y_offset", "scale"} {
		if _, ok := g[k]; !ok {
			t.Fatalf("glyph hash missing %q: %#v", k, g)
		}
	}

	af := arabicFont(t)
	aface := af.Face(32)
	ar := Shape(aface, "بيت", map[string]any{
		"direction": "rtl",
		"script":    "arab",
		"vertical":  true,
		"features":  []any{"liga"},
	})
	if len(ar) == 0 {
		t.Fatal("arabic run empty")
	}

	// A nil face yields an empty run.
	if got := Shape(nil, "x", nil); len(got) != 0 {
		t.Fatalf("nil face run = %#v", got)
	}
}

func TestShapeOptionVariants(t *testing.T) {
	f, _ := OpenFont(MostLegible())
	face := f.Face(24)
	// features as a []string, vertical=nil (truthy nil), direction default.
	Shape(face, "AV", map[string]any{
		"features": []string{"kern"},
		"vertical": nil,
	})
	// features as a single string (stringList default branch), vertical=false.
	Shape(face, "AV", map[string]any{
		"features": "kern",
		"vertical": false,
	})
	// features nil value (stringList nil branch).
	Shape(face, "AV", map[string]any{"features": nil})
}

func TestBidi(t *testing.T) {
	if got := VisualOrder("abc", "ltr"); got != "abc" {
		t.Fatalf("VisualOrder ltr = %q", got)
	}
	// A right-to-left run is reordered.
	rtl := VisualOrder("بيت", "rtl")
	if rtl == "بيت" {
		t.Fatalf("rtl run not reordered: %q", rtl)
	}
	_ = VisualOrder("abc", "auto")
	_ = VisualOrder("abc", "")

	levels := ResolveLevels("aب", "auto")
	if len(levels) != 2 {
		t.Fatalf("levels = %#v", levels)
	}
	if levels[0].(int) != 0 {
		t.Fatalf("first level = %v", levels[0])
	}
}

func TestDirection(t *testing.T) {
	// Exercise every alias branch of direction().
	for _, ltr := range []string{"ltr", "l", "lefttoright", "left_to_right"} {
		VisualOrder("a", ltr)
	}
	for _, rtl := range []string{"rtl", "r", "righttoleft", "right_to_left"} {
		VisualOrder("a", rtl)
	}
	VisualOrder("a", "nonsense") // -> Auto
}

func TestCallDispatch(t *testing.T) {
	m := NewModule()
	f, _ := OpenFont(MostLegible())
	face := f.Face(24)

	// Module method returning (value, error): success and error.
	v, err := Call(m, "open_font", string(MostLegible()))
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := v.(*Font); !ok {
		t.Fatalf("open_font returned %T", v)
	}
	if _, err := Call(m, "open_font", "garbage"); err == nil {
		t.Fatal("want parse error via Call")
	}

	// Font method mapping a rune from a String and from an Integer.
	if got, _ := Call(f, "glyph_index", "A"); got == nil {
		t.Fatal("glyph_index(\"A\") nil")
	}
	if got, _ := Call(f, "glyph_index", 65); got == nil {
		t.Fatal("glyph_index(65) nil")
	}
	if got, _ := Call(f, "glyph_index", int64(65)); got == nil {
		t.Fatal("glyph_index(int64) nil")
	}
	if got, _ := Call(f, "glyph_index", float64(65)); got == nil {
		t.Fatal("glyph_index(float64) nil")
	}
	// Omitted trailing argument defaults to nil -> rune 0.
	if _, err := Call(f, "glyph_index"); err != nil {
		t.Fatal(err)
	}

	// Face int argument from int64/float64, and a value method result.
	if _, err := Call(f, "face", int64(24)); err != nil {
		t.Fatal(err)
	}
	if _, err := Call(f, "face", float64(24)); err != nil {
		t.Fatal(err)
	}

	// String argument: nil -> "", and a non-string stringified.
	if got, _ := Call(face, "measure"); got.(int) != 0 {
		t.Fatalf("measure() = %v", got)
	}
	if _, err := Call(face, "measure", 123); err != nil {
		t.Fatal(err)
	}

	// Bool argument via coerce: nil -> false, non-bool -> truthy. (A real bool
	// takes the assignable fast-path.)
	if _, err := Call(face, "set_hinting", true); err != nil {
		t.Fatal(err)
	}
	if _, err := Call(face, "set_hinting", nil); err != nil {
		t.Fatal(err)
	}
	if _, err := Call(face, "set_hinting", "on"); err != nil {
		t.Fatal(err)
	}

	// []byte argument: assignable, from String, and nil.
	if _, err := Call(m, "open_font", MostLegible()); err != nil {
		t.Fatal(err)
	}
	if _, err := Call(m, "open_font"); err == nil {
		t.Fatal("open_font(nil) should fail to parse")
	}

	// Pointer argument: assignable *Face, and nil *Face.
	if got, _ := Call(m, "shape", face, "AV", nil); len(got.([]any)) == 0 {
		t.Fatal("shape via Call empty")
	}
	if got, _ := Call(m, "shape", nil, "AV", nil); len(got.([]any)) != 0 {
		t.Fatal("shape nil face should be empty")
	}

	// Map argument: assignable map[string]any.
	if _, err := Call(face, "set_variation", map[string]any{"wght": 700}); err != nil {
		t.Fatal(err)
	}

	// A method returning nothing yields (nil, nil).
	if v, err := Call(face, "set_hinting", true); v != nil || err != nil {
		t.Fatalf("void call = %v, %v", v, err)
	}
}

func TestCallErrors(t *testing.T) {
	m := NewModule()
	f, _ := OpenFont(MostLegible())
	face := f.Face(24)

	if _, err := Call(nil, "x"); err == nil || !strings.Contains(err.Error(), "nil receiver") {
		t.Fatalf("want nil receiver, got %v", err)
	}
	if _, err := Call(m, "no_such"); err == nil || !strings.Contains(err.Error(), "unknown method") {
		t.Fatalf("want unknown method, got %v", err)
	}
	if _, err := Call(face, "measure", "a", "b"); err == nil || !strings.Contains(err.Error(), "at most") {
		t.Fatalf("want too-many-args, got %v", err)
	}
	// rune coercion failures.
	if _, err := Call(f, "glyph_index", ""); err == nil || !strings.Contains(err.Error(), "argument") {
		t.Fatalf("want empty-char error, got %v", err)
	}
	if _, err := Call(f, "glyph_index", true); err == nil {
		t.Fatal("want non-character error")
	}
	// int coercion failure.
	if _, err := Call(f, "face", "big"); err == nil {
		t.Fatal("want integer error")
	}
	// []byte coercion failure.
	if _, err := Call(m, "open_font", 123); err == nil {
		t.Fatal("want bytes error")
	}
	// pointer coercion failure (non-nil, non-assignable).
	if _, err := Call(m, "shape", "not-a-face", "x", nil); err == nil {
		t.Fatal("want pointer error")
	}
	// map coercion failure (default branch).
	if _, err := Call(face, "set_variation", "not-a-map"); err == nil {
		t.Fatal("want map error")
	}
}

func TestMethods(t *testing.T) {
	names := Methods(NewModule())
	want := map[string]bool{"open_font": false, "visual_order": false, "resolve_levels": false, "shape": false}
	for _, n := range names {
		if _, ok := want[n]; ok {
			want[n] = true
		}
	}
	for n, seen := range want {
		if !seen {
			t.Fatalf("Methods() missing %q: %v", n, names)
		}
	}
	// Sorted.
	for i := 1; i < len(names); i++ {
		if names[i-1] > names[i] {
			t.Fatalf("Methods() not sorted: %v", names)
		}
	}
}

func TestNameHelpers(t *testing.T) {
	if got := camelize("glyph_index"); got != "GlyphIndex" {
		t.Fatalf("camelize = %q", got)
	}
	// Empty words (leading/double underscores) are skipped.
	if got := camelize("_open__font_"); got != "OpenFont" {
		t.Fatalf("camelize empties = %q", got)
	}
	if got := snakeize("GlyphIndex"); got != "glyph_index" {
		t.Fatalf("snakeize = %q", got)
	}
	// An acronym run followed by a lowercase (the nextLower boundary).
	if got := snakeize("HTTPServer"); got != "http_server" {
		t.Fatalf("snakeize acronym = %q", got)
	}
}

func TestCoerceHelpersDirect(t *testing.T) {
	// The int fast-path inside toInt (unreachable through coerce, since a Go
	// int is directly assignable).
	if n, ok := toInt(7); !ok || n != 7 {
		t.Fatalf("toInt(int) = %d, %v", n, ok)
	}
	// toFloat over every accepted numeric kind and its rejection.
	if v, ok := toFloat(3.5); !ok || v != 3.5 {
		t.Fatalf("toFloat(float64) = %v, %v", v, ok)
	}
	if v, ok := toFloat(3); !ok || v != 3 {
		t.Fatalf("toFloat(int) = %v, %v", v, ok)
	}
	if v, ok := toFloat(int64(3)); !ok || v != 3 {
		t.Fatalf("toFloat(int64) = %v, %v", v, ok)
	}
	if _, ok := toFloat("x"); ok {
		t.Fatal("toFloat(string) should fail")
	}
	// truthy over its three cases.
	if truthy(nil) || !truthy(true) || truthy(false) || !truthy("x") {
		t.Fatal("truthy branch wrong")
	}
	// The non-byte Slice branch of coerce is unreachable through Call (no
	// method takes such a slice); exercise it directly.
	if _, err := coerce([]any{1}, reflect.TypeOf([]int{})); err == nil {
		t.Fatal("want non-byte slice error")
	}
	// The default branch: an unsupported, non-assignable Go kind (no method
	// parameter is a float, so this is only reachable directly).
	if _, err := coerce(1, reflect.TypeOf(float64(0))); err == nil {
		t.Fatal("want unsupported-kind error")
	}
	// stringList over a []string (also unreachable via a Ruby Hash, which
	// carries []any).
	if got := stringList([]string{"a"}); len(got) != 1 || got[0] != "a" {
		t.Fatalf("stringList([]string) = %v", got)
	}
}
