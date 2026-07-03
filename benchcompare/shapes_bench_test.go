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
	"reflect"
	"testing"

	"github.com/tochemey/goserde/test/data/shapes"
)

// Head-to-head across the six benchmark shapes (Record above plus the five
// generated shapes). The goserde side runs the actual goserdegen output from
// test/data/shapes; mus and benc run the hand-written codecs in mus_shapes.go
// and benc_shapes.go, each using its library's fastest settings. Sample values
// mirror the make bench-shapes fixtures, so numbers line up across suites.

func sampleSmallFixed() shapes.SmallFixed {
	return shapes.SmallFixed{A: -5, B: 99, C: 2.71828, D: true}
}

func sampleFlatMixed() shapes.FlatMixed {
	return shapes.FlatMixed{ID: 1, Ratio: 2, Name: "hello world", Data: []byte("0123456789")}
}

func sampleNested() shapes.Nested {
	in := shapes.Inner{X: 7, Y: 8}

	return shapes.Nested{
		Label: "n",
		Pos:   shapes.Inner{X: 1, Y: 2},
		Opt:   &in,
		Path:  []shapes.Inner{{X: 3, Y: 4}, {X: 5, Y: 6}, {X: 7, Y: 8}},
	}
}

func sampleCollection() shapes.CollectionHeavy {
	return shapes.CollectionHeavy{
		Counts: map[string]int64{"a": 1, "b": 2, "c": 3, "d": 4},
		Floats: []float64{1, 2, 3, 4, 5},
		Names:  []string{"x", "y", "z"},
	}
}

func sampleStringHeavy() shapes.StringHeavy {
	return shapes.StringHeavy{Title: "Title", Body: "lorem ipsum dolor sit amet consectetur", Tags: []string{"go", "fast", "serde"}}
}

// TestShapesRoundTrip keeps every hand-written competitor codec honest: each
// library must round-trip the same sample values it is benchmarked on.
func TestShapesRoundTrip(t *testing.T) {
	sf := sampleSmallFixed()
	mb := make([]byte, musSmallFixedSize(&sf))
	musSmallFixedMarshal(&sf, mb)

	if got, _, err := musSmallFixedUnmarshal(mb); err != nil || got != sf {
		t.Fatalf("mus SmallFixed: got %+v err %v", got, err)
	}

	bb := make([]byte, bencSmallFixedSize(&sf))
	bencSmallFixedMarshal(&sf, bb)

	if got, err := bencSmallFixedUnmarshal(bb); err != nil || got != sf {
		t.Fatalf("benc SmallFixed: got %+v err %v", got, err)
	}

	fm := sampleFlatMixed()
	mb = make([]byte, musFlatMixedSize(&fm))
	musFlatMixedMarshal(&fm, mb)

	if got, _, err := musFlatMixedUnmarshal(mb); err != nil || !reflect.DeepEqual(got, fm) {
		t.Fatalf("mus FlatMixed: got %+v err %v", got, err)
	}

	bb = make([]byte, bencFlatMixedSize(&fm))
	bencFlatMixedMarshal(&fm, bb)

	if got, err := bencFlatMixedUnmarshal(bb); err != nil || !reflect.DeepEqual(got, fm) {
		t.Fatalf("benc FlatMixed: got %+v err %v", got, err)
	}

	ns := sampleNested()
	mb = make([]byte, musNestedSize(&ns))
	musNestedMarshal(&ns, mb)

	if got, _, err := musNestedUnmarshal(mb); err != nil || !reflect.DeepEqual(got, ns) {
		t.Fatalf("mus Nested: got %+v err %v", got, err)
	}

	bb = make([]byte, bencNestedSize(&ns))
	bencNestedMarshal(&ns, bb)

	if got, err := bencNestedUnmarshal(bb); err != nil || !reflect.DeepEqual(got, ns) {
		t.Fatalf("benc Nested: got %+v err %v", got, err)
	}

	ch := sampleCollection()
	mb = make([]byte, musCollectionSize(&ch))
	musCollectionMarshal(&ch, mb)

	if got, _, err := musCollectionUnmarshal(mb); err != nil || !reflect.DeepEqual(got, ch) {
		t.Fatalf("mus Collection: got %+v err %v", got, err)
	}

	bb = make([]byte, bencCollectionSize(&ch))
	bencCollectionMarshal(&ch, bb)

	if got, err := bencCollectionUnmarshal(bb); err != nil || !reflect.DeepEqual(got, ch) {
		t.Fatalf("benc Collection: got %+v err %v", got, err)
	}

	sh := sampleStringHeavy()
	mb = make([]byte, musStringHeavySize(&sh))
	musStringHeavyMarshal(&sh, mb)

	if got, _, err := musStringHeavyUnmarshal(mb); err != nil || !reflect.DeepEqual(got, sh) {
		t.Fatalf("mus StringHeavy: got %+v err %v", got, err)
	}

	bb = make([]byte, bencStringHeavySize(&sh))
	bencStringHeavyMarshal(&sh, bb)

	if got, err := bencStringHeavyUnmarshal(bb); err != nil || !reflect.DeepEqual(got, sh) {
		t.Fatalf("benc StringHeavy: got %+v err %v", got, err)
	}

	t.Logf("payload bytes (goserde/mus/benc): SmallFixed %d/%d/%d FlatMixed %d/%d/%d Nested %d/%d/%d Collection %d/%d/%d StringHeavy %d/%d/%d",
		sf.Size(), musSmallFixedSize(&sf), bencSmallFixedSize(&sf),
		fm.Size(), musFlatMixedSize(&fm), bencFlatMixedSize(&fm),
		ns.Size(), musNestedSize(&ns), bencNestedSize(&ns),
		ch.Size(), musCollectionSize(&ch), bencCollectionSize(&ch),
		sh.Size(), musStringHeavySize(&sh), bencStringHeavySize(&sh))
}

