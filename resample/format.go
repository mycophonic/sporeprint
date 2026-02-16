/*
   Copyright Mycophonic.

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

package resample

import (
	"encoding/binary"
	"errors"
	"math"
)

// Format represents a PCM sample format.
type Format int

const (
	// FormatS16LE is signed 16-bit little-endian PCM.
	FormatS16LE Format = iota
	// FormatS32LE is signed 32-bit little-endian PCM.
	FormatS32LE
	// FormatF32LE is IEEE 754 32-bit float little-endian PCM.
	FormatF32LE
)

const (
	bytesPerS16 = 2
	bytesPerS32 = 4
	bytesPerF32 = 4
)

// ErrInvalidFormat is returned when the format string is not recognized.
var ErrInvalidFormat = errors.New("resample: invalid sample format")

// ParseFormat converts a format string to a Format constant.
// Accepted values: "s16le", "s32le", "f32le".
func ParseFormat(str string) (Format, error) {
	switch str {
	case "s16le":
		return FormatS16LE, nil
	case "s32le":
		return FormatS32LE, nil
	case "f32le":
		return FormatF32LE, nil
	default:
		return 0, ErrInvalidFormat
	}
}

// BytesPerSample returns the number of bytes per sample for the format.
func (f Format) BytesPerSample() int {
	switch f {
	case FormatS16LE:
		return bytesPerS16
	case FormatS32LE:
		return bytesPerS32
	case FormatF32LE:
		return bytesPerF32
	default:
		return 0
	}
}

// DecodeToFloat64 converts raw PCM bytes to mono float64 samples.
// Output values are in int16 scale [-32768, 32767] to match the resampler's expected range.
// Multi-channel input is averaged (equal-power downmix).
// The input data length must be aligned to frame boundaries (bytesPerSample * channels).
func DecodeToFloat64(data []byte, format Format, channels int) []float64 {
	bps := format.BytesPerSample()
	bytesPerFrame := bps * channels

	if bytesPerFrame == 0 || len(data) < bytesPerFrame {
		return nil
	}

	numFrames := len(data) / bytesPerFrame
	out := make([]float64, numFrames)

	switch format {
	case FormatS16LE:
		decodeS16LE(data, out, channels, numFrames)
	case FormatS32LE:
		decodeS32LE(data, out, channels, numFrames)
	case FormatF32LE:
		decodeF32LE(data, out, channels, numFrames)
	default:
		return nil
	}

	return out
}

func decodeS16LE(data []byte, out []float64, channels, numFrames int) {
	if channels == 1 {
		for frame := range numFrames {
			offset := frame * bytesPerS16
			raw := binary.LittleEndian.Uint16(data[offset:])

			out[frame] = float64(int16(raw)) //nolint:gosec // PCM decode
		}

		return
	}

	invCh := float64(1) / float64(channels)

	for frame := range numFrames {
		var sum float64

		base := frame * channels * bytesPerS16
		for ch := range channels {
			offset := base + ch*bytesPerS16
			raw := binary.LittleEndian.Uint16(data[offset:])

			sum += float64(int16(raw)) //nolint:gosec // PCM decode
		}

		out[frame] = sum * invCh
	}
}

const s32ToS16Scale = 1.0 / 65536.0

func decodeS32LE(data []byte, out []float64, channels, numFrames int) {
	if channels == 1 {
		for frame := range numFrames {
			offset := frame * bytesPerS32
			//nolint:gosec // intentional uint32 → int32 reinterpret for PCM decoding
			sample := int32(binary.LittleEndian.Uint32(data[offset:]))

			out[frame] = float64(sample) * s32ToS16Scale
		}

		return
	}

	invCh := s32ToS16Scale / float64(channels)

	for frame := range numFrames {
		var sum float64

		base := frame * channels * bytesPerS32
		for ch := range channels {
			offset := base + ch*bytesPerS32
			//nolint:gosec // intentional uint32 → int32 reinterpret for PCM decoding
			sample := int32(binary.LittleEndian.Uint32(data[offset:]))

			sum += float64(sample)
		}

		out[frame] = sum * invCh
	}
}

const f32ToS16Scale = 32767.0

func decodeF32LE(data []byte, out []float64, channels, numFrames int) {
	if channels == 1 {
		for frame := range numFrames {
			offset := frame * bytesPerF32
			sample := math.Float32frombits(binary.LittleEndian.Uint32(data[offset:]))

			out[frame] = float64(sample) * f32ToS16Scale
		}

		return
	}

	invCh := f32ToS16Scale / float64(channels)

	for frame := range numFrames {
		var sum float64

		base := frame * channels * bytesPerF32
		for ch := range channels {
			offset := base + ch*bytesPerF32
			sample := math.Float32frombits(binary.LittleEndian.Uint32(data[offset:]))

			sum += float64(sample)
		}

		out[frame] = sum * invCh
	}
}
