// Copyright (c) 2026 the go-ruby-opentype/opentype authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package opentype

import (
	"fmt"
	"reflect"
	"sort"
	"strings"
	"unicode"

	"github.com/go-opentype/bidi"
	"github.com/go-opentype/fonts"
	otf "github.com/go-opentype/opentype"
	"github.com/go-opentype/shape"
)

// Module is the package-level Ruby receiver: the `Opentype` module under rbgo.
// Its methods parse fonts, load bundled families, run the Unicode
// Bidirectional Algorithm and shape text, all returning Ruby-shaped values.
// A Module is stateless and safe for concurrent use.
type Module struct{}

// NewModule returns the package-level receiver. The package-level convenience
// functions (OpenFont, Parse, Load, Families, VisualOrder, ResolveLevels and
// Shape) delegate to it.
func NewModule() *Module { return &Module{} }

var mod = &Module{}

// Font is a Ruby-facing handle over a parsed github.com/go-opentype/opentype
// font. It is immutable and safe for concurrent use.
type Font struct{ f *otf.Font }

// Face is a Ruby-facing handle over a sized github.com/go-opentype/opentype
// face. Like the underlying face it caches rasterised glyphs and is not safe
// for concurrent use.
type Face struct{ fc *otf.Face }

// OpenFont parses a TrueType/OpenType blob into a Font handle. It wraps
// opentype.Parse.
func (m *Module) OpenFont(ttf []byte) (*Font, error) {
	f, err := otf.Parse(ttf)
	if err != nil {
		return nil, err
	}
	return &Font{f: f}, nil
}

// Parse is an alias for OpenFont, mirroring fonts.Parse.
func (m *Module) Parse(ttf []byte) (*Font, error) { return m.OpenFont(ttf) }

// MostLegible returns the bytes of the one family embedded directly in the
// fonts registry — Atkinson Hyperlegible, designed for maximum legibility.
func (m *Module) MostLegible() []byte { return fonts.MostLegible() }

// DefaultFont parses MostLegible into a ready Font handle.
func (m *Module) DefaultFont() (*Font, error) { return m.OpenFont(m.MostLegible()) }

// Load returns the bytes of a bundled family by name. Only the registry's
// embedded default (MostLegible) carries bytes in this package — every other
// family lives in its own subpackage and must be loaded by importing it — so
// Load returns those default bytes when name is empty or names the default
// family (case-insensitively), and nil otherwise.
func (m *Module) Load(name string) []byte {
	if name == "" {
		return m.MostLegible()
	}
	// fonts.All is curated-first and always non-empty; its head is the one
	// family embedded directly here (the MostLegible default).
	if def := fonts.All()[0]; strings.EqualFold(name, def.Name) {
		return m.MostLegible()
	}
	return nil
}

// Families lists every bundled family as an Array of Hashes, each with "name",
// "kind", "license" and "import_path". It wraps fonts.All (metadata only — no
// bytes are linked).
func (m *Module) Families() []any {
	all := fonts.All()
	out := make([]any, len(all))
	for i, fam := range all {
		out[i] = map[string]any{
			"name":        fam.Name,
			"kind":        fam.Kind.String(),
			"license":     fam.License,
			"import_path": fam.ImportPath,
		}
	}
	return out
}

// VisualOrder resolves and reorders text to visual (left-to-right) order under
// the given base direction ("ltr", "rtl" or "auto"; empty means auto). It
// wraps bidi.VisualOrder and returns a String.
func (m *Module) VisualOrder(text, base string) string {
	return bidi.VisualOrder(text, direction(base))
}

// ResolveLevels runs the Unicode Bidirectional Algorithm over text and returns
// an Array of the resolved embedding level (an Int) of every rune, under the
// given base direction ("ltr", "rtl" or "auto"). It wraps bidi.ResolveLevels.
func (m *Module) ResolveLevels(text, base string) []any {
	levels := bidi.ResolveLevels([]rune(text), direction(base))
	out := make([]any, len(levels))
	for i, lv := range levels {
		out[i] = int(lv)
	}
	return out
}

