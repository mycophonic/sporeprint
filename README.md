# Sporeprint

> * an audio fingerprinting tool with cgo bindings for Chromaprint
> * [the powdery deposit obtained by allowing spores of a fungal fruit body to fall onto a surface underneath](https://en.wikipedia.org/wiki/Spore_print)

![logo.png](logo.png)


## Purpose

fpcalc (the Chromaprint default command line tools) heavily relies on ffmpeg, and Chromaprint
offers multiple different FFT backends with a variety of licenses.

While this is fine for users, the [licensing situation](https://github.com/acoustid/chromaprint/blob/master/LICENSE.md)
with it is complex if you intend on reusing and distributing.

Further, fpcalc itself is in C++, and dynamically links to ffmpeg libs.

Sporeprint (Apache license) addresses the first problem by linking solely against [Chromaprint](https://github.com/acoustid/chromaprint) (MIT),
compiled only with [KissFFT](https://github.com/mborgerding/kissfft) (BSD-3), and offers go bindings for Chromaprint, making it
possible to integrate as a library without having to shell out.

TL;DR sporeprint is "fpcalc-light" for the go world, without license minefield or dynamic linkage.

## Trade-offs

### Sporeprint does not convert the audio data on its own...

Sporeprint expects PCM. And see below...

### ... it must be little-endian 16 bits 11025/mono

Chromaprint internally performs fingerprinting on 11025/mono, and _claims_ to accept different rates (and stereo)
as input, apparently using a homegrown resampler (?) to convert to its desired format.

However, this resampler appears to be buggy, and definitely breaks on certain inputs (or over a certain number of seconds).
Or at least yields different results than ffmpeg.

This problem is presumably hidden because fpcalc itself does resample to the final desired format (using ffmpeg)
_before_ calling Chromaprint methods. The bottom-line is still: it seems you cannot reliably use Chromaprint without
taking care of the downsampling yourself.

```bash
ffmpeg -i track.flac -f s16le -ar 11025 -ac 1 pipe:1 2>/dev/null | sporeprint
```

fpcalc will yield the same results when used with stdin:
```bash
ffmpeg -i track.flac -f s16le -ar 11025 -ac 1 pipe:1 2>/dev/null | fpcalc -format s16le -rate 11025 -channels 1  -
```

Obviously, the result may deviate slightly when using fpcalc directly on the source file (ffmpeg versions variants is a possible culprit).

Why mono and 11025?

Presumably Chromaprint authors figured this was the sweet spot for accuracy vs. speed, which certainly makes sense.

## Build

```bash
# Dependencies (Debian/Ubuntu)
sudo apt install cmake build-essential

# Build
./build.sh
```

This builds Chromaprint with **KissFFT only** - no FFmpeg, no FFTW3.

Then
```
make build
```

Binary will be in `bin/sporeprint`.

## Using in Go

Obviously you need to accept CGO.

See `cmd/sporeprint/main.go` for a working example, or just shell out to the provided binary.

## Status

Sporeprint has been tested against a mixture of about 10k files, providing the same fingerprint as fpcalc (stdin mode).

In terms of performance, both tools perform exactly the same, which is unsurprising given Chromaprint FFT bears the grunt
of the cost.

## Caveats

1. Some golangci-lint are apparently screwing the pooch with cgo.
Need to review the rules in there.
2. build command is unoptimized