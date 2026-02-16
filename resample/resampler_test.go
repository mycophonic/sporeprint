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

package resample_test

import (
	"testing"

	"github.com/mycophonic/sporeprint/resample"
)

// TestResamplerMatchesSinglePass verifies that the streaming Resampler produces
// identical output to the all-at-once Resample() function.
func TestResamplerMatchesSinglePass(t *testing.T) {
	t.Parallel()

	rates := []int{8000, 16000, 22050, 44100, 48000, 96000}

	for _, rate := range rates {
		t.Run(itoa(rate), func(t *testing.T) {
			t.Parallel()

			srcInt16 := generateSine(rate, 440.0, 2.0)

			// Single-pass via Resample().
			singlePass, err := resample.Resample(srcInt16, rate, 1)
			if err != nil {
				t.Fatalf("Resample: %v", err)
			}

			// Convert to float64 for streaming Resampler (matches downmixToMono output).
			srcF64 := int16ToFloat64(srcInt16)

			// Streaming via Resampler with 500-sample chunks.
			resampler, err := resample.NewResampler(rate)
			if err != nil {
				t.Fatalf("NewResampler: %v", err)
			}

			var streamed []int16

			chunkSize := 500
			for idx := 0; idx < len(srcF64); idx += chunkSize {
				end := min(idx+chunkSize, len(srcF64))

				out := resampler.Write(srcF64[idx:end])
				streamed = append(streamed, out...)
			}

			streamed = append(streamed, resampler.Flush()...)

			// Compare.
			if len(singlePass) != len(streamed) {
				t.Fatalf("length mismatch: single=%d, streamed=%d", len(singlePass), len(streamed))
			}

			var diffCount, maxDiff int

			for idx := range singlePass {
				diff := int(singlePass[idx]) - int(streamed[idx])
				if diff < 0 {
					diff = -diff
				}

				if diff > 0 {
					diffCount++

					if diff > maxDiff {
						maxDiff = diff
					}
				}
			}

			if diffCount > 0 {
				t.Errorf("%dHz: %d diffs out of %d, max=%d", rate, diffCount, len(singlePass), maxDiff)
			}
		})
	}
}

// TestResamplerVaryingChunkSizes verifies that chunk size does not affect output.
func TestResamplerVaryingChunkSizes(t *testing.T) {
	t.Parallel()

	src := int16ToFloat64(generateSine(44100, 440.0, 1.0))
	chunkSizes := []int{1, 50, 100, 500, 4096}

	// Reference: use a large chunk (entire input).
	ref, err := resample.NewResampler(44100)
	if err != nil {
		t.Fatalf("NewResampler: %v", err)
	}

	var refOut []int16

	refOut = append(refOut, ref.Write(src)...)
	refOut = append(refOut, ref.Flush()...)

	for _, cs := range chunkSizes {
		t.Run(itoa(cs), func(t *testing.T) {
			t.Parallel()

			resampler, err := resample.NewResampler(44100)
			if err != nil {
				t.Fatalf("NewResampler: %v", err)
			}

			var out []int16

			for idx := 0; idx < len(src); idx += cs {
				end := min(idx+cs, len(src))

				out = append(out, resampler.Write(src[idx:end])...)
			}

			out = append(out, resampler.Flush()...)

			if len(out) != len(refOut) {
				t.Fatalf("chunk=%d: length mismatch: got %d, want %d", cs, len(out), len(refOut))
			}

			for idx := range out {
				if out[idx] != refOut[idx] {
					t.Errorf("chunk=%d: sample %d differs: got %d, want %d",
						cs, idx, out[idx], refOut[idx])

					break
				}
			}
		})
	}
}

// TestResamplerEmptyWrite verifies Write with empty input doesn't panic.
func TestResamplerEmptyWrite(t *testing.T) {
	t.Parallel()

	resampler, err := resample.NewResampler(44100)
	if err != nil {
		t.Fatalf("NewResampler: %v", err)
	}

	got := resampler.Write(nil)
	if got != nil {
		t.Errorf("empty write: got %v, want nil", got)
	}

	got = resampler.Write([]float64{})
	if got != nil {
		t.Errorf("zero-length write: got %v, want nil", got)
	}

	// Flush should still work after empty writes.
	flush := resampler.Flush()
	if flush != nil {
		t.Errorf("flush after empty writes: got %v, want nil", flush)
	}
}

// TestResamplerNilPassthrough verifies NewResampler returns nil for identity rate.
func TestResamplerNilPassthrough(t *testing.T) {
	t.Parallel()

	resampler, err := resample.NewResampler(11025)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resampler != nil {
		t.Error("expected nil resampler for identity rate")
	}
}

// TestResamplerSmallInput verifies behavior when input is smaller than filter length.
func TestResamplerSmallInput(t *testing.T) {
	t.Parallel()

	resampler, err := resample.NewResampler(44100)
	if err != nil {
		t.Fatalf("NewResampler: %v", err)
	}

	// Feed just 10 samples (filter length is 80 for 44100→11025).
	small := make([]float64, 10)
	for idx := range small {
		small[idx] = float64(idx * 1000)
	}

	got := resampler.Write(small)
	// May produce 0 output since filter needs more data.
	t.Logf("small write: %d output samples", len(got))

	flush := resampler.Flush()
	t.Logf("flush: %d output samples", len(flush))

	// Should produce some output from flush.
	total := len(got) + len(flush)
	if total == 0 {
		t.Error("expected some output from flush")
	}
}

// int16ToFloat64 converts int16 samples to float64 (matching downmixToMono's output).
func int16ToFloat64(src []int16) []float64 {
	out := make([]float64, len(src))

	for idx, val := range src {
		out[idx] = float64(val)
	}

	return out
}
