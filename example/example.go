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

// Package example is a standalone, illustrative use of goserde: it shows how to
// annotate structs for code generation and what the generator produces. It is
// kept out of the test suite and coverage on purpose.
//
// Annotate a struct with //goserde:generate and run `go generate ./...` to
// produce its Size/Marshal/Unmarshal methods. Encode and decode with the
// generated methods directly (the allocation-free hot path) or with the
// codec.Bytes / codec.Into / codec.From convenience helpers:
//
//	u := &example.User{ID: 1, Name: "ada", Scores: []int32{10, -3, 42}}
//
//	buf := make([]byte, u.Size())
//	u.Marshal(buf)
//
//	var out example.User
//	if _, err := out.Unmarshal(buf); err != nil {
//		// handle truncated input (safe mode) ...
//	}
package example

//go:generate go run ../cmd/goserdegen -dir . -out goserde_gen.go

// User mixes fixed-width fields with variable-length strings and slices.
//
//goserde:generate
type User struct {
	ID       uint64
	Age      uint16
	Height   float32
	Verified bool
	Name     string
	Nicks    []string
	Scores   []int32
	Avatar   []byte
}

// Point is all-fixed-width, so its generated codec takes the blittable memmove
// fast path.
//
//goserde:generate
type Point struct {
	X int32
	Y int32
	Z int32
}
