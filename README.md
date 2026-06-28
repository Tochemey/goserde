# goserde

[![CI](https://img.shields.io/github/actions/workflow/status/tochemey/goserde/ci.yml?branch=main&label=CI)](https://github.com/tochemey/goserde/actions/workflows/ci.yml)
[![codecov](https://img.shields.io/codecov/c/github/tochemey/goserde?branch=main)](https://codecov.io/gh/tochemey/goserde)
[![Go Reference](https://pkg.go.dev/badge/github.com/tochemey/goserde.svg)](https://pkg.go.dev/github.com/tochemey/goserde)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

A code-generated, zero-reflection binary serializer for Go, built for maximum
encode/decode throughput when you own both ends of the wire.

goserde generates type-specific `Size`/`Marshal`/`Unmarshal` methods for your
structs at build time. There is no reflection on the hot path, no schema
language, and no runtime type registry: just straight-line Go over a single
running offset, with inlinable primitives and zero-copy decoding.

```
Marshal    14 ns/op    0 allocs     (28× faster than encoding/json)
Unmarshal  28 ns/op    1 alloc      (81× faster than encoding/json)
```

## Is goserde for you?

**Use it when** you control both the encoder and the decoder and want the
smallest, fastest possible binary representation: RPC between your own services,
on-disk caches, IPC, or embedded pipelines. The default **fast mode** assumes
trusted input on the same architecture; an opt-in **safe mode** adds
bounds-checked decoding and a portable, little-endian format (see [Modes](#modes)).

**Look elsewhere when** you need schema evolution, a self-describing format, or
forward/backward compatibility with peers on different versions. goserde's schema
lives entirely in the generated code, with no field tags or type markers on the
wire, so the format does not evolve on its own. For that, reach for a
schema-evolving format such as Protobuf, Cap'n Proto, or FlatBuffers.

## Install

```bash
# The runtime library (imported by generated code)
go get github.com/tochemey/goserde

# The code generator
go install github.com/tochemey/goserde/cmd/goserdegen@latest
```

The library has **zero external dependencies** and builds offline.

## Quick start

Annotate a struct with the `//goserde:generate` directive:

```go
package model

//go:generate goserdegen -dir . -out goserde_gen.go

//goserde:generate
type User struct {
	ID   uint64
	Name string
	Tags []int32
}
```

Generate the codec:

```bash
go generate ./...
```

Encode and decode with the convenience helpers:

```go
import "github.com/tochemey/goserde/runtime"

u := &User{ID: 1, Name: "ada", Tags: []int32{1, 2, 3}}

// Encode into a fresh, exactly sized buffer.
data := runtime.Bytes(u)

// Decode.
var out User
if err := runtime.From(&out, data); err != nil {
	log.Fatal(err)
}
```

On the hot path, drive the generated methods directly and reuse buffers to stay
allocation-free:

```go
buf = runtime.Into(u, buf) // reuses buf when it has capacity, else allocates

var out User
out.Unmarshal(data)        // returns (bytesRead int, err error)
```

## Supported types

| Category   | Types                                                                        |
|------------|------------------------------------------------------------------------------|
| Integers   | `int`, `int8/16/32/64`, `uint`, `uint8/16/32/64`                             |
| Floats     | `float32`, `float64`                                                         |
| Scalars    | `bool`, `string`, `[]byte`                                                   |
| Composites | slices `[]T`, fixed arrays `[N]T`, maps `map[K]V`, pointers `*T`             |
| Structs    | nested structs (each annotated with `//goserde:generate`)                    |
| Time       | `time.Time` (as int64 Unix-nanoseconds, UTC)                                 |
| Unions     | interface fields via `//goserde:union` (see [Tagged unions](#tagged-unions)) |

Fields tagged `` `goserde:"-"` `` are excluded from the wire and decode back to
their zero value, useful for fields of types goserde cannot serialize
(channels, funcs).

```go
//goserde:generate
type Session struct {
	ID    uint64
	Token string
	conn  net.Conn `goserde:"-"` // excluded
}
```

### Tagged unions

Declare a polymorphic field by listing its concrete members above an interface.
Each member must itself be a `//goserde:generate` struct whose pointer implements
the interface:

```go
//goserde:generate
type Circle struct{ R float64 }

func (*Circle) isShape() {}

//goserde:generate
type Square struct{ Side float64 }

func (*Square) isShape() {}

//goserde:union Circle Square
type Shape interface{ isShape() }

//goserde:generate
type Drawing struct {
	Name  string
	Shape Shape   // holds *Circle, *Square, or nil
}
```

On the wire a union is a varint tag (`0` = nil, `1` = first member, …) followed
by the member's payload. **Member order is the wire contract**: only ever
append new members, never reorder. Decoding an unknown tag returns
`runtime.ErrUnknownUnionTag` rather than panicking.

## Modes

A package is generated in exactly one mode; the method signatures are identical,
so callers don't change.

| Mode               | Flag    | Unmarshal on bad input           | Decoded `string`/`[]byte` | Portable            |
|--------------------|---------|----------------------------------|---------------------------|---------------------|
| **Fast** (default) | none    | may panic (trusted bytes)        | zero-copy (aliases input) | no (native memory)  |
| **Safe**           | `-safe` | returns `runtime.ErrShortBuffer` | copied (owns its bytes)   | yes (little-endian) |

Fast mode is for trusted bytes on identical architectures: it uses `unsafe`
zero-copy decoding and a single `memmove` for all-fixed-width structs. Safe mode
bounds-checks every read, copies length-prefixed payloads, and encodes
field-by-field in little-endian, portable across architectures and Go versions,
at a modest cost.

```bash
goserdegen -dir . -out goserde_gen.go        # fast (default)
goserdegen -dir . -out goserde_gen.go -safe  # safe / portable
```

## Performance

All numbers below are from the same machine (Go 1.26, darwin/arm64, Apple M1).
Reproduce with `make bench` (vs the standard library), `make bench-shapes`
(across shapes), and `make compare` (vs mus and benc).

### vs the standard library

Flat `Record` struct (`uint64`, `float64`, `bool`, `string`, `[]uint32`,
`[]byte`):

| Operation | goserde            | encoding/json      | encoding/gob¹ |
|-----------|--------------------|--------------------|---------------|
| Marshal   | **14 ns**, 0 alloc | 399 ns, 1 alloc    | 191 ns        |
| Unmarshal | **28 ns**, 1 alloc | 2258 ns, 12 allocs | n/a           |

¹ gob marshal uses a reused encoder. goserde is ~28× faster than JSON on
marshal and ~81× on unmarshal, with a fraction of the allocations.

### vs other fast serializers

Same `Record` data, 3 runs each, with [mus](https://github.com/mus-format/mus-go)
v0.10.2 and [benc](https://github.com/deneonet/benc) v1.1.8:

| Serializer   | Marshal     | Unmarshal   | Payload | Decode allocs |
|--------------|-------------|-------------|---------|---------------|
| **goserde**  | **13.3 ns** | **28.0 ns** | 139 B   | **1**         |
| benc         | 25.3 ns     | 45.1 ns     | 143 B   | 1             |
| mus (raw)    | 34.3 ns     | 68.6 ns     | 139 B   | 2             |
| mus (varint) | 34.2 ns     | 73.0 ns     | 109 B   | 2             |

goserde is ~1.9× / ~1.6× faster than benc (marshal/unmarshal) and ~2.6× faster
than mus on both. The main reason is **no interface dispatch in the generated
code**: running mus in fixed-width "raw" mode produces goserde's exact 139 B
format yet barely changes its speed, so the win is the straight-line code, not
the format. mus and benc are fuller-featured libraries (schema evolution,
validation, versioning); goserde trades those for raw throughput.

### Across struct shapes

goserde's lead is shape-dependent. Marshal is zero-alloc on every shape:

| Shape               | Marshal | Unmarshal | Decode allocs | Notes                          |
|---------------------|---------|-----------|---------------|--------------------------------|
| Fixed-width (4 fld) | 2.4 ns  | 2.4 ns    | 0             | single memmove (blittable)     |
| Flat + string/bytes | 7.1 ns  | 6.3 ns    | 0             | zero-copy `string`/`[]byte`    |
| String-heavy        | 19.6 ns | 33.0 ns   | 1             | `[]string` must allocate       |
| Nested + pointers   | 16.9 ns | 36.1 ns   | 2             | pointer + slice-of-struct      |
| Map-heavy           | 77.5 ns | 151.9 ns  | 4             | `make(map)` + per-entry insert |

**Decode reuse:** decoding repeatedly into the *same* value reuses its slices
(by capacity) and maps (via `clear`). The map-heavy shape above drops from
**152 ns / 4 allocs to 67 ns / 0 allocs** when the destination is reused across
calls, the right pattern for a read loop.

Fixed and flat structs are where goserde dominates. Map-heavy decode is its
weakest point on a fresh destination, where mus and benc are competitive.

## Wire format

Fixed-width little-endian numerics; LEB128 varint length prefixes for
strings, slices, and maps; a 1-byte nil flag for pointers; an 8-byte
Unix-nanosecond int64 for `time.Time`. All-fixed-width structs are encoded as a
single memory copy in fast mode. There are no field names, tags, or type
markers on the wire: the schema lives entirely in the generated code.

This format is **not** self-describing and, in fast mode, **not** portable across
architectures or Go versions. That is the deliberate trade for speed.

## How it works

The generator (`cmd/goserdegen`) is **standard-library only**: it loads and
type-checks your package with `go/parser` and `go/types` (no `go/packages`, no
network), finds `//goserde:generate` structs, and emits a codec shaped exactly
like the hand-tuned reference in [`runtime/record.go`](runtime/record.go).
Generated code imports goserde's `runtime` package, whose primitives (varint,
zigzag, little-endian fixed-width R/W, zero-copy conversions, float bit-casts)
are written to inline.

The import path of that `runtime` package is derived automatically from the
module that built `goserdegen`, so forking under a different module path "just
works" with no flags to keep in sync.

## Development

```bash
go build ./...        # build everything (zero dependencies)
go test ./...         # run the full test suite
go generate ./...     # regenerate codecs after editing annotated structs
make bench-shapes     # benchmark across struct shapes
make compare          # head-to-head vs mus and benc (needs network)
```
