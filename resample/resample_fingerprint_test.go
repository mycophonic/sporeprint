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
	"fmt"
	"math"
	"math/bits"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/mycophonic/sporeprint/chromaprint"
	"github.com/mycophonic/sporeprint/resample"
)

// TestFingerprintMatchesFFmpeg verifies that resampling with our internal resampler
// produces the same Chromaprint fingerprint as resampling with ffmpeg's aresample filter.
//
// This is the critical end-to-end validation: even if individual samples differ slightly,
// the fingerprint (which operates on energy features over large windows) must be identical.
func TestFingerprintMatchesFFmpeg(t *testing.T) {
	t.Parallel()

	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not found")
	}

	testCases := []struct {
		name       string
		sampleRate int
		channels   int
		freq       float64
		duration   float64
	}{
		{"44100Hz_mono_440Hz", 44100, 1, 440.0, 30.0},
		{"48000Hz_mono_440Hz", 48000, 1, 440.0, 30.0},
		{"44100Hz_stereo_440Hz", 44100, 2, 440.0, 30.0},
		{"96000Hz_mono_440Hz", 96000, 1, 440.0, 30.0},
		{"22050Hz_mono_440Hz", 22050, 1, 440.0, 30.0},
		{"8000Hz_mono_440Hz", 8000, 1, 440.0, 30.0},
		{"44100Hz_mono_mixed", 44100, 1, 0, 30.0},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			compareFingerprintsWithFFmpeg(t, tc.sampleRate, tc.channels, tc.freq, tc.duration)
		})
	}
}

//nolint:thelper // this is the test body, not a helper
func compareFingerprintsWithFFmpeg(
	t *testing.T,
	sampleRate, channels int,
	freq, duration float64,
) {
	// Generate source PCM.
	srcSamples := generateTestSignal(sampleRate, channels, freq, duration)

	// Path A: ffmpeg resamples to 11025Hz mono, then we fingerprint.
	ffmpegFP := fingerprintViaFFmpeg(t, srcSamples, sampleRate, channels)

	// Path B: internal resampler to 11025Hz mono, then fingerprint.
	goSamples, err := resample.Resample(srcSamples, sampleRate, channels)
	if err != nil {
		t.Fatalf("resample: %v", err)
	}

	goFP := fingerprintSamples(t, goSamples)

	// Compare fingerprints.
	if ffmpegFP == goFP {
		t.Logf("fingerprints match exactly (%d chars)", len(goFP))

		return
	}

	// If not exact, compare decoded sub-fingerprints for similarity.
	ffmpegRaw, err := chromaprint.Decode(ffmpegFP)
	if err != nil {
		t.Fatalf("decode ffmpeg fingerprint: %v", err)
	}

	goRaw, err := chromaprint.Decode(goFP)
	if err != nil {
		t.Fatalf("decode go fingerprint: %v", err)
	}

	t.Logf("fingerprint lengths: ffmpeg=%d, go=%d sub-fingerprints", len(ffmpegRaw), len(goRaw))

	compareLen := min(len(ffmpegRaw), len(goRaw))
	if compareLen == 0 {
		t.Fatal("empty fingerprint")
	}

	var matchCount int

	var totalBitErrors int

	for idx := range compareLen {
		if ffmpegRaw[idx] == goRaw[idx] {
			matchCount++
		}

		totalBitErrors += bits.OnesCount32(ffmpegRaw[idx] ^ goRaw[idx])
	}

	matchPct := float64(matchCount) / float64(compareLen) * 100
	bitErrorRate := float64(totalBitErrors) / float64(compareLen*32) * 100

	t.Logf("sub-fingerprint match: %d/%d (%.1f%%)", matchCount, compareLen, matchPct)
	t.Logf("bit error rate: %.2f%%", bitErrorRate)

	if matchPct < 100 {
		t.Errorf("fingerprints differ: %.1f%% sub-fingerprint match, %.2f%% bit error rate",
			matchPct, bitErrorRate)
	}
}

// fingerprintViaFFmpeg resamples PCM through ffmpeg and returns the chromaprint fingerprint.
func fingerprintViaFFmpeg(t *testing.T, samples []int16, sampleRate, channels int) string {
	t.Helper()

	tmpDir := t.TempDir()

	srcBytes := samplesToBytes(samples)

	srcPath := filepath.Join(tmpDir, "source.raw")
	if err := os.WriteFile(srcPath, srcBytes, 0o600); err != nil {
		t.Fatalf("write source: %v", err)
	}

	outPath := filepath.Join(tmpDir, "resampled.raw")

	cmd := exec.Command("ffmpeg",
		"-f", "s16le",
		"-ar", itoa(sampleRate),
		"-ac", itoa(channels),
		"-i", srcPath,
		"-af", aresampleFilter,
		"-f", "s16le",
		"-ac", "1",
		"-ar", itoa(targetRate),
		"-y", outPath,
	)

	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("ffmpeg failed: %v\n%s", err, output)
	}

	outBytes, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read ffmpeg output: %v", err)
	}

	ffmpegSamples := bytesToSamples(outBytes)

	return fingerprintSamples(t, ffmpegSamples)
}

// fingerprintSamples feeds mono 11025Hz int16 samples to chromaprint and returns the fingerprint.
func fingerprintSamples(t *testing.T, samples []int16) string {
	t.Helper()

	ctx := chromaprint.New()
	defer ctx.Free()

	if err := ctx.Start(targetRate, 1); err != nil {
		t.Fatalf("chromaprint start: %v", err)
	}

	if err := ctx.Feed(samples); err != nil {
		t.Fatalf("chromaprint feed: %v", err)
	}

	if err := ctx.Finish(); err != nil {
		t.Fatalf("chromaprint finish: %v", err)
	}

	fp, err := ctx.Fingerprint()
	if err != nil {
		t.Fatalf("chromaprint fingerprint: %v", err)
	}

	return fp
}

// generateTestSignal creates a test signal. If freq is 0, generates a multi-frequency signal.
func generateTestSignal(sampleRate, channels int, freq, seconds float64) []int16 {
	numFrames := int(float64(sampleRate) * seconds)
	out := make([]int16, numFrames*channels)

	for frame := range numFrames {
		timePos := float64(frame) / float64(sampleRate)

		var val float64
		if freq > 0 {
			val = math.Sin(2.0*math.Pi*freq*timePos) * 0.9 * math.MaxInt16
		} else {
			// Multi-frequency signal: chord of 261Hz (C4), 329Hz (E4), 392Hz (G4).
			val = (math.Sin(2.0*math.Pi*261.63*timePos) +
				math.Sin(2.0*math.Pi*329.63*timePos) +
				math.Sin(2.0*math.Pi*392.00*timePos)) / 3.0 * 0.9 * math.MaxInt16
		}

		sample := int16(val)

		for ch := range channels {
			out[frame*channels+ch] = sample
		}
	}

	return out
}

func itoa(val int) string {
	return fmt.Sprintf("%d", val)
}
