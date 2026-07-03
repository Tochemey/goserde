# Changelog

All notable changes to goserde are documented in this file. The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Changed

- **Fast mode now bulk-copies slices and arrays of fixed-width elements.** Instead of an element loop, Marshal and Unmarshal emit a single raw-memory copy for slices and fixed arrays whose elements are explicitly sized basics (`bool`, `u/int8/16/32/64`, `float32/64`), named variants of those, or nested fixed arrays of the same. Decoding still fills an owned backing array (reused by capacity), so nothing new aliases the input buffer. `[]int` and `[]uint` keep the element-wise path because their in-memory width is platform-dependent, and `[]time.Time` keeps it because times encode as UnixNano. Safe mode is unaffected, and the fast-mode wire format is unchanged on little-endian architectures.
- **Performance:** on the reference `Record` (Apple M1), marshal drops from ~14 ns to ~11 ns and unmarshal from ~28 ns to ~24 ns. The win grows with slice length: a 1000-element `[]uint32` encodes and decodes at memcpy speed (~60 ns). The hand-tuned reference codecs (`codec/record.go`, `benchcompare`) were updated to the same shape.
- **Decode now reuses pointer pointees and union members.** Decoding into a value whose pointer field is already non-nil overwrites the existing pointee in place instead of allocating a fresh one, and a union field whose dynamic type matches the incoming tag is decoded into the existing member (a mismatch or nil allocates as before). This extends the existing slice and map reuse contract to pointers and unions in both modes; a wire nil still clears the field. The flip sides of in-place reuse, shared with `encoding/json`'s pointer behavior: a retained old pointee is mutated rather than orphaned; a destination whose pointer fields or union members alias each other is decoded through the shared pointee (each slot overwrites it, so do not alias slots in a reused destination); fields the generator skips (unexported or `goserde:"-"`) keep the reused pointee's previous contents instead of resetting to zero; and after a safe-mode decode error the destination is, as before, in an unspecified partial state and must not be used.
- **Fast mode now bulk-copies slices and arrays of blittable structs.** A named struct whose fields are all fixed-width already encodes as one raw-memory copy; slices and arrays of such structs now do too, replacing the per-element method loop with a single memmove on both marshal and unmarshal (sized via `unsafe.Sizeof`, since element padding is platform-dependent). Decode still fills an owned backing array reused by capacity. The wire bytes are unchanged: the loop produced the same contiguous native-layout block. On a 1000-element slice of an 8-byte struct (Apple M1), marshal drops from 1990 ns to 119 ns and unmarshal from 1990 ns to 110 ns, about 17x; the `Nested` reference shape improves from 16.9 ns to 12.3 ns marshal and 36 ns to 32 ns unmarshal (10 ns reused). Safe mode keeps the portable field-by-field loop.
- **Marshal and Size iterate struct, array, and `time.Time` slice elements by index.** The previous `for _, e := range` loops copied each element into the iteration variable before encoding it, which costs real time for wide elements: marshalling 16 wide (~90 B) struct elements drops from 161 ns to 137 ns (~15%). Slim elements such as strings keep the value loop, which is marginally faster for them.
- **Performance:** repeated decode into the same destination on the pointer-heavy `Nested` shape (Apple M1) drops from 36 ns / 32 B / 2 allocs to 14.6 ns / 0 allocs, about 2.5x. Fresh-destination decode is unchanged.
- `codec.UvarintSize` is now branch-free via `bits.Len64` (same results, lower inlining cost). Benchmarks are unchanged; investigation confirmed the varint primitives all inline and are not a bottleneck.

### Added

