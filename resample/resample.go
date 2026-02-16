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
	"errors"
	"math"
)

// Chromaprint-compatible resampling parameters matching fpcalc's SetCompatibleMode().
// These reproduce ffmpeg's aresample filter:
//
//	aresample=resampler=swr:filter_size=16:phase_shift=8:cutoff=0.8:linear_interp=1
const (
	targetRate = 11025
	filterSize = 16
	phaseShift = 8
	phaseCount = 1 << phaseShift // 256 polyphase banks.
	cutoff     = 0.8
	kaiserBeta = 9.0
)

var (
	// ErrInvalidRate is returned when the source sample rate is invalid.
	ErrInvalidRate = errors.New("resample: invalid sample rate")
	// ErrInvalidChannels is returned when the channel count is invalid.
	ErrInvalidChannels = errors.New("resample: invalid channel count")
)

// Resample converts interleaved PCM samples from the source format to
// 11025 Hz mono signed 16-bit samples, matching ffmpeg's aresample filter
// with parameters: filter_size=16, phase_shift=8, cutoff=0.8, linear_interp=1.
//
// Input: interleaved int16 samples at srcRate Hz with srcChannels channels.
// Output: mono int16 samples at 11025 Hz.
func Resample(samples []int16, srcRate, srcChannels int) ([]int16, error) {
	if srcRate <= 0 {
		return nil, ErrInvalidRate
	}

	if srcChannels <= 0 {
		return nil, ErrInvalidChannels
	}

	if len(samples) == 0 {
		return nil, nil
	}

	// Downmix to mono float64.
	mono := downmixToMono(samples, srcChannels)

	// Identity rate: convert directly to int16.
	if srcRate == targetRate {
		return FloatToInt16(mono), nil
	}

	// Resample via streaming Resampler in a single pass.
	resampler, err := NewResampler(srcRate)
	if err != nil {
		return nil, err
	}

	out := resampler.Write(mono)

	return append(out, resampler.Flush()...), nil
}

// downmixToMono converts interleaved int16 samples to mono float64.
// For mono input, samples are simply converted to float64.
// For multi-channel, channels are averaged (matching ffmpeg's equal-power downmix).
func downmixToMono(samples []int16, channels int) []float64 {
	if channels == 1 {
		out := make([]float64, len(samples))
		for idx, sample := range samples {
			out[idx] = float64(sample)
		}

		return out
	}

	frames := len(samples) / channels
	out := make([]float64, frames)

	for frame := range frames {
		var sum float64

		base := frame * channels
		for channel := range channels {
			sum += float64(samples[base+channel])
		}

		out[frame] = sum / float64(channels)
	}

	return out
}

// FloatToInt16 converts float64 samples to int16 with clamping.
// Uses RoundToEven to match swr's lrintf (banker's rounding).
func FloatToInt16(src []float64) []int16 {
	out := make([]int16, len(src))

	for idx, val := range src {
		rounded := math.RoundToEven(val)

		switch {
		case rounded > math.MaxInt16:
			out[idx] = math.MaxInt16
		case rounded < math.MinInt16:
			out[idx] = math.MinInt16
		default:
			out[idx] = int16(rounded)
		}
	}

	return out
}

// reduce computes the greatest common divisor and returns numerator/gcd, denominator/gcd.
// This matches ffmpeg's av_reduce for reducing rate fractions.
func reduce(numerator, denominator int64) (reduced, reducedDenom int64) {
	divisor := gcd(numerator, denominator)
	if divisor == 0 {
		return numerator, denominator
	}

	return numerator / divisor, denominator / divisor
}

// gcd computes the greatest common divisor using the Euclidean algorithm.
func gcd(first, second int64) int64 {
	if first < 0 {
		first = -first
	}

	if second < 0 {
		second = -second
	}

	for second != 0 {
		first, second = second, first%second
	}

	return first
}

// divMod performs floor division and modulo for potentially negative dividends.
// Go's % operator truncates toward zero; we need floor semantics to correctly
// handle negative index values (filter delay region).
func divMod(dividend, divisor int64) (quotient, remainder int64) {
	quotient = dividend / divisor
	remainder = dividend % divisor

	if remainder < 0 {
		quotient--
		remainder += divisor
	}

	return quotient, remainder
}