// --- SmallFixed ---

func BenchmarkSmallFixedGoserde_M(b *testing.B) {
	v := sampleSmallFixed()
	buf := make([]byte, v.Size())
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = v.Marshal(buf)
	}
}

func BenchmarkSmallFixedGoserde_U(b *testing.B) {
	v := sampleSmallFixed()
	buf := make([]byte, v.Size())
	v.Marshal(buf)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var o shapes.SmallFixed
		o.Unmarshal(buf)
	}
}

func BenchmarkSmallFixedMus_M(b *testing.B) {
	v := sampleSmallFixed()
	buf := make([]byte, musSmallFixedSize(&v))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = musSmallFixedMarshal(&v, buf)
	}
}

func BenchmarkSmallFixedMus_U(b *testing.B) {
	v := sampleSmallFixed()
	buf := make([]byte, musSmallFixedSize(&v))
	musSmallFixedMarshal(&v, buf)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = musSmallFixedUnmarshal(buf)
	}
}

func BenchmarkSmallFixedBenc_M(b *testing.B) {
	v := sampleSmallFixed()
	buf := make([]byte, bencSmallFixedSize(&v))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = bencSmallFixedMarshal(&v, buf)
	}
}

func BenchmarkSmallFixedBenc_U(b *testing.B) {
	v := sampleSmallFixed()
	buf := make([]byte, bencSmallFixedSize(&v))
	bencSmallFixedMarshal(&v, buf)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = bencSmallFixedUnmarshal(buf)
	}
}

// --- FlatMixed ---

func BenchmarkFlatMixedGoserde_M(b *testing.B) {
	v := sampleFlatMixed()
	buf := make([]byte, v.Size())
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = v.Marshal(buf)
	}
}

func BenchmarkFlatMixedGoserde_U(b *testing.B) {
	v := sampleFlatMixed()
	buf := make([]byte, v.Size())
	v.Marshal(buf)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var o shapes.FlatMixed
		o.Unmarshal(buf)
	}
}

func BenchmarkFlatMixedMus_M(b *testing.B) {
	v := sampleFlatMixed()
	buf := make([]byte, musFlatMixedSize(&v))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = musFlatMixedMarshal(&v, buf)
	}
}

func BenchmarkFlatMixedMus_U(b *testing.B) {
	v := sampleFlatMixed()
	buf := make([]byte, musFlatMixedSize(&v))
	musFlatMixedMarshal(&v, buf)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = musFlatMixedUnmarshal(buf)
	}
}

func BenchmarkFlatMixedBenc_M(b *testing.B) {
	v := sampleFlatMixed()
	buf := make([]byte, bencFlatMixedSize(&v))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = bencFlatMixedMarshal(&v, buf)
	}
}

