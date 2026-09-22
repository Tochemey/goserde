# goserde

[![GitHub Actions Workflow Status](https://img.shields.io/github/actions/workflow/status/tochemey/goserde/ci.yml)](https://github.com/tochemey/goserde/actions/workflows/ci.yml)
[![codecov](https://img.shields.io/codecov/c/github/tochemey/goserde?branch=main)](https://codecov.io/gh/tochemey/goserde)
[![Go Reference](https://pkg.go.dev/badge/github.com/tochemey/goserde.svg)](https://pkg.go.dev/github.com/tochemey/goserde)

A code-generated, zero-reflection binary serializer for Go, built for maximum encode/decode throughput when you own both ends of the wire.

goserde generates type-specific `Size`/`Marshal`/`Append`/`Unmarshal` methods for your structs at build time. There is no reflection on the hot path, no schema language, and no runtime type registry: just straight-line Go over a single running offset, with inlinable primitives and zero-copy decoding.

```
Marshal    11 ns/op    0 allocs     (37× faster than encoding/json)
Unmarshal  24 ns/op    1 alloc      (96× faster than encoding/json)
```

## Table of contents

- [Relevance](#relevance)
- [Install](#install)
- [Quick start](#quick-start)
- [Supported types](#supported-types)
  - [Tagged unions](#tagged-unions)
- [Modes](#modes)
  - [Owning the generated methods](#owning-the-generated-methods)
- [Performance](#performance)
  - [vs the standard library](#vs-the-standard-library)
  - [vs other fast serializers](#vs-other-fast-serializers)
  - [Across struct shapes](#across-struct-shapes)
- [Wire format](#wire-format)
- [Compatibility](#compatibility)
- [How it works](#how-it-works)
- [Development](#development)

## Relevance

**Use it when** you control both the encoder and the decoder and want the smallest, fastest possible binary representation: RPC between your own services, on-disk caches, IPC, or embedded pipelines. The default **fast mode** assumes trusted input on the same architecture; an opt-in **safe mode** adds bounds-checked decoding, copies of decoded data, and a portable little-endian format for untrusted or cross-machine input (see [Modes](#modes)).

**Look elsewhere when** you need schema evolution, a self-describing format, or forward/backward compatibility with peers on different versions. goserde's schema lives entirely in the generated code, with no field tags or type markers on the wire, so the format does not evolve on its own. For that, reach for a schema-evolving format such as Protobuf, Cap'n Proto, or FlatBuffers.

## Install

```bash
# The codec support library (imported by generated code)
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
import "github.com/tochemey/goserde/codec"

u := &User{ID: 1, Name: "ada", Tags: []int32{1, 2, 3}}

// Encode into a fresh, exactly sized buffer.
data := codec.Bytes(u)

// Decode.
var out User
if err := codec.From(&out, data); err != nil {
	log.Fatal(err)
}
```

On the hot path, drive the generated methods directly and reuse buffers to stay allocation-free:

```go
buf = codec.Into(u, buf) // single-pass append encode; reuses buf's capacity

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

Fields tagged `` `goserde:"-"` `` are excluded from the wire and are never written by decode: zero in a fresh destination, unchanged in a reused one. This is useful for fields of types goserde cannot serialize (channels, funcs).

```go
//goserde:generate
type Session struct {
	ID    uint64
	Token string
	conn  net.Conn `goserde:"-"` // excluded
}
```

### Tagged unions

Declare a polymorphic field by listing its concrete members above an interface. Each member must itself be a `//goserde:generate` struct whose pointer implements the interface:

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

On the wire a union is a varint tag (`0` = nil, `1` = first member, …) followed by the member's payload. **Member order is the wire contract**: only ever append new members, never reorder. Decoding an unknown tag returns `codec.ErrUnknownUnionTag` rather than panicking.

## Modes

A package is generated in exactly one mode; the method signatures are identical, so callers don't change.

| Mode               | Flag    | Unmarshal on bad input         | Decoded `string`/`[]byte` | Portable            |
|--------------------|---------|--------------------------------|---------------------------|---------------------|
| **Fast** (default) | none    | may panic (trusted bytes)      | zero-copy (aliases input) | no (native memory)  |
| **Safe**           | `-safe` | returns `codec.ErrShortBuffer` | copied (owns its bytes)   | yes (little-endian) |

Fast mode is for trusted bytes on identical architectures: it uses `unsafe` zero-copy decoding and a single `memmove` for all-fixed-width structs and for slices and arrays of fixed-width elements or of those structs. Safe mode bounds-checks every read, copies length-prefixed payloads, and encodes field-by-field in little-endian, portable across architectures and Go versions, at a modest cost.

**Decoding untrusted input? Use safe mode.** Fast mode trusts its bytes, and that has two consequences. First, malformed input may panic or read out of bounds, so never decode bytes you did not produce yourself (network payloads, files written by other processes, user input) in fast mode. Second, a decoded `string` or `[]byte` aliases the buffer passed to `Unmarshal` rather than copying, so that buffer must outlive the decoded value and must not be mutated or returned to a pool while the value is in use. Safe mode removes both hazards: it returns `codec.ErrShortBuffer` on truncated input and copies decoded data so the result owns its bytes.

```bash
goserdegen -dir . -out goserde_gen.go        # fast (default)
goserdegen -dir . -out goserde_gen.go -safe  # safe / portable
```

`-out` names the generated file and can be customized, but it is always a bare `.go` file name written into `-dir`: Go methods must be declared in the same package as their receiver type, so the generated code cannot live anywhere else (and a `_test.go` name is rejected because the methods would vanish from ordinary builds). The out file is never parsed on the next run, so a stale or broken copy cannot break regeneration.

### Owning the generated methods

If you prefer the codec methods to live next to the struct, move them: cut the four methods (`Size`, `Marshal`, `Append`, `Unmarshal`) for a type out of the generated file and paste them into your own file in the same package. On the next run the generator detects the full method set, prints `skipping T: codec methods are user-owned`, and stops emitting that type; it remains fully usable as a nested field or union member of generated types.

Two rules apply. Own all four methods or none: a partial set is rejected, since a half-generated codec has no sound owner for the wire format. And an adopted codec stops tracking the generator: wire-format fixes and optimizations in newer goserdegen releases will not reach it, so keep it in sync yourself (generated types whose slices contain a user-owned struct also drop the bulk-copy fast path, since your codec is a black box to the generator).

## Performance

All numbers below are from the same machine (Go 1.26, darwin/arm64, Apple M1). Reproduce with `make bench` (vs the standard library), `make bench-shapes` (across shapes), and `make compare` (vs mus and benc).

### vs the standard library

Flat `Record` struct (`uint64`, `float64`, `bool`, `string`, `[]uint32`, `[]byte`):

| Operation | goserde            | encoding/json      | encoding/gob¹ |
|-----------|--------------------|--------------------|---------------|
| Marshal   | **11 ns**, 0 alloc | 400 ns, 1 alloc    | 190 ns        |
| Unmarshal | **24 ns**, 1 alloc | 2264 ns, 12 allocs | n/a           |

¹ gob marshal uses a reused encoder. goserde is ~37× faster than JSON on marshal and ~96× on unmarshal, with a fraction of the allocations.

### vs other fast serializers

Same `Record` data, 3 runs each, with [mus](https://github.com/mus-format/mus-go) v0.10.2 and [benc](https://github.com/deneonet/benc) v1.1.8:

| Serializer   | Marshal     | Unmarshal   | Payload | Decode allocs |
|--------------|-------------|-------------|---------|---------------|
| **goserde**  | **10.6 ns** | **23.3 ns** | 139 B   | **1**         |
| benc         | 25.3 ns     | 43.9 ns     | 143 B   | 1             |
| mus (raw)    | 33.9 ns     | 68.7 ns     | 139 B   | 2             |
| mus (varint) | 33.7 ns     | 74.5 ns     | 109 B   | 2             |

goserde is ~2.4× / ~1.9× faster than benc (marshal/unmarshal) and ~3× faster than mus on both. The main reasons are **no interface dispatch in the generated code** and **single-memmove encoding of fixed-width slices**: running mus in fixed-width "raw" mode produces goserde's exact 139 B format yet barely changes its speed, so the win is the straight-line code, not the format. mus and benc are fuller-featured libraries (schema evolution, validation, versioning); goserde trades those for raw throughput.

The head-to-head holds across every benchmark shape, not just `Record`. Each cell is marshal / unmarshal ns on the same fixtures as the shape table below, goserde running its generated codecs and mus/benc hand-written at their fastest settings:

| Shape        | goserde         | mus         | benc        |
|--------------|-----------------|-------------|-------------|
| Fixed-width  | **0.4 / 0.9**¹  | 2.4 / 2.8   | 3.2 / 3.6   |
| Flat mixed   | **6.3 / 5.4**   | 10.1 / 20.2 | 6.7 / 9.5   |
| Nested + ptr | **11.4 / 31.1** | 19.1 / 45.4 | 16.5 / 39.6 |
| Map-heavy    | **76 / 147**    | 111 / 195   | 88 / 202    |
| String-heavy | **19.4 / 31.0** | 30.3 / 48.8 | 19.8 / 39.7 |

¹ Direct calls that inline fully; the shape table below goes through a shared helper that adds ~2 ns. Payload sizes are equal or smaller for goserde on every shape except fixed-width, where the whole-struct memmove writes its 24 padded bytes against 17 for mus/benc: padding on the wire is the price of the sub-ns copy.

### Across struct shapes

goserde's lead is shape-dependent. Marshal is zero-alloc on every shape:

| Shape               | Marshal | Unmarshal | Decode allocs | Notes                          |
|---------------------|---------|-----------|---------------|--------------------------------|
| Fixed-width (4 fld) | 2.4 ns  | 2.4 ns    | 0             | single memmove (blittable)     |
| Flat + string/bytes | 7.1 ns  | 6.3 ns    | 0             | zero-copy `string`/`[]byte`    |
| String-heavy        | 19.6 ns | 32.0 ns   | 1             | `[]string` must allocate       |
| Nested + pointers   | 12.3 ns | 32.0 ns   | 2             | pointer + slice-of-struct      |
| Map-heavy           | 78.7 ns | 149.8 ns  | 4             | `make(map)` + per-entry insert |

**Decode reuse:** decoding repeatedly into the *same* value reuses its slices (by capacity), maps (via `clear`), pointer pointees, and union members whose dynamic type matches the incoming tag. The map-heavy shape above drops from **150 ns / 4 allocs to 63 ns / 0 allocs** when the destination is reused across calls, and the nested-pointer shape from **32 ns / 2 allocs to 10 ns / 0 allocs**, the right pattern for a read loop. The trade: a reused destination is overwritten in place, so copy out anything you need to keep before decoding over it, and (as with `encoding/json`) its pointer fields and union members must not alias each other, since every aliased slot decodes through the one shared pointee.

**Encode without the Size pass:** `codec.Into` encodes through the generated `Append` method, a single tree walk that grows the buffer as it goes, instead of the two-pass `Size()`-then-`Marshal`. On a reused buffer that makes the map-heavy shape **124 -> 79 ns (-36%)**, nested **13.6 -> 8.8 ns (-35%)**, and string-heavy **23.5 -> 19.6 ns (-17%)**: encoding into a reused buffer now costs the same as the bare Marshal pass. `codec.Bytes` keeps the two-pass form deliberately, since one exactly sized allocation is optimal for one-shot use.

Fixed and flat structs are where goserde dominates. Map-heavy decode is its weakest shape in absolute terms, though still ahead of mus and benc (see the table above); destination reuse is what makes it cheap in a loop.

## Wire format

Fixed-width little-endian numerics; LEB128 varint length prefixes for strings, slices, and maps; a 1-byte nil flag for pointers; an 8-byte Unix-nanosecond int64 for `time.Time`. In fast mode, all-fixed-width structs and slices/arrays of fixed-width elements (or of all-fixed-width structs) are encoded as a single memory copy. There are no field names, tags, or type markers on the wire: the schema lives entirely in the generated code.

This format is **not** self-describing and, in fast mode, **not** portable across architectures or Go versions. That is the deliberate trade for speed.

## Compatibility

goserde is pre-1.0: the API and wire format may change between minor versions until 1.0.0. From 1.0.0 onward the following semantic-versioning promise applies.

**Covered by SemVer (stable within a major version):**

- The generated method set on your types: `Size() int`, `Marshal([]byte) int`, `Append([]byte) []byte`, and `Unmarshal([]byte) (int, error)`.
- The full `codec` package: the convenience API (`Bytes`, `Into`, `From`, and the `Marshaler` / `Unmarshaler` interfaces) and the low-level encoding primitives (`PutU32`/`U32` and friends, the `AppendU32`/`AppendUvarint` append family, `PutUvarint`/`Uvarint`/`UvarintSize`, `Zig`/`Zag`, `B2S`/`S2B`, the float bit-casts) you can use to hand-write a codec instead of running the generator. Generated and hand-written codecs call the same primitives, so they share an identical wire format.
- The generator directives `//goserde:generate` and `//goserde:union`, and the `goserde:"-"` struct tag.

**Wire-format stability:**

- **Safe mode** (`-safe`) is the portable format: little-endian and bounds-checked, stable across architectures and Go versions. Use it for data at rest or exchanged between machines.
- **Fast mode** (default) is not portable. It uses native byte order and encodes all-fixed-width structs as a copy of their in-memory layout, so the bytes can differ across architectures and across Go versions whose struct layout differs. Treat fast-mode bytes as ephemeral and decode them with the same goserde version, architecture, and Go toolchain that produced them.

Regenerating codecs after upgrading goserde is always safe and recommended.

## How it works

The generator (`cmd/goserdegen`) is **standard-library only**: it loads and type-checks your package with `go/parser` and `go/types` (no `go/packages`, no network), finds `//goserde:generate` structs, and emits a codec shaped exactly like the hand-tuned reference in [`codec/record.go`](codec/record.go). Generated code imports goserde's `codec` package, whose primitives (varint, zigzag, little-endian fixed-width R/W, zero-copy conversions, float bit-casts) are written to inline.

The import path of that `codec` package is derived automatically from the module that built `goserdegen`, so forking under a different module path "just works" with no flags to keep in sync.

## Development

```bash
go build ./...        # build everything (zero dependencies)
go test ./...         # run the full test suite
go generate ./...     # regenerate codecs after editing annotated structs
make bench-shapes     # benchmark across struct shapes
make compare          # head-to-head vs mus and benc (needs network)
```
