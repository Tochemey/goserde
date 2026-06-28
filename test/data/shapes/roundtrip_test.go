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

import (
	"reflect"
	"testing"
)

// rt marshals in, decodes into a fresh value, and asserts the result
// deep-equals the original, failing the test on any mismatch.
func rt[T any](t *testing.T, in T, marshal func([]byte) int, size func() int, unmarshal func(*T, []byte)) {
	buf := make([]byte, size())

	n := marshal(buf)
	if n != len(buf) {
		t.Fatalf("%T: Size=%d Marshal wrote %d", in, len(buf), n)
	}

	var out T
	unmarshal(&out, buf)

	if !reflect.DeepEqual(in, out) {
		t.Fatalf("%T mismatch:\n in=%+v\nout=%+v", in, in, out)
	}
}

func TestAllShapes(t *testing.T) {
	sf := SmallFixed{A: -5, B: 99, C: 2.71828, D: true}
	rt(t, sf, sf.Marshal, sf.Size, func(o *SmallFixed, b []byte) { o.Unmarshal(b) })

	fm := FlatMixed{ID: 1 << 40, Ratio: 0.5, Name: "flat", Data: []byte("xyz")}
	rt(t, fm, fm.Marshal, fm.Size, func(o *FlatMixed, b []byte) { o.Unmarshal(b) })

	in := Inner{X: 7, Y: 8}
	ns := Nested{Label: "n", Pos: Inner{1, 2}, Opt: &in, Path: []Inner{{3, 4}, {5, 6}}}
	rt(t, ns, ns.Marshal, ns.Size, func(o *Nested, b []byte) { o.Unmarshal(b) })

	// nil pointer case
	ns2 := Nested{Label: "nil", Pos: Inner{0, 0}, Opt: nil, Path: nil}
	rt(t, ns2, ns2.Marshal, ns2.Size, func(o *Nested, b []byte) { o.Unmarshal(b) })

	ch := CollectionHeavy{
		Counts: map[string]int64{"a": 1, "b": 2, "c": 3},
		Floats: []float64{1.1, 2.2, 3.3},
		Names:  []string{"x", "y"},
	}
	rt(t, ch, ch.Marshal, ch.Size, func(o *CollectionHeavy, b []byte) { o.Unmarshal(b) })

	sh := StringHeavy{Title: "T", Body: "lorem ipsum dolor", Tags: []string{"go", "fast"}}
	rt(t, sh, sh.Marshal, sh.Size, func(o *StringHeavy, b []byte) { o.Unmarshal(b) })
}
