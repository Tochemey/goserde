// MIT License
//
// Copyright (c) 2026 Arsene Tochemey Gandote
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package shapes

import "time"

//go:generate go run ../../../cmd/goserdegen -dir . -out goserde_gen.go

// This package is a spread of struct shapes mirroring what the standard
// serializer suites exercise, used for round-trip tests and benchmarks.

// SmallFixed is all fixed-width, so its codec takes the blittable memmove path.
//
//goserde:generate
type SmallFixed struct {
	A int32   // fixed-width
	B int32   // fixed-width
	C float64 // fixed-width IEEE 754
	D bool    // single-byte flag
}

// FlatMixed combines fixed-width fields with a string and a byte slice.
//
//goserde:generate
type FlatMixed struct {
	ID    uint64  // fixed-width identifier
	Ratio float64 // fixed-width IEEE 754
	Name  string  // length-prefixed bytes, decoded zero-copy
	Data  []byte  // length-prefixed raw bytes, decoded zero-copy
}

// Inner is a small nested target used as a field and slice/array element by the
// other shapes; it must itself be annotated so a codec is generated for it.
//
//goserde:generate
type Inner struct {
	X int32 // fixed-width
	Y int32 // fixed-width
}

// Nested exercises a nested struct value, a pointer to a struct, and a
// slice of structs.
//
//goserde:generate
type Nested struct {
	Label string  // length-prefixed bytes
	Pos   Inner   // nested struct, delegates to Inner's codec
	Opt   *Inner  // optional nested struct (1-byte nil flag)
	Path  []Inner // length-prefixed slice of nested structs
}

// CollectionHeavy is map- and slice-heavy; map decode is the codec's weak spot.
//
//goserde:generate
type CollectionHeavy struct {
	Counts map[string]int64 // length-prefixed map of string keys to fixed values
	Floats []float64        // length-prefixed slice of fixed-width elements
	Names  []string         // length-prefixed slice of length-prefixed strings
}

// StringHeavy is dominated by variable-length string data.
//
//goserde:generate
type StringHeavy struct {
	Title string   // length-prefixed bytes
	Body  string   // length-prefixed bytes
	Tags  []string // length-prefixed slice of length-prefixed strings
}

// TimeStruct exercises time.Time as a direct field, a pointer, and a slice
// element. time.Time serializes as int64 unix-nanoseconds (monotonic clock and
// location are not preserved); values must fall in ~[1678, 2262] to round-trip.
//
//goserde:generate
type TimeStruct struct {
	Label   string      // length-prefixed bytes
	Created time.Time   // 8-byte unix-nanoseconds
	Updated *time.Time  // optional time (1-byte nil flag)
	Stamps  []time.Time // length-prefixed slice of times
}

// FixedArrays is all-fixed-width (arrays of fixed-width basics), so it must take
// the blittable memmove fast path.
//
//goserde:generate
type FixedArrays struct {
	Hash [16]byte // fixed array, bulk-copied
	Quad [4]int32 // fixed array of fixed-width elements
	Flag bool     // single-byte flag
}

// MixedArrays exercises arrays whose elements aren't bulk-copyable (strings,
// nested structs) plus a byte array in a non-blittable struct.
//
//goserde:generate
type MixedArrays struct {
	Name   string    // length-prefixed bytes
	Words  [3]string // fixed-count array of length-prefixed strings
	Points [2]Inner  // fixed-count array of nested structs
	Bytes  [8]byte   // fixed array, bulk-copied
}

// Tagged exercises field exclusion via the `goserde:"-"` struct tag. Skip is an
// unsupported type, proving exclusion happens before the type is type-checked.
//
//goserde:generate
type Tagged struct {
	ID     int64    // serialized
	Secret string   `goserde:"-"` // excluded from the wire
	Skip   chan int `goserde:"-"` // excluded; its type isn't serializable at all
	Name   string   // serialized
}

// Hue is a defined type with a uint8 underlying. As a slice or array element it
// must not take the raw []byte copy path, since []byte is not assignable to
// []Hue.
type Hue uint8

// Grade is a defined type with an int32 underlying, exercised as a scalar field
// and as a slice element.
type Grade int32

// NamedScalars exercises named (defined) types whose underlying is a fixed-width
// basic, used as slice and array elements. The leading string keeps the struct
// off the blittable path, forcing the element-wise codecs that the raw byte
// copy fast path would otherwise mis-handle.
//
//goserde:generate
type NamedScalars struct {
	Label   string  // length-prefixed bytes; forces a non-blittable struct
	Palette []Hue   // named uint8 element: element-wise, not a []byte copy
	Swatch  [4]Hue  // named uint8 array element: element-wise, not a bulk copy
	Levels  []Grade // named int32 element
	Accent  *Hue    // pointer to a named uint8
}

// NamedFixed is all-fixed-width even though its fields use defined types, so it
// must still take the blittable memmove path; it guards that named byte and
// integer elements round-trip through the raw copy.
//
//goserde:generate
type NamedFixed struct {
	Tag   Hue    // named uint8 scalar
	Codes [3]Hue // named uint8 array
	Score Grade  // named int32 scalar
	Flag  bool   // single-byte flag
}

// Circle is a Geometry union member.
//
//goserde:generate
type Circle struct {
	R float64 // radius
}

func (*Circle) isGeometry() {}

// Square is a Geometry union member.
//
//goserde:generate
type Square struct {
	Side float64 // edge length
}

func (*Square) isGeometry() {}

// Geometry is a tagged union over Circle and Square. Member order fixes the
// on-wire tag IDs (Circle = 1, Square = 2), so members are only ever appended.
//
//goserde:union Circle Square
type Geometry interface {
	isGeometry()
}

// Drawing exercises a tagged-union field and a slice of unions.
//
//goserde:generate
type Drawing struct {
	Name   string     // length-prefixed bytes
	Shape  Geometry   // union field (tag + member, or nil)
	Layers []Geometry // slice of unions
}
