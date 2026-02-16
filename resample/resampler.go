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

import "math"

// Resampler performs streaming polyphase sinc resampling from an arbitrary
// source sample rate to 11025 Hz. It maintains internal state between Write
// calls for seamless chunked processing with minimal memory overhead.
//
// Source data must be mono float64 in int16 scale [-32768, 32767].
// Use [DecodeToFloat64] to convert raw PCM bytes to this format.
type Resampler struct {
	filterLength int
	bank         [][]float64
	srcRate      int
	srcIncr      int64
	dstIncrDiv   int64
	dstIncrMod   int64

	// Persistent state between chunks.
	buffer   []float64
	index    int64
	frac     int64
	totalSrc int64
	totalOut int64
}

// NewResampler creates a streaming resampler for the given source sample rate.
// Returns an error if srcRate is invalid. Returns nil if srcRate == 11025
// (no resampling needed).
func NewResampler(srcRate int) (*Resampler, error) {
	if srcRate <= 0 {
		return nil, ErrInvalidRate
	}

	if srcRate == targetRate {
		return nil, nil //nolint:nilnil // nil resampler = passthrough
	}

	factor := min(float64(targetRate)*cutoff/float64(srcRate), 1)

	filterLength := max(int(math.Ceil(float64(filterSize)/factor)), 1)

	if filterLength > 1 && filterLength%2 != 0 {
		filterLength++
	}

	bank := buildFilterBank(factor, filterLength, phaseCount, kaiserBeta)

	srcIncr, dstIncr := reduce(int64(targetRate), int64(srcRate)*int64(phaseCount))

	return &Resampler{
		filterLength: filterLength,
		bank:         bank,
		srcRate:      srcRate,
		srcIncr:      srcIncr,
		dstIncrDiv:   dstIncr / srcIncr,
		dstIncrMod:   dstIncr % srcIncr,
		index:        int64(-phaseCount) * int64((filterLength-1)/2),
	}, nil
}

// Write accepts mono float64 samples and returns resampled int16 output.
// It buffers the minimum necessary overlap (filterLength-1 samples) between calls.
// Returns nil when insufficient data has accumulated to produce output.
func (r *Resampler) Write(samples []float64) []int16 {
	if len(samples) == 0 {
		return nil
	}

	r.totalSrc += int64(len(samples))
	r.buffer = append(r.buffer, samples...)

	return r.process(false)
}

// Flush signals end of input and processes remaining buffered samples
// with zero-padding (matching swr's final flush behavior).
// Must be called exactly once after all Write calls.
func (r *Resampler) Flush() []int16 {
	if r.totalSrc == 0 {
		return nil
	}

	return r.process(true)
}

//nolint:revive // isFinal distinguishes Write vs Flush processing modes
func (r *Resampler) process(isFinal bool) []int16 {
	bufLen := int64(len(r.buffer))

	// Output count cap matching resampleLinear: numSrc * targetRate / srcRate.
	outCap := r.totalSrc * int64(targetRate) / int64(r.srcRate)

	var out []int16

	for r.totalOut+int64(len(out)) < outCap {
		sampleIndex, phase := divMod(r.index, int64(phaseCount))

		// In normal mode: stop when filter extends past available data.
		if !isFinal && sampleIndex+int64(r.filterLength) > bufLen {
			break
		}

		// In final mode: stop when we've consumed all input.
		if isFinal && sampleIndex >= bufLen {
			break
		}

		// Convolve with current and next filter phase.
		filterCur := r.bank[phase]
		filterNxt := r.bank[phase+1]

		var val, valNext float64

		for tap := range r.filterLength {
			srcIdx := sampleIndex + int64(tap)

			var sample float64
			if srcIdx >= 0 && srcIdx < bufLen {
				sample = r.buffer[srcIdx]
			}

			val += sample * filterCur[tap]
			valNext += sample * filterNxt[tap]
		}

		// Linear interpolation between phases.
		val += (valNext - val) * (float64(r.frac) / float64(r.srcIncr))

		// Convert to int16 with banker's rounding (matching swr's lrintf).
		out = append(out, clipInt16(math.RoundToEven(val)))

		// Advance phase.
		r.frac += r.dstIncrMod
		r.index += r.dstIncrDiv

		if r.frac >= r.srcIncr {
			r.frac -= r.srcIncr
			r.index++
		}
	}

	r.totalOut += int64(len(out))

	// Remove consumed samples, keeping overlap for next call.
	if !isFinal {
		r.compact()
	}

	return out
}

// compact removes consumed samples from the buffer and adjusts the index.
func (r *Resampler) compact() {
	sampleIndex, _ := divMod(r.index, int64(phaseCount))

	consumed := int(sampleIndex)
	if consumed <= 0 || consumed >= len(r.buffer) {
		return
	}

	copy(r.buffer, r.buffer[consumed:])
	r.buffer = r.buffer[:len(r.buffer)-consumed]
	r.index -= int64(consumed) * int64(phaseCount)
}

func clipInt16(val float64) int16 {
	switch {
	case val > math.MaxInt16:
		return math.MaxInt16
	case val < math.MinInt16:
		return math.MinInt16
	default:
		return int16(val)
	}
}
