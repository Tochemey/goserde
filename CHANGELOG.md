# Changelog

All notable changes to goserde are documented in this file. The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

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

[0.1.0]: https://github.com/tochemey/goserde/releases/tag/v0.1.0
