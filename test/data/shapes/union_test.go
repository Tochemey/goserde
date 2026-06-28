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

	"github.com/tochemey/goserde/runtime"
)

// TestUnionRoundTrip checks that a tagged-union field and a slice of unions
// round-trip for each member variant and for the nil interface.
func TestUnionRoundTrip(t *testing.T) {
	cases := []Drawing{
		{Name: "circle", Shape: &Circle{R: 2.5}, Layers: []Geometry{&Circle{R: 1}, &Square{Side: 3}}},
		{Name: "square", Shape: &Square{Side: 4}, Layers: []Geometry{&Square{Side: 9}}},
		{Name: "nil", Shape: nil, Layers: nil},
	}

	for _, in := range cases {
		buf := make([]byte, in.Size())

		n := in.Marshal(buf)
		if n != len(buf) {
			t.Fatalf("%s: Marshal wrote %d bytes, Size reported %d", in.Name, n, len(buf))
		}

		var out Drawing

		m, err := out.Unmarshal(buf)
		if err != nil {
			t.Fatalf("%s: Unmarshal: %v", in.Name, err)
		}

		if m != len(buf) {
			t.Fatalf("%s: Unmarshal consumed %d bytes, want %d", in.Name, m, len(buf))
		}

		if !reflect.DeepEqual(in, out) {
			t.Fatalf("%s: round-trip mismatch:\n in=%+v\nout=%+v", in.Name, in, out)
		}
	}
}

// TestUnionUnknownTag verifies that decoding a discriminator with no matching
// member returns ErrUnknownUnionTag instead of panicking — the forward-compat
// path for a peer that encoded a newer member.
func TestUnionUnknownTag(t *testing.T) {
	d := Drawing{Name: "x", Shape: &Circle{R: 1}}
	buf := make([]byte, d.Size())
	d.Marshal(buf)

	// Name "x" encodes as a 1-byte length prefix + 1 byte, so the Shape tag sits
	// at index 2. Overwrite it with an undeclared member ID.
	buf[2] = 99

	var out Drawing
	if _, err := out.Unmarshal(buf); err != runtime.ErrUnknownUnionTag {
		t.Fatalf("unknown tag: got err=%v, want runtime.ErrUnknownUnionTag", err)
	}
}
