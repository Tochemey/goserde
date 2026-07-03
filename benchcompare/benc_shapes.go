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
	bstd "github.com/deneonet/benc/std"

	"github.com/tochemey/goserde/test/data/shapes"
)

// benc codecs for the shapes package: fixed-width numerics plus unsafe string
// decoding, matching the settings the Record head-to-head uses.

// --- SmallFixed ---

func bencSmallFixedSize(*shapes.SmallFixed) int {
	return bstd.SizeInt32() + bstd.SizeInt32() + bstd.SizeFloat64() + bstd.SizeBool()
}

func bencSmallFixedMarshal(v *shapes.SmallFixed, b []byte) int {
	n := bstd.MarshalInt32(0, b, v.A)
	n = bstd.MarshalInt32(n, b, v.B)
	n = bstd.MarshalFloat64(n, b, v.C)
	n = bstd.MarshalBool(n, b, v.D)
	return n
}

func bencSmallFixedUnmarshal(b []byte) (v shapes.SmallFixed, err error) {
	n, a, err := bstd.UnmarshalInt32(0, b)
	if err != nil {
		return
	}

	v.A = a
	n, v.B, err = bstd.UnmarshalInt32(n, b)
	if err != nil {
		return
	}

	n, v.C, err = bstd.UnmarshalFloat64(n, b)
	if err != nil {
		return
	}

	_, v.D, err = bstd.UnmarshalBool(n, b)
	return
}

// --- FlatMixed ---

func bencFlatMixedSize(v *shapes.FlatMixed) int {
	return bstd.SizeUint64() + bstd.SizeFloat64() + bstd.SizeString(v.Name) + bstd.SizeBytes(v.Data)
}

func bencFlatMixedMarshal(v *shapes.FlatMixed, b []byte) int {
	n := bstd.MarshalUint64(0, b, v.ID)
	n = bstd.MarshalFloat64(n, b, v.Ratio)
	n = bstd.MarshalString(n, b, v.Name)
	n = bstd.MarshalBytes(n, b, v.Data)
	return n
}

func bencFlatMixedUnmarshal(b []byte) (v shapes.FlatMixed, err error) {
	n, id, err := bstd.UnmarshalUint64(0, b)
	if err != nil {
		return
	}

	v.ID = id
	n, v.Ratio, err = bstd.UnmarshalFloat64(n, b)
	if err != nil {
		return
	}

	n, v.Name, err = bstd.UnmarshalUnsafeString(n, b)
	if err != nil {
		return
	}

	_, v.Data, err = bstd.UnmarshalBytesCropped(n, b)
	return
}

// --- Nested (Inner, *Inner, []Inner) ---

func bencInnerSize() int { return bstd.SizeInt32() + bstd.SizeInt32() }

func bencInnerMarshal(n int, b []byte, v shapes.Inner) int {
	n = bstd.MarshalInt32(n, b, v.X)
	return bstd.MarshalInt32(n, b, v.Y)
}

func bencInnerUnmarshal(n int, b []byte) (int, shapes.Inner, error) {
	var v shapes.Inner

	n, x, err := bstd.UnmarshalInt32(n, b)
	if err != nil {
		return n, v, err
	}

	v.X = x
	n, v.Y, err = bstd.UnmarshalInt32(n, b)
	return n, v, err
}

func bencNestedSize(v *shapes.Nested) int {
	s := bstd.SizeString(v.Label) + bencInnerSize()
	s += bstd.SizeBool() // presence flag for Opt (benc has no pointer type)

	if v.Opt != nil {
		s += bencInnerSize()
	}

	return s + bstd.SizeFixedSlice(v.Path, bencInnerSize())
}

func bencNestedMarshal(v *shapes.Nested, b []byte) int {
	n := bstd.MarshalString(0, b, v.Label)
	n = bencInnerMarshal(n, b, v.Pos)
	n = bstd.MarshalBool(n, b, v.Opt != nil)

	if v.Opt != nil {
		n = bencInnerMarshal(n, b, *v.Opt)
	}

	return bstd.MarshalSlice(n, b, v.Path, bencInnerMarshal)
}

func bencNestedUnmarshal(b []byte) (v shapes.Nested, err error) {
	n, label, err := bstd.UnmarshalUnsafeString(0, b)
	if err != nil {
		return
	}

	v.Label = label
	n, v.Pos, err = bencInnerUnmarshal(n, b)
	if err != nil {
		return
	}

	var present bool

	n, present, err = bstd.UnmarshalBool(n, b)
	if err != nil {
		return
	}

	if present {
		var in shapes.Inner

		n, in, err = bencInnerUnmarshal(n, b)
		if err != nil {
			return
		}

		v.Opt = &in
	}

	_, v.Path, err = bstd.UnmarshalSlice[shapes.Inner](n, b, bencInnerUnmarshal)
	return
}

// --- CollectionHeavy (map + slices) ---

func bencCollectionSize(v *shapes.CollectionHeavy) int {
	s := bstd.SizeMap(v.Counts, bstd.SizeString, bstd.SizeInt64)
	s += bstd.SizeFixedSlice(v.Floats, bstd.SizeFloat64())
	s += bstd.SizeSlice(v.Names, bstd.SizeString)
	return s
}

func bencCollectionMarshal(v *shapes.CollectionHeavy, b []byte) int {
	n := bstd.MarshalMap(0, b, v.Counts, bstd.MarshalString, bstd.MarshalInt64)
	n = bstd.MarshalSlice(n, b, v.Floats, bstd.MarshalFloat64)
	n = bstd.MarshalSlice(n, b, v.Names, bstd.MarshalString)
	return n
}

func bencCollectionUnmarshal(b []byte) (v shapes.CollectionHeavy, err error) {
	n, counts, err := bstd.UnmarshalMap[string, int64](0, b, bstd.UnmarshalUnsafeString, bstd.UnmarshalInt64)
	if err != nil {
		return
	}

	v.Counts = counts
	n, v.Floats, err = bstd.UnmarshalSlice[float64](n, b, bstd.UnmarshalFloat64)
	if err != nil {
		return
	}

	_, v.Names, err = bstd.UnmarshalSlice[string](n, b, bstd.UnmarshalUnsafeString)
	return
}

// --- StringHeavy ---

func bencStringHeavySize(v *shapes.StringHeavy) int {
	return bstd.SizeString(v.Title) + bstd.SizeString(v.Body) + bstd.SizeSlice(v.Tags, bstd.SizeString)
}

func bencStringHeavyMarshal(v *shapes.StringHeavy, b []byte) int {
	n := bstd.MarshalString(0, b, v.Title)
	n = bstd.MarshalString(n, b, v.Body)
	n = bstd.MarshalSlice(n, b, v.Tags, bstd.MarshalString)
	return n
}

func bencStringHeavyUnmarshal(b []byte) (v shapes.StringHeavy, err error) {
	n, title, err := bstd.UnmarshalUnsafeString(0, b)
	if err != nil {
		return
	}

	v.Title = title
	n, v.Body, err = bstd.UnmarshalUnsafeString(n, b)
	if err != nil {
		return
	}

	_, v.Tags, err = bstd.UnmarshalSlice[string](n, b, bstd.UnmarshalUnsafeString)
	return
}
