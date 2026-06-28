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

package runtime

import (
	"reflect"
	"testing"
)

func TestBytesFromRoundTrip(t *testing.T) {
	in := sample()

	b := Bytes(in)
	if len(b) != in.Size() {
		t.Fatalf("Bytes len = %d, want %d", len(b), in.Size())
	}

	var out Record
	if err := From(&out, b); err != nil {
		t.Fatalf("From: %v", err)
	}

	if !reflect.DeepEqual(in, &out) {
		t.Fatalf("round-trip mismatch:\n in=%+v\nout=%+v", in, &out)
	}
}

func TestIntoReusesBufferWithCapacity(t *testing.T) {
	in := sample()
	buf := make([]byte, in.Size())

	got := Into(in, buf)
	if len(got) != in.Size() {
		t.Fatalf("Into len = %d, want %d", len(got), in.Size())
	}

	if &got[0] != &buf[0] {
		t.Error("Into allocated a new buffer despite sufficient capacity")
	}

	var out Record
	if err := From(&out, got); err != nil {
		t.Fatalf("From: %v", err)
	}

	if !reflect.DeepEqual(in, &out) {
		t.Fatalf("round-trip mismatch after Into reuse")
	}
}

func TestIntoGrowsWhenTooSmall(t *testing.T) {
	in := sample()

	got := Into(in, nil) // nil buffer forces allocation
	if len(got) != in.Size() {
		t.Fatalf("Into len = %d, want %d", len(got), in.Size())
	}

	var out Record
	if err := From(&out, got); err != nil {
		t.Fatalf("From: %v", err)
	}

	if !reflect.DeepEqual(in, &out) {
		t.Fatalf("round-trip mismatch after Into grow")
	}
}