// Shape turns text into a positioned glyph run against face, returning an Array
// of Hashes, each with "gid", "cluster", "x_advance", "y_advance", "x_offset",
// "y_offset" and "scale". It wraps shape.Shape. The opts Hash carries
// "direction" ("ltr"/"rtl"/"auto"), "script" (e.g. "arab"), "features" (an
// Array of feature tags) and "vertical" (a Bool). A nil face yields an empty
// Array.
func (m *Module) Shape(face *Face, text string, opts map[string]any) []any {
	if face == nil {
		return []any{}
	}
	glyphs := shape.Shape(face.fc, text, shapeOptions(opts))
	out := make([]any, len(glyphs))
	for i, g := range glyphs {
		out[i] = map[string]any{
			"gid":       int(g.GID),
			"cluster":   g.Cluster,
			"x_advance": g.XAdvance,
			"y_advance": g.YAdvance,
			"x_offset":  g.XOffset,
			"y_offset":  g.YOffset,
			"scale":     g.Scale,
		}
	}
	return out
}

// shapeOptions builds shape.Options from a Ruby opts Hash. A nil Hash is the
// zero Options.
func shapeOptions(opts map[string]any) shape.Options {
	var o shape.Options
	if opts == nil {
		return o
	}
	if v, ok := opts["direction"]; ok {
		o.Direction = direction(fmt.Sprint(v))
	}
	if v, ok := opts["script"]; ok {
		o.Script = fmt.Sprint(v)
	}
	if v, ok := opts["vertical"]; ok {
		o.Vertical = truthy(v)
	}
	if v, ok := opts["features"]; ok {
		o.Features = stringList(v)
	}
	return o
}

// direction maps a Ruby direction name to a bidi.Direction. Unknown names
// (including the empty string) select Auto.
func direction(name string) bidi.Direction {
	switch strings.ToLower(name) {
	case "ltr", "l", "lefttoright", "left_to_right":
		return bidi.LeftToRight
	case "rtl", "r", "righttoleft", "right_to_left":
		return bidi.RightToLeft
	default:
		return bidi.Auto
	}
}

// stringList coerces a Ruby Array (or a single String) into a []string of
// feature tags.
func stringList(v any) []string {
	switch x := v.(type) {
	case nil:
		return nil
	case []string:
		return x
	case []any:
		out := make([]string, len(x))
		for i, e := range x {
			out[i] = fmt.Sprint(e)
		}
		return out
	default:
		return []string{fmt.Sprint(x)}
	}
}

// NumGlyphs reports how many glyphs the font contains.
func (f *Font) NumGlyphs() int { return f.f.NumGlyphs() }

// GlyphIndex maps a rune to its glyph id (an Int), or nil when the font has no
// glyph for it.
func (f *Font) GlyphIndex(r rune) any {
	gid, ok := f.f.GlyphIndex(r)
	if !ok {
		return nil
	}
	return int(gid)
}

// Face sizes the font at px pixels, returning a Face handle. It wraps
// Font.NewFace.
func (f *Font) Face(px int) *Face { return &Face{fc: f.f.NewFace(px)} }

// Axes lists the font's variation axes as an Array of Hashes, each with "tag",
// "min", "default", "max", "flags" and "name_id". A non-variable font yields an
// empty Array.
func (f *Font) Axes() []any {
	axes := f.f.Axes()
	out := make([]any, len(axes))
	for i, a := range axes {
		out[i] = map[string]any{
			"tag":     a.Tag,
			"min":     a.Min,
			"default": a.Default,
			"max":     a.Max,
			"flags":   int(a.Flags),
			"name_id": int(a.NameID),
		}
	}
	return out
}

