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

package codec

import "math"

// F64bits returns the IEEE 754 binary representation of f, the bit pattern that
// generated code writes for a float64 field.
func F64bits(f float64) uint64 { return math.Float64bits(f) }

// Bitsf64 returns the float64 whose IEEE 754 binary representation is u. It is
// the inverse of [F64bits] and is used to decode a float64 field.
func Bitsf64(u uint64) float64 { return math.Float64frombits(u) }

// F32bits returns the IEEE 754 binary representation of f, the bit pattern that
// generated code writes for a float32 field.
func F32bits(f float32) uint32 { return math.Float32bits(f) }

// Bitsf32 returns the float32 whose IEEE 754 binary representation is u. It is
// the inverse of [F32bits] and is used to decode a float32 field.
func Bitsf32(u uint32) float32 { return math.Float32frombits(u) }

// f64bits and bitsf64 are unexported aliases used by the hand-written reference
// codec in record.go so it reads like generator output without a package prefix.
func f64bits(f float64) uint64 { return F64bits(f) }
func bitsf64(u uint64) float64 { return Bitsf64(u) }
