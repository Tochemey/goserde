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

package example

import (
	"encoding/json"
	"reflect"
	"testing"
)

// sampleUser returns a User populated across every field kind for round-trip
// and benchmark tests.
func sampleUser() *User {
	return &User{
		ID: 0x1122334455667788, Age: 31, Height: 1.83, Verified: true,
		Name:   "generated-codec-user",
		Nicks:  []string{"neo", "trinity", "morpheus"},
		Scores: []int32{-5, 0, 42, 1000, -999},
		Avatar: []byte("PNGDATA....binary blob here...."),
	}
}

func TestGenUserRoundTrip(t *testing.T) {
	in := sampleUser()
	buf := make([]byte, in.Size())

	n := in.Marshal(buf)
	if n != len(buf) {
		t.Fatalf("Size=%d Marshal wrote %d", len(buf), n)
	}

	var out User
	if _, err := out.Unmarshal(buf); err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(in, &out) {
		t.Fatalf("mismatch\n in=%+v\nout=%+v", in, &out)
	}
}

func TestGenPointRoundTrip(t *testing.T) {
	in := &Point{X: -7, Y: 13, Z: 2147483647}
	buf := make([]byte, in.Size())
	in.Marshal(buf)

	var out Point
	out.Unmarshal(buf)

	if in.X != out.X || in.Y != out.Y || in.Z != out.Z {
		t.Fatalf("mismatch %+v %+v", in, &out)
	}
}

func BenchmarkGenUserMarshal(b *testing.B) {
	in := sampleUser()
	buf := make([]byte, in.Size())
	b.ReportAllocs()
	for b.Loop() {
		_ = in.Marshal(buf)
	}
}

func BenchmarkGenUserUnmarshal(b *testing.B) {
	in := sampleUser()
	buf := make([]byte, in.Size())
	in.Marshal(buf)
	b.ReportAllocs()
	for b.Loop() {
		var out User
		out.Unmarshal(buf)
	}
}

func BenchmarkGenUserJSONMarshal(b *testing.B) {
	in := sampleUser()
	b.ReportAllocs()
	for b.Loop() {
		_, _ = json.Marshal(in)
	}
}

func BenchmarkGenUserJSONUnmarshal(b *testing.B) {
	in := sampleUser()
	data, _ := json.Marshal(in)
	b.ReportAllocs()
	for b.Loop() {
		var out User
		_ = json.Unmarshal(data, &out)
	}
}