// NamedInstances lists the font's named variation instances as an Array of
// Hashes, each with "subfamily_name_id", "flags", "coordinates" (a Hash of axis
// tag to coordinate) and "post_script_name_id".
func (f *Font) NamedInstances() []any {
	nis := f.f.NamedInstances()
	out := make([]any, len(nis))
	for i, ni := range nis {
		coords := make(map[string]any, len(ni.Coordinates))
		for tag, c := range ni.Coordinates {
			coords[tag] = c
		}
		out[i] = map[string]any{
			"subfamily_name_id":   int(ni.SubfamilyNameID),
			"flags":               int(ni.Flags),
			"coordinates":         coords,
			"post_script_name_id": int(ni.PostScriptNameID),
		}
	}
	return out
}

// Measure returns the advance width of text in whole pixels, ignoring kerning.
func (fc *Face) Measure(text string) int { return fc.fc.Measure(text) }

// Advance returns the advance width of a single rune in whole pixels.
func (fc *Face) Advance(r rune) int { return fc.fc.Advance(r) }

// Kern returns the kerning adjustment, in pixels, between two adjacent runes.
func (fc *Face) Kern(prev, r rune) int { return fc.fc.Kern(prev, r) }

// Metrics returns the face's vertical metrics as a Hash with "ascent",
// "descent", "height" and "scale".
func (fc *Face) Metrics() map[string]any {
	m := fc.fc.Metrics()
	return map[string]any{
		"ascent":  m.Ascent,
		"descent": m.Descent,
		"height":  m.Height,
		"scale":   fc.fc.Scale(),
	}
}

// GlyphInfo returns a GlyphMask-style Hash for rune r placed at pen (x, y):
// "found" (a Bool), "advance", the "bounds" of the placed glyph (a Hash with
// "min_x"/"min_y"/"max_x"/"max_y"/"width"/"height"), the mask "origin" (a Hash
// with "x"/"y"), and, when found, the 8-bit alpha coverage "mask" as a Hash
// with "width"/"height"/"stride"/"pix" (the raw coverage bytes). It wraps
// Face.GlyphMask.
func (fc *Face) GlyphInfo(r rune, x, y int) map[string]any {
	bounds, mask, mp, advance, ok := fc.fc.GlyphMask(r, x, y)
	info := map[string]any{
		"found":   ok,
		"advance": advance,
		"bounds": map[string]any{
			"min_x":  bounds.Min.X,
			"min_y":  bounds.Min.Y,
			"max_x":  bounds.Max.X,
			"max_y":  bounds.Max.Y,
			"width":  bounds.Dx(),
			"height": bounds.Dy(),
		},
		"origin": map[string]any{"x": mp.X, "y": mp.Y},
	}
	if mask != nil {
		info["mask"] = map[string]any{
			"width":  mask.Rect.Dx(),
			"height": mask.Rect.Dy(),
			"stride": mask.Stride,
			"pix":    mask.Pix,
		}
	}
	return info
}

// SetHinting turns the face's grid-fitting hinting on or off.
func (fc *Face) SetHinting(on bool) { fc.fc.SetHinting(on) }

// SetVariation moves the face along its variation axes. coords is a Hash of
// axis tag to a numeric user coordinate (e.g. {"wght" => 700}); non-numeric
// values are ignored. It returns the normalised coordinates as a Ruby Hash and
// wraps Face.SetVariation.
func (fc *Face) SetVariation(coords map[string]any) map[string]any {
	norm := make(map[string]float64, len(coords))
	out := make(map[string]any, len(coords))
	for tag, v := range coords {
		f, ok := toFloat(v)
		if !ok {
			continue
		}
		norm[tag] = f
		out[tag] = f
	}
	fc.fc.SetVariation(norm)
	return out
}

// --- dynamic dispatch (the uniform surface rbgo binds via method_missing) ---

var errorType = reflect.TypeOf((*error)(nil)).Elem()