- **User-owned codecs.** A `//goserde:generate` struct that already declares all four codec methods (`Size`, `Marshal`, `Append`, `Unmarshal`) elsewhere in its package is now skipped with a notice instead of causing a method-redeclaration compile error, so you can move the generated methods into your own file next to the struct and keep running `go generate`. Declaring only part of the set is rejected with an error. Adopted codecs stop tracking the generator (wire-format fixes will not reach them), and slices of user-owned structs keep the per-element loop instead of the bulk copy, since the hand-written codec is a black box. Supporting changes: `-out` is validated as a bare non-test `.go` file name (Go methods must live in the receiver's package, so the file is always written into `-dir`), and the out file is no longer parsed during generation, so a stale or broken previous run can never break regeneration.
- **Single-pass encoding via a generated `Append` method.** Every codec now also gets `Append(b []byte) []byte`, which appends the value's encoding to b in one tree walk, growing the buffer as needed, and writes byte-identical output to `Marshal`. `codec.Into` now encodes through it, eliminating the separate `Size()` walk (which re-iterated every slice, map, and string): on a reused buffer (Apple M1), the map-heavy shape drops from 124 ns to 79 ns (~36%), nested from 13.6 ns to 8.8 ns, and string-heavy from 23.5 ns to 19.6 ns, making `Into` cost the same as the bare Marshal pass. `codec.Bytes` keeps the two-pass form: one exactly sized allocation is optimal for one-shot encoding. New `codec` append primitives (`AppendU16/32/64`, `AppendUvarint`) back the generated code and are covered by the SemVer promise. Note for hand-written codecs: the `codec.Marshaler` interface now includes `Append`, a compile-time breaking change permitted pre-1.0; implement it by mirroring your `Marshal` with the append primitives (see `codec/record.go`).
- **Benchmark coverage and guardrails.** `benchcompare` now runs the mus and benc head-to-head across all five generated shapes (fixed-width, flat mixed, nested + pointer, map-heavy, string-heavy) in addition to `Record`, with round-trip tests keeping every hand-written competitor codec honest; goserde leads every cell, including map-heavy decode (README tables updated). The shape suite gains union round-trip and 1000-element map, string-slice, and struct-slice benchmarks. CI gains a report-only `bench-smoke` job that runs the shape benchmarks on a pull request's base and head and prints a `benchstat` comparison in the job summary, so perf regressions surface in review without a flaky hard gate.

### Fixed

- **Blittable structs no longer leak skipped fields onto the wire.** Since v0.1.0, a struct whose *serialized* fields were all fixed-width took the whole-struct memmove even when it also had an excluded (`goserde:"-"`) or unexported field, so the skipped field's raw bytes were written to the output (a potential secret leak) and overwritten on decode, violating the exclusion contract. The blit fast path now requires the serializable fields to cover the entire struct; partially covered structs, and slices/arrays of them, fall back to the correct field-by-field codec. Structs with every field exported and untagged (the common case) keep the memmove path, and the wire format changes only for the previously mis-encoded structs.

## [v0.1.0] - 2026-06-29

First public release of goserde: a code-generated, zero-reflection binary serializer for Go, built for maximum encode/decode throughput when you own both ends of the wire.

goserde generates type-specific `Size`/`Marshal`/`Unmarshal` methods for your structs at build time. No reflection on the hot path, no schema language, no runtime type registry, and zero external dependencies.

### Highlights

- **Code generation, not reflection.** Annotate a struct with `//goserde:generate`, run `go generate`, and get straight-line `Size`/`Marshal`/`Unmarshal` over a single running offset.
- **Two modes, one API.** Generated method signatures are identical; you pick the trade-off at generation time:
  - **Fast** (default): zero-copy decoding via `unsafe` and a single `memmove` for all-fixed-width structs. For trusted bytes on the same architecture.
  - **Safe** (`-safe`): bounds-checked decoding that returns `codec.ErrShortBuffer` instead of panicking, copies decoded data, and uses a portable little-endian format. For untrusted or cross-machine input.
- **Broad type support:** all integer and float widths, `bool`, `string`, `[]byte`, slices, fixed arrays, maps, pointers, nested annotated structs, `time.Time` (as int64 Unix-nanoseconds, UTC), and tagged unions via `//goserde:union`. Exclude fields with `` `goserde:"-"` ``.
- **Zero dependencies.** The generator is standard-library only (`go/parser` + `go/types`, no `go/packages`, no network). Builds and tests fully offline.
- **A public codec toolkit.** The `codec` package exposes both a convenience API (`Bytes`/`Into`/`From`) and the low-level primitives (varint, zigzag, fixed-width LE read/write, zero-copy conversions, float bit-casts) so you can hand-write a codec that shares the exact wire format.

### Performance

Flat `Record` struct, same machine (Go 1.26, darwin/arm64, Apple M1):

| Operation | goserde         | encoding/json      |
|-----------|-----------------|--------------------|
| Marshal   | 14 ns, 0 allocs | 399 ns, 1 alloc    |
| Unmarshal | 28 ns, 1 alloc  | 2258 ns, 12 allocs |

Roughly 28x faster than JSON on marshal and 81x on unmarshal, and ahead of `benc` and `mus` on the same data. Reproduce with `make bench`, `make bench-shapes`, and `make compare`.

### Install

```bash
# Codec support library (imported by generated code)
go get github.com/tochemey/goserde

# Code generator
go install github.com/tochemey/goserde/cmd/goserdegen@v0.1.0
```

### Compatibility and caveats

goserde is **pre-1.0**: the API and wire format may change between minor versions until 1.0.0.

- **Fast mode is for trusted, same-machine bytes.** It trusts its input (malformed bytes may panic) and decoded `string`/`[]byte` values alias the input buffer, so that buffer must outlive the decoded value and must not be mutated or pooled while it is in use. Decode anything you did not produce yourself with safe mode.
- **No schema evolution.** The schema lives entirely in the generated code: no field names, tags, or type markers on the wire. Adding, removing, or reordering fields breaks compatibility. For evolving schemas, reach for Protobuf, Cap'n Proto, or FlatBuffers.
- **Wire-format portability:** safe-mode bytes are portable across architectures and Go versions; fast-mode bytes are native-layout and should be treated as ephemeral.

Requires Go 1.26.

[Unreleased]: https://github.com/tochemey/goserde/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/tochemey/goserde/releases/tag/v0.1.0
