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

package benchcompare

import (
	"github.com/mus-format/mus-go/ord"
	"github.com/mus-format/mus-go/unsafe"

	"github.com/tochemey/goserde/test/data/shapes"
)

// mus codecs for the shapes package, using mus's fastest (unsafe) primitives
// throughout so each library competes at its best.

// --- SmallFixed ---

func musSmallFixedSize(v *shapes.SmallFixed) int {
	return unsafe.Int32.Size(v.A) + unsafe.Int32.Size(v.B) +
		unsafe.Float64.Size(v.C) + unsafe.Bool.Size(v.D)
}

func musSmallFixedMarshal(v *shapes.SmallFixed, b []byte) int {
	n := unsafe.Int32.Marshal(v.A, b)
	n += unsafe.Int32.Marshal(v.B, b[n:])
	n += unsafe.Float64.Marshal(v.C, b[n:])
	n += unsafe.Bool.Marshal(v.D, b[n:])
	return n
}

func musSmallFixedUnmarshal(b []byte) (v shapes.SmallFixed, n int, err error) {
	v.A, n, err = unsafe.Int32.Unmarshal(b)
	var c int
	v.B, c, err = unsafe.Int32.Unmarshal(b[n:])
	n += c
	v.C, c, err = unsafe.Float64.Unmarshal(b[n:])
	n += c
	v.D, c, err = unsafe.Bool.Unmarshal(b[n:])
	n += c
	return
}

// --- FlatMixed ---

func musFlatMixedSize(v *shapes.FlatMixed) int {
	return unsafe.Uint64.Size(v.ID) + unsafe.Float64.Size(v.Ratio) +
		unsafe.String.Size(v.Name) + unsafe.ByteSlice.Size(v.Data)
}

func musFlatMixedMarshal(v *shapes.FlatMixed, b []byte) int {
	n := unsafe.Uint64.Marshal(v.ID, b)
	n += unsafe.Float64.Marshal(v.Ratio, b[n:])
	n += unsafe.String.Marshal(v.Name, b[n:])
	n += unsafe.ByteSlice.Marshal(v.Data, b[n:])
	return n
}

func musFlatMixedUnmarshal(b []byte) (v shapes.FlatMixed, n int, err error) {
	v.ID, n, err = unsafe.Uint64.Unmarshal(b)
	var c int
	v.Ratio, c, err = unsafe.Float64.Unmarshal(b[n:])
	n += c
	v.Name, c, err = unsafe.String.Unmarshal(b[n:])
	n += c
	v.Data, c, err = unsafe.ByteSlice.Unmarshal(b[n:])
	n += c
	return
}

// --- Nested (Inner, *Inner, []Inner) ---

// innerSer is a mus.Serializer for shapes.Inner, needed by the pointer and
// slice combinators.
type innerSer struct{}

func (innerSer) Marshal(v shapes.Inner, b []byte) int {
	n := unsafe.Int32.Marshal(v.X, b)
	return n + unsafe.Int32.Marshal(v.Y, b[n:])
}

func (innerSer) Unmarshal(b []byte) (v shapes.Inner, n int, err error) {
	v.X, n, err = unsafe.Int32.Unmarshal(b)
	var c int
	v.Y, c, err = unsafe.Int32.Unmarshal(b[n:])
	n += c
	return
}

func (innerSer) Size(v shapes.Inner) int {
	return unsafe.Int32.Size(v.X) + unsafe.Int32.Size(v.Y)
}

func (innerSer) Skip(b []byte) (n int, err error) {
	n, err = unsafe.Int32.Skip(b)
	if err != nil {
		return
	}

	var c int
	c, err = unsafe.Int32.Skip(b[n:])
	return n + c, err
}

var (
	musInnerPtrSer   = ord.NewPtrSer[shapes.Inner](innerSer{})
	musInnerSliceSer = ord.NewSliceSer[shapes.Inner](innerSer{})
)

func musNestedSize(v *shapes.Nested) int {
	return unsafe.String.Size(v.Label) + innerSer{}.Size(v.Pos) +
		musInnerPtrSer.Size(v.Opt) + musInnerSliceSer.Size(v.Path)
}

func musNestedMarshal(v *shapes.Nested, b []byte) int {
	n := unsafe.String.Marshal(v.Label, b)
	n += innerSer{}.Marshal(v.Pos, b[n:])
	n += musInnerPtrSer.Marshal(v.Opt, b[n:])
	n += musInnerSliceSer.Marshal(v.Path, b[n:])
	return n
}

func musNestedUnmarshal(b []byte) (v shapes.Nested, n int, err error) {
	v.Label, n, err = unsafe.String.Unmarshal(b)
	var c int
	v.Pos, c, err = innerSer{}.Unmarshal(b[n:])
	n += c
	v.Opt, c, err = musInnerPtrSer.Unmarshal(b[n:])
	n += c
	v.Path, c, err = musInnerSliceSer.Unmarshal(b[n:])
	n += c
	return
}

// --- CollectionHeavy (map + slices) ---

var (
	musCountsSer = ord.NewMapSer[string, int64](unsafe.String, unsafe.Int64)
	musFloatsSer = ord.NewSliceSer[float64](unsafe.Float64)
	musNamesSer  = ord.NewSliceSer[string](unsafe.String)
)

func musCollectionSize(v *shapes.CollectionHeavy) int {
	return musCountsSer.Size(v.Counts) + musFloatsSer.Size(v.Floats) +
		musNamesSer.Size(v.Names)
}

func musCollectionMarshal(v *shapes.CollectionHeavy, b []byte) int {
	n := musCountsSer.Marshal(v.Counts, b)
	n += musFloatsSer.Marshal(v.Floats, b[n:])
	n += musNamesSer.Marshal(v.Names, b[n:])
	return n
}

func musCollectionUnmarshal(b []byte) (v shapes.CollectionHeavy, n int, err error) {
	v.Counts, n, err = musCountsSer.Unmarshal(b)
	var c int
	v.Floats, c, err = musFloatsSer.Unmarshal(b[n:])
	n += c
	v.Names, c, err = musNamesSer.Unmarshal(b[n:])
	n += c
	return
}

// --- StringHeavy ---

func musStringHeavySize(v *shapes.StringHeavy) int {
	return unsafe.String.Size(v.Title) + unsafe.String.Size(v.Body) +
		musNamesSer.Size(v.Tags)
}

func musStringHeavyMarshal(v *shapes.StringHeavy, b []byte) int {
	n := unsafe.String.Marshal(v.Title, b)
	n += unsafe.String.Marshal(v.Body, b[n:])
	n += musNamesSer.Marshal(v.Tags, b[n:])
	return n
}

func musStringHeavyUnmarshal(b []byte) (v shapes.StringHeavy, n int, err error) {
	v.Title, n, err = unsafe.String.Unmarshal(b)
	var c int
	v.Body, c, err = unsafe.String.Unmarshal(b[n:])
	n += c
	v.Tags, c, err = musNamesSer.Unmarshal(b[n:])
	n += c
	return
}