// Call dispatches a Ruby-style snake_case method name to the matching exported
// method of recv (a *Module, *Font or *Face), coercing each Ruby-supplied
// argument to the Go parameter type. Trailing arguments may be omitted; they
// default to nil. The result is the method's Ruby-shaped return value (or nil
// for a method that returns nothing); a trailing error return is unwrapped into
// Call's own error. This is the single entry point an rbgo binding drives.
func Call(recv any, method string, args ...any) (any, error) {
	rv := reflect.ValueOf(recv)
	if !rv.IsValid() {
		return nil, fmt.Errorf("opentype: nil receiver")
	}
	m := rv.MethodByName(camelize(method))
	if !m.IsValid() {
		return nil, fmt.Errorf("opentype: unknown method %q", method)
	}
	mt := m.Type()
	if len(args) > mt.NumIn() {
		return nil, fmt.Errorf("opentype: %s takes at most %d argument(s), got %d",
			method, mt.NumIn(), len(args))
	}

	in := make([]reflect.Value, mt.NumIn())
	for i := 0; i < mt.NumIn(); i++ {
		var arg any
		if i < len(args) {
			arg = args[i]
		}
		v, err := coerce(arg, mt.In(i))
		if err != nil {
			return nil, fmt.Errorf("opentype: %s argument %d: %w", method, i+1, err)
		}
		in[i] = v
	}

	outs := m.Call(in)
	if n := len(outs); n > 0 && mt.Out(n-1) == errorType {
		if e := outs[n-1]; !e.IsNil() {
			return nil, e.Interface().(error)
		}
		outs = outs[:n-1]
	}
	if len(outs) == 0 {
		return nil, nil
	}
	return outs[0].Interface(), nil
}

// Methods lists, sorted, the Ruby-style snake_case names Call accepts for recv.
func Methods(recv any) []string {
	t := reflect.TypeOf(recv)
	names := make([]string, 0, t.NumMethod())
	for i := 0; i < t.NumMethod(); i++ {
		names = append(names, snakeize(t.Method(i).Name))
	}
	sort.Strings(names)
	return names
}

// coerce converts a Ruby-supplied argument into the Go type target expects.
func coerce(val any, target reflect.Type) (reflect.Value, error) {
	if val != nil {
		if vv := reflect.ValueOf(val); vv.Type().AssignableTo(target) {
			return vv, nil
		}
	}
	switch target.Kind() {
	case reflect.Int32: // rune
		r, err := toRune(val)
		if err != nil {
			return reflect.Value{}, err
		}
		return reflect.ValueOf(r).Convert(target), nil
	case reflect.Int, reflect.Int64:
		n, ok := toInt(val)
		if !ok {
			return reflect.Value{}, fmt.Errorf("cannot use %T as an integer", val)
		}
		return reflect.ValueOf(n).Convert(target), nil
	case reflect.String:
		if val == nil {
			return reflect.ValueOf("").Convert(target), nil
		}
		return reflect.ValueOf(fmt.Sprint(val)).Convert(target), nil
	case reflect.Bool:
		if val == nil {
			return reflect.ValueOf(false), nil
		}
		return reflect.ValueOf(truthy(val)), nil
	case reflect.Slice:
		if target.Elem().Kind() == reflect.Uint8 { // []byte
			b, err := toBytes(val)
			if err != nil {
				return reflect.Value{}, err
			}
			return reflect.ValueOf(b).Convert(target), nil
		}
		return reflect.Value{}, fmt.Errorf("cannot use %T as %s", val, target)
	case reflect.Map:
		if val == nil { // an omitted Hash becomes a nil map
			return reflect.Zero(target), nil
		}
		return reflect.Value{}, fmt.Errorf("cannot use %T as %s", val, target)
	case reflect.Pointer:
		if val == nil {
			return reflect.Zero(target), nil
		}
		return reflect.Value{}, fmt.Errorf("cannot use %T as %s", val, target)
	default: // any other Go kind must be directly assignable
		return reflect.Value{}, fmt.Errorf("cannot use %T as %s", val, target)
	}
}

