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

//go:generate go run ../../../cmd/goserdegen -dir . -out goserde_gen.go

// User is an annotated fixture mixing fixed-width fields with variable-length
// strings and slices; goserdegen generates its codec.
//
//goserde:generate
type User struct {
	ID       uint64   // fixed-width identifier
	Age      uint16   // fixed-width
	Height   float32  // fixed-width IEEE 754
	Verified bool     // single-byte flag
	Name     string   // length-prefixed bytes, decoded zero-copy
	Nicks    []string // length-prefixed slice of length-prefixed strings
	Scores   []int32  // length-prefixed slice of fixed-width elements
	Avatar   []byte   // length-prefixed raw bytes, decoded zero-copy
}

// Point is an all-fixed-width fixture, so its generated codec takes the
// blittable memmove fast path.
//
//goserde:generate
type Point struct {
	X int32 // fixed-width coordinate
	Y int32 // fixed-width coordinate
	Z int32 // fixed-width coordinate
}
