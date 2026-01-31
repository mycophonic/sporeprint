# Code Audit: Sporeprint

Audit date: 2026-01-31

## Scope

All source code, build system, CI/CD, configuration, documentation,
and dependencies in the sporeprint project.

## Project Overview

Go CGO bindings for Chromaprint (MIT) with KissFFT (BSD-3). CLI tool and
library for audio fingerprinting. Apache 2.0 licensed.

- 5 Go packages, ~1,210 lines of production Go code
- Static linking to libchromaprint.a (157K)
- Platforms: macOS, Linux, Windows
- Go 1.25.6, Chromaprint 1.6.0

## Findings

### Security

**No critical or high findings.**

CGO hardening is thorough: `-fstack-protector-strong`, `-fPIE`,
`-D_FORTIFY_SOURCE=2`, full RELRO, NX stack (Linux). PIE binaries
with ASLR. Stripped release builds.

C memory management is correct: all `C.CString` allocations freed via
defer, `C.chromaprint_dealloc` called on all C-allocated buffers,
`unsafe.Slice` used for zero-copy with immediate `copy` to Go-owned memory.

No command injection, no SQL, no network, no user-controlled format strings.

The `fingerprint` command streams stdin in fixed 8KB chunks into
Chromaprint's `Feed()`. Memory usage is bounded regardless of audio
duration (Chromaprint's internal hash array grows at ~8 hashes/second,
~64 bytes/second).

### CGO Correctness

`chromaprint.go` guards all methods with nil-context checks, returning
`ErrFreed` after `Free()`. Double-free is safe (idempotent).

`Decode()` properly allocates C string, decodes via C API, copies result
to Go slice, frees C memory. The `unsafe.Slice` -> `copy` pattern avoids
holding C pointers past dealloc.

No data races: `Context` is not safe for concurrent use, but this is
standard Go convention (caller's responsibility). Tests are parallel
but each creates its own `Context`.

### Algorithm (compare package)

Based on the AcoustID PostgreSQL matching function. The implementation
matches the reference:

- Alignment search within +/-120 hashes (~15 seconds drift)
- XOR + popcount for bit distance (MaxBitError=2, 93.75% per-hash threshold)
- Histogram of matches per offset, score = best count / min(len)

Time complexity: O(N * MaxAlignOffset) where N = max fingerprint length.
For typical 120-second fingerprints (~960 hashes): ~230K iterations.
Benchmark confirms 3.4us per comparison. No performance concern.

The `counts` histogram allocates `len(fp1) + len(fp2) + 1` ints. For
120-second fingerprints this is ~15KB. Negligible.

### Test Coverage

**chromaprint:** 7 test functions (including subtests for post-Free
error checking). Covers lifecycle, determinism, chunked feeding, empty
input, error paths.

**compare:** 12 test functions + 1 benchmark. Covers identical, empty,
invalid encoding, phase-shifted signals, spectrally distinct signals,
offset detection, bit error rate, threshold matching. Uses real
Chromaprint fingerprints from synthetic PCM.

**integration (tests/):** 10 audio format subtests comparing sporeprint
output against fpcalc on generated audio (FLAC, MP3, AAC, OGG, ALAC at
various bitrates/sample rates).

**manual scripts:** battle.sh (correctness on real collections) and
speed.sh (throughput benchmarking).

**Coverage reporting:** `test-unit-cover` target produces per-function
coverage, HTML report, and enforces a configurable `COVER_MIN` threshold
(set to 35% for sporeprint). Runs as part of `make test` in CI.

Current total is 36.4%. The library packages are well covered
(chromaprint 63.4%, compare 88.9%). The total is pulled down by
`cmd/sporeprint` (CLI main, untestable via unit tests), `testutils`
(exercised indirectly by integration tests in a separate binary), and
`version` (trivial getters populated via ldflags).

### Build System

Comprehensive Makefile infrastructure (546 lines in common.mk). Linting
runs 66+ linters cross-platform. CI tests on ubuntu-24.04, macos-15,
windows-2025. Weekly go-latest job for forward compatibility.

Chromaprint built from source (v1.6.0 tag) as static library with
KissFFT backend. No dynamic linking.

Race test target correctly forces `-linkmode=external` to work around
Go internal linker limitations with CGO hardening flags.

### Linting Configuration

.golangci.yml enables all linters with 17 justified exclusions. Revive
configured with `enable-all-rules`. Test files have relaxed rules
(varnamelen, wrapcheck, etc.) which is appropriate.

Import grouping enforced by gci. Formatting by gofumpt + golines (120
char max).

### Documentation

README covers purpose, trade-offs, and build instructions. Honest about
Chromaprint's internal resampler issues.

QA.md reports real-world testing: 12,145 files, ~0.28% divergence from
fpcalc. Transparent about the 34 cases where fingerprints differ.

All packages have doc.go files. compare package has algorithm
documentation in doc.go; chromaprint package documents CGO bindings.

### Licensing

Apache 2.0 for sporeprint. NOTICE file attributes Chromaprint (MIT) with
full license text. KissFFT is BSD-3 (bundled inside Chromaprint build).
go-licenses lint target verifies all dependencies are Apache-2.0, MIT,
BSD-2-Clause, or BSD-3-Clause.

### Dependencies

4 direct Go dependencies:

| Module | Version | Purpose |
|--------|---------|---------|
| urfave/cli/v3 | v3.6.2 | CLI framework |
| mycophonic/primordium | v0.0.0-... | App initialization |
| mycophonic/agar | v0.1.0 | Audio generation (test only) |
| containerd/nerdctl/mod/tigron | v0.0.0-... | Test framework (test only) |

12 indirect dependencies (zerolog, x/crypto, x/sys, etc.). All from
reputable sources.

## Issues

| # | Severity | Finding | Status |
|---|----------|---------|--------|
| 1 | Info | Integration test profiling yields no useful CPU data (CGO + subprocess opacity) | Known limitation |

## Non-Issues (Investigated and Dismissed)

- **Memory exhaustion from unbounded audio:** Not real. Streaming 8KB
  buffer + Chromaprint internal state grows at ~64 bytes/second. No
  accumulation.
- **Dependency on external binaries (ffmpeg, fpcalc):** Test-only, with
  proper skip logic when not found. CI uses trusted package managers.
- **No default maximum audio length:** Processing time scales linearly
  but memory does not. No mitigation needed.

## Verdict

Production-ready. No security vulnerabilities identified. Code quality
is high with comprehensive linting, testing, and coverage enforcement.
No open issues remain. The only noted item is a known limitation of
Go's CPU profiler with CGO and subprocess-based integration tests.
