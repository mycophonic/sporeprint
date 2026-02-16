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
	"encoding/binary"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/mycophonic/sporeprint/resample"
)

const (
	aresampleFilter = "aresample=resampler=swr:filter_size=16:phase_shift=8:cutoff=0.8:linear_interp=1"
	targetRate      = 11025
)

// TestCompareWithFFmpeg generates PCM at various rates, resamples with both
// our Go implementation and ffmpeg's swr, and compares the results sample-by-sample.
func TestCompareWithFFmpeg(t *testing.T) {
	t.Parallel()

	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not found, skipping comparison test")
	}

	testCases := []struct {
		name       string
		sampleRate int
		channels   int
		freq       float64
		duration   float64
	}{
		{"44100Hz_mono_440Hz", 44100, 1, 440.0, 2.0},
		{"48000Hz_mono_440Hz", 48000, 1, 440.0, 2.0},
		{"44100Hz_stereo_440Hz", 44100, 2, 440.0, 2.0},
		{"96000Hz_mono_440Hz", 96000, 1, 440.0, 2.0},
		{"22050Hz_mono_440Hz", 22050, 1, 440.0, 2.0},
		{"44100Hz_mono_1000Hz", 44100, 1, 1000.0, 2.0},
		{"44100Hz_mono_100Hz", 44100, 1, 100.0, 2.0},
		{"8000Hz_mono_440Hz", 8000, 1, 440.0, 2.0},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			compareWithFFmpeg(t, tc.sampleRate, tc.channels, tc.freq, tc.duration)
		})
	}
}

//nolint:thelper // this is the test body, not a helper
func compareWithFFmpeg(
	t *testing.T,
	sampleRate, channels int,
	freq, duration float64,
) {
	tmpDir := t.TempDir()

	// Generate source PCM.
	srcSamples := generateStereoSine(sampleRate, channels, freq, duration)
	srcBytes := samplesToBytes(srcSamples)

	srcPath := filepath.Join(tmpDir, "source.raw")
	if err := os.WriteFile(srcPath, srcBytes, 0o600); err != nil {
		t.Fatalf("write source: %v", err)
	}

	// Run ffmpeg aresample.
	ffmpegOutPath := filepath.Join(tmpDir, "ffmpeg_out.raw")

	ffmpegArgs := []string{
		"-f", "s16le",
		"-ar", fmt.Sprintf("%d", sampleRate),
		"-ac", fmt.Sprintf("%d", channels),
		"-i", srcPath,
		"-af", aresampleFilter,
		"-f", "s16le",
		"-ac", "1",
		"-ar", fmt.Sprintf("%d", targetRate),
		"-y", ffmpegOutPath,
	}

	cmd := exec.Command("ffmpeg", ffmpegArgs...)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("ffmpeg failed: %v\n%s", err, output)
	}

	ffmpegBytes, err := os.ReadFile(ffmpegOutPath)
	if err != nil {
		t.Fatalf("read ffmpeg output: %v", err)
	}

	ffmpegSamples := bytesToSamples(ffmpegBytes)

	// Run our resampler.
	goSamples, err := resample.Resample(srcSamples, sampleRate, channels)
	if err != nil {
		t.Fatalf("resample: %v", err)
	}

	// Compare.
	reportComparison(t, goSamples, ffmpegSamples)
}

func reportComparison(t *testing.T, goSamples, ffmpegSamples []int16) {
	t.Helper()

	t.Logf("output lengths: go=%d, ffmpeg=%d (diff=%d)",
		len(goSamples), len(ffmpegSamples), len(goSamples)-len(ffmpegSamples))

	// Compare the overlapping region.
	compareLen := min(len(goSamples), len(ffmpegSamples))
	if compareLen == 0 {
		t.Error("no samples to compare")

		return
	}

	var (
		maxDiff   int
		totalDiff int64
		diffCount int
	)

	diffBuckets := make([]int, 7) // 0, 1, 2-5, 6-10, 11-50, 51-100, 100+

	for idx := range compareLen {
		diff := int(goSamples[idx]) - int(ffmpegSamples[idx])
		if diff < 0 {
			diff = -diff
		}

		if diff > 0 {
			diffCount++
			totalDiff += int64(diff)
		}

		if diff > maxDiff {
			maxDiff = diff
		}

		switch {
		case diff == 0:
			diffBuckets[0]++
		case diff == 1:
			diffBuckets[1]++
		case diff <= 5:
			diffBuckets[2]++
		case diff <= 10:
			diffBuckets[3]++
		case diff <= 50:
			diffBuckets[4]++
		case diff <= 100:
			diffBuckets[5]++
		default:
			diffBuckets[6]++
		}
	}

	avgDiff := float64(0)
	if diffCount > 0 {
		avgDiff = float64(totalDiff) / float64(diffCount)
	}

	pctExact := float64(diffBuckets[0]) / float64(compareLen) * 100
	pctClose := float64(diffBuckets[0]+diffBuckets[1]) / float64(compareLen) * 100

	t.Logf("compared %d samples:", compareLen)
	t.Logf("  exact match:  %d (%.1f%%)", diffBuckets[0], pctExact)
	t.Logf("  diff=1:       %d", diffBuckets[1])
	t.Logf("  diff=2-5:     %d", diffBuckets[2])
	t.Logf("  diff=6-10:    %d", diffBuckets[3])
	t.Logf("  diff=11-50:   %d", diffBuckets[4])
	t.Logf("  diff=51-100:  %d", diffBuckets[5])
	t.Logf("  diff>100:     %d", diffBuckets[6])
	t.Logf("  max diff:     %d", maxDiff)
	t.Logf("  avg diff (non-zero): %.2f", avgDiff)
	t.Logf("  within ±1:    %.1f%%", pctClose)
}

func generateStereoSine(sampleRate, channels int, freq, seconds float64) []int16 {
	numFrames := int(float64(sampleRate) * seconds)
	out := make([]int16, numFrames*channels)

	for frame := range numFrames {
		timePos := float64(frame) / float64(sampleRate)
		val := math.Sin(2.0*math.Pi*freq*timePos) * 0.9 * math.MaxInt16
		sample := int16(val)

		for ch := range channels {
			out[frame*channels+ch] = sample
		}
	}

	return out
}

func samplesToBytes(samples []int16) []byte {
	buf := make([]byte, len(samples)*2)

	for idx, sample := range samples {
		binary.LittleEndian.PutUint16(buf[idx*2:], uint16(sample))
	}

	return buf
}

func bytesToSamples(data []byte) []int16 {
	numSamples := len(data) / 2
	out := make([]int16, numSamples)

	for idx := range numSamples {
		out[idx] = int16(binary.LittleEndian.Uint16(data[idx*2:]))
	}

	return out
}