func BenchmarkFlatMixedBenc_U(b *testing.B) {
	v := sampleFlatMixed()
	buf := make([]byte, bencFlatMixedSize(&v))
	bencFlatMixedMarshal(&v, buf)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = bencFlatMixedUnmarshal(buf)
	}
}

// --- Nested ---

func BenchmarkNestedGoserde_M(b *testing.B) {
	v := sampleNested()
	buf := make([]byte, v.Size())
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = v.Marshal(buf)
	}
}

func BenchmarkNestedGoserde_U(b *testing.B) {
	v := sampleNested()
	buf := make([]byte, v.Size())
	v.Marshal(buf)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var o shapes.Nested
		o.Unmarshal(buf)
	}
}

func BenchmarkNestedMus_M(b *testing.B) {
	v := sampleNested()
	buf := make([]byte, musNestedSize(&v))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = musNestedMarshal(&v, buf)
	}
}

func BenchmarkNestedMus_U(b *testing.B) {
	v := sampleNested()
	buf := make([]byte, musNestedSize(&v))
	musNestedMarshal(&v, buf)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = musNestedUnmarshal(buf)
	}
}

func BenchmarkNestedBenc_M(b *testing.B) {
	v := sampleNested()
	buf := make([]byte, bencNestedSize(&v))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = bencNestedMarshal(&v, buf)
	}
}

func BenchmarkNestedBenc_U(b *testing.B) {
	v := sampleNested()
	buf := make([]byte, bencNestedSize(&v))
	bencNestedMarshal(&v, buf)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = bencNestedUnmarshal(buf)
	}
}

// --- CollectionHeavy ---

func BenchmarkCollectionGoserde_M(b *testing.B) {
	v := sampleCollection()
	buf := make([]byte, v.Size())
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = v.Marshal(buf)
	}
}

func BenchmarkCollectionGoserde_U(b *testing.B) {
	v := sampleCollection()
	buf := make([]byte, v.Size())
	v.Marshal(buf)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var o shapes.CollectionHeavy
		o.Unmarshal(buf)
	}
}

func BenchmarkCollectionMus_M(b *testing.B) {
	v := sampleCollection()
	buf := make([]byte, musCollectionSize(&v))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = musCollectionMarshal(&v, buf)
	}
}

func BenchmarkCollectionMus_U(b *testing.B) {
	v := sampleCollection()
	buf := make([]byte, musCollectionSize(&v))
	musCollectionMarshal(&v, buf)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = musCollectionUnmarshal(buf)
	}
}

func BenchmarkCollectionBenc_M(b *testing.B) {
	v := sampleCollection()
	buf := make([]byte, bencCollectionSize(&v))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = bencCollectionMarshal(&v, buf)
	}
}

func BenchmarkCollectionBenc_U(b *testing.B) {
	v := sampleCollection()
	buf := make([]byte, bencCollectionSize(&v))
	bencCollectionMarshal(&v, buf)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = bencCollectionUnmarshal(buf)
	}
}

// --- StringHeavy ---

func BenchmarkStringHeavyGoserde_M(b *testing.B) {
	v := sampleStringHeavy()
	buf := make([]byte, v.Size())
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = v.Marshal(buf)
	}
}

func BenchmarkStringHeavyGoserde_U(b *testing.B) {
	v := sampleStringHeavy()
	buf := make([]byte, v.Size())
	v.Marshal(buf)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var o shapes.StringHeavy
		o.Unmarshal(buf)
	}
}

func BenchmarkStringHeavyMus_M(b *testing.B) {
	v := sampleStringHeavy()
	buf := make([]byte, musStringHeavySize(&v))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = musStringHeavyMarshal(&v, buf)
	}
}

func BenchmarkStringHeavyMus_U(b *testing.B) {
	v := sampleStringHeavy()
	buf := make([]byte, musStringHeavySize(&v))
	musStringHeavyMarshal(&v, buf)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = musStringHeavyUnmarshal(buf)
	}
}

func BenchmarkStringHeavyBenc_M(b *testing.B) {
	v := sampleStringHeavy()
	buf := make([]byte, bencStringHeavySize(&v))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = bencStringHeavyMarshal(&v, buf)
	}
}

func BenchmarkStringHeavyBenc_U(b *testing.B) {
	v := sampleStringHeavy()
	buf := make([]byte, bencStringHeavySize(&v))
	bencStringHeavyMarshal(&v, buf)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = bencStringHeavyUnmarshal(buf)
	}
}
