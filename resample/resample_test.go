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
	"math"
	"testing"

	"github.com/mycophonic/sporeprint/resample"
)

func TestResampleEmptyInput(t *testing.T) {
	t.Parallel()

	out, err := resample.Resample(nil, 44100, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if out != nil {
		t.Fatalf("expected nil output, got %d samples", len(out))
	}
}

func TestResampleInvalidRate(t *testing.T) {
	t.Parallel()

	_, err := resample.Resample([]int16{1, 2, 3}, 0, 1)
	if err == nil {
		t.Fatal("expected error for rate=0")
	}

	_, err = resample.Resample([]int16{1, 2, 3}, -1, 1)
	if err == nil {
		t.Fatal("expected error for rate=-1")
	}
}

func TestResampleInvalidChannels(t *testing.T) {
	t.Parallel()

	_, err := resample.Resample([]int16{1, 2, 3}, 44100, 0)
	if err == nil {
		t.Fatal("expected error for channels=0")
	}
}

func TestResampleIdentityRate(t *testing.T) {
	t.Parallel()

	// 11025 Hz mono → 11025 Hz mono: output should match input.
	input := generateSine(11025, 440.0, 1.0)

	out, err := resample.Resample(input, 11025, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(out) != len(input) {
		t.Fatalf("length mismatch: input=%d, output=%d", len(input), len(out))
	}

	// Should be identical (no resampling path).
	for idx := range out {
		if out[idx] != input[idx] {
			t.Errorf("sample %d: got %d, want %d", idx, out[idx], input[idx])

			break
		}
	}
}

func TestResampleDownsample4to1(t *testing.T) {
	t.Parallel()

	// 44100 Hz → 11025 Hz (exact 4:1 ratio).
	input := generateSine(44100, 440.0, 1.0)

	out, err := resample.Resample(input, 44100, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Output should be approximately 1/4 the input length.
	expectedLen := len(input) / 4
	tolerance := expectedLen / 10 // 10% tolerance for filter delay.

	if abs(len(out)-expectedLen) > tolerance {
		t.Fatalf("output length %d far from expected %d", len(out), expectedLen)
	}

	// Verify output is not silence (should contain the 440 Hz tone).
	maxAmp := int16(0)

	for _, sample := range out {
		if sample > maxAmp {
			maxAmp = sample
		} else if -sample > maxAmp {
			maxAmp = -sample
		}
	}

	if maxAmp < 100 {
		t.Errorf("output appears silent: max amplitude = %d", maxAmp)
	}
}

func TestResampleDownsample48kTo11025(t *testing.T) {
	t.Parallel()

	// 48000 Hz → 11025 Hz (non-integer ratio).
	input := generateSine(48000, 440.0, 1.0)

	out, err := resample.Resample(input, 48000, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedLen := int(float64(len(input)) * 11025.0 / 48000.0)
	tolerance := expectedLen / 10

	if abs(len(out)-expectedLen) > tolerance {
		t.Fatalf("output length %d far from expected %d", len(out), expectedLen)
	}
}

func TestResampleStereoDownmix(t *testing.T) {
	t.Parallel()

	// Stereo 44100 Hz: left channel = sine, right channel = silence.
	monoSine := generateSine(44100, 440.0, 0.5)
	stereo := make([]int16, len(monoSine)*2)

	for idx, sample := range monoSine {
		stereo[idx*2] = sample // left
		stereo[idx*2+1] = 0    // right (silence)
	}

	out, err := resample.Resample(stereo, 44100, 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Output should be roughly half the amplitude of the mono input
	// (averaged with silence channel), at 11025 Hz.
	if len(out) == 0 {
		t.Fatal("output is empty")
	}

	maxAmp := int16(0)

	for _, sample := range out {
		if sample > maxAmp {
			maxAmp = sample
		} else if -sample > maxAmp {
			maxAmp = -sample
		}
	}

	// Should be audible but attenuated.
	if maxAmp < 50 {
		t.Errorf("stereo downmix output too quiet: max amplitude = %d", maxAmp)
	}
}

func TestResampleDeterminism(t *testing.T) {
	t.Parallel()

	input := generateSine(44100, 440.0, 1.0)

	out1, err := resample.Resample(input, 44100, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	out2, err := resample.Resample(input, 44100, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(out1) != len(out2) {
		t.Fatalf("non-deterministic output length: %d vs %d", len(out1), len(out2))
	}

	for idx := range out1 {
		if out1[idx] != out2[idx] {
			t.Fatalf("non-deterministic at sample %d: %d vs %d", idx, out1[idx], out2[idx])
		}
	}
}

func TestResampleMultipleRates(t *testing.T) {
	t.Parallel()

	rates := []int{8000, 11025, 16000, 22050, 32000, 44100, 48000, 88200, 96000}

	for _, rate := range rates {
		input := generateSine(rate, 440.0, 0.5)

		out, err := resample.Resample(input, rate, 1)
		if err != nil {
			t.Errorf("rate %d: unexpected error: %v", rate, err)

			continue
		}

		if len(out) == 0 {
			t.Errorf("rate %d: empty output", rate)
		}
	}
}

// generateSine creates a mono int16 sine wave at the given sample rate, frequency, and duration.
func generateSine(sampleRate int, freq, seconds float64) []int16 {
	numSamples := int(float64(sampleRate) * seconds)
	out := make([]int16, numSamples)

	for idx := range numSamples {
		t := float64(idx) / float64(sampleRate)
		val := math.Sin(2.0*math.Pi*freq*t) * 0.9 * math.MaxInt16
		out[idx] = int16(val)
	}

	return out
}

func abs(x int) int {
	if x < 0 {
		return -x
	}

	return x
}