// toRune extracts a rune from a Ruby value: an Integer code point or the first
// character of a String.
func toRune(val any) (rune, error) {
	switch x := val.(type) {
	case nil:
		return 0, nil
	case string:
		rs := []rune(x)
		if len(rs) == 0 {
			return 0, fmt.Errorf("empty string is not a character")
		}
		return rs[0], nil
	case int:
		return rune(x), nil
	case int64:
		return rune(x), nil
	case float64:
		return rune(int64(x)), nil
	default:
		return 0, fmt.Errorf("cannot use %T as a character", val)
	}
}

// toInt coerces a Ruby number to an int.
func toInt(val any) (int, bool) {
	switch x := val.(type) {
	case int:
		return x, true
	case int64:
		return int(x), true
	case float64:
		return int(x), true
	default:
		return 0, false
	}
}

// toFloat coerces a Ruby number to a float64.
func toFloat(val any) (float64, bool) {
	switch x := val.(type) {
	case float64:
		return x, true
	case int:
		return float64(x), true
	case int64:
		return float64(x), true
	default:
		return 0, false
	}
}

// toBytes coerces a Ruby String (or byte slice) into a []byte.
func toBytes(val any) ([]byte, error) {
	switch x := val.(type) {
	case nil:
		return nil, nil
	case string:
		return []byte(x), nil
	default:
		return nil, fmt.Errorf("cannot use %T as bytes", val)
	}
}

// truthy applies Ruby truthiness: only false and nil are falsey.
func truthy(val any) bool {
	switch x := val.(type) {
	case nil:
		return false
	case bool:
		return x
	default:
		return true
	}
}

// camelize turns a snake_case method name into its exported Go method name.
func camelize(s string) string {
	var b strings.Builder
	for _, w := range strings.Split(s, "_") {
		if w == "" {
			continue
		}
		r := []rune(w)
		r[0] = unicode.ToUpper(r[0])
		b.WriteString(string(r))
	}
	return b.String()
}

// snakeize is the inverse: an exported Go method name to snake_case.
func snakeize(s string) string {
	var b []rune
	rs := []rune(s)
	for i, r := range rs {
		if unicode.IsUpper(r) {
			prevLower := i > 0 && unicode.IsLower(rs[i-1])
			nextLower := i+1 < len(rs) && unicode.IsLower(rs[i+1])
			if i > 0 && (prevLower || nextLower) {
				b = append(b, '_')
			}
			r = unicode.ToLower(r)
		}
		b = append(b, r)
	}
	return string(b)
}

// --- package-level convenience wrappers over the default Module ---

// OpenFont parses a font blob into a Font handle.
func OpenFont(ttf []byte) (*Font, error) { return mod.OpenFont(ttf) }

// Parse is an alias for OpenFont.
func Parse(ttf []byte) (*Font, error) { return mod.Parse(ttf) }

// MostLegible returns the bytes of the bundled Atkinson Hyperlegible family.
func MostLegible() []byte { return mod.MostLegible() }

// DefaultFont parses MostLegible into a Font handle.
func DefaultFont() (*Font, error) { return mod.DefaultFont() }

// Load returns the bytes of a bundled family by name (see Module.Load).
func Load(name string) []byte { return mod.Load(name) }

// Families lists every bundled family as an Array of Hashes.
func Families() []any { return mod.Families() }

// VisualOrder reorders text to visual order under base ("ltr"/"rtl"/"auto").
func VisualOrder(text, base string) string { return mod.VisualOrder(text, base) }

// ResolveLevels returns the bidi embedding levels of text as an Array.
func ResolveLevels(text, base string) []any { return mod.ResolveLevels(text, base) }

// Shape turns text into a positioned glyph run against face (see Module.Shape).
func Shape(face *Face, text string, opts map[string]any) []any {
	return mod.Shape(face, text, opts)
}
