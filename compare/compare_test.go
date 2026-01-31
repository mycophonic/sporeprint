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

package compare_test

import (
	"testing"

	"github.com/mycophonic/sporeprint/chromaprint"
	"github.com/mycophonic/sporeprint/compare"
)

// generateFingerprint creates a real encoded fingerprint from a deterministic
// PCM signal defined by seed. Note: the seed is additive, so different seeds
// produce phase-shifted versions of the same waveform. Chromaprint operates
// on spectral features, so phase-shifted signals yield identical fingerprints.
// Use generateDistinctFingerprint for spectrally different signals.
func generateFingerprint(t *testing.T, seed, numSeconds int) string {
	t.Helper()

	ctx := chromaprint.New()
	defer ctx.Free()

	if err := ctx.Start(11025, 1); err != nil {
		t.Fatalf("Start() failed: %v", err)
	}

	samples := make([]int16, 11025*numSeconds)
	for i := range samples {
		samples[i] = int16(((i + seed) * 17) % 65536)
	}

	if err := ctx.Feed(samples); err != nil {
		t.Fatalf("Feed() failed: %v", err)
	}

	if err := ctx.Finish(); err != nil {
		t.Fatalf("Finish() failed: %v", err)
	}

	fp, err := ctx.Fingerprint()
	if err != nil {
		t.Fatalf("Fingerprint() failed: %v", err)
	}

	return fp
}

// generateDistinctFingerprint creates a real encoded fingerprint with
// spectrally distinct content controlled by multiplier. Different multipliers
// change the frequency of the generated waveform, producing genuinely
// different Chromaprint fingerprints (unlike phase shifts from additive seeds).
func generateDistinctFingerprint(t *testing.T, multiplier, numSeconds int) string {
	t.Helper()

	ctx := chromaprint.New()
	defer ctx.Free()

	if err := ctx.Start(11025, 1); err != nil {
		t.Fatalf("Start() failed: %v", err)
	}

	samples := make([]int16, 11025*numSeconds)
	for i := range samples {
		samples[i] = int16((i * multiplier) % 65536)
	}

	if err := ctx.Feed(samples); err != nil {
		t.Fatalf("Feed() failed: %v", err)
	}

	if err := ctx.Finish(); err != nil {
		t.Fatalf("Finish() failed: %v", err)
	}

	fp, err := ctx.Fingerprint()
	if err != nil {
		t.Fatalf("Fingerprint() failed: %v", err)
	}

	return fp
}

func TestCompareIdentical(t *testing.T) {
	t.Parallel()

	fp := generateFingerprint(t, 0, 3)

	score, err := compare.Compare(fp, fp)
	if err != nil {
		t.Fatalf("Compare() failed: %v", err)
	}

	if score != 1.0 {
		t.Errorf("identical fingerprints should have score 1.0, got %f", score)
	}
}

func TestCompareEmpty(t *testing.T) {
	t.Parallel()

	_, err := compare.Compare("", "something")
	if err == nil {
		t.Error("Compare with empty fp1 should return error")
	}

	_, err = compare.Compare("something", "")
	if err == nil {
		t.Error("Compare with empty fp2 should return error")
	}
}

func TestCompareInvalidEncoding(t *testing.T) {
	t.Parallel()

	_, err := compare.Compare("not-a-fingerprint!!!", "also-invalid!!!")
	if err == nil {
		t.Error("Compare with invalid encoding should return error")
	}
}

// TestComparePhaseShiftedSignal verifies that phase-shifted versions of the
// same waveform are correctly identified as matching. Chromaprint operates on
// spectral features, so an additive seed offset does not change the frequency
// content — the fingerprints should score high.
func TestComparePhaseShiftedSignal(t *testing.T) {
	t.Parallel()

	fp1 := generateFingerprint(t, 0, 3)
	fp2 := generateFingerprint(t, 100000, 3)

	score, err := compare.Compare(fp1, fp2)
	if err != nil {
		t.Fatalf("Compare() failed: %v", err)
	}

	if score < 0.7 {
		t.Errorf("phase-shifted same signal should have high score, got %f", score)
	}
}

func TestCompareDifferentTracks(t *testing.T) {
	t.Parallel()

	fp1 := generateDistinctFingerprint(t, 17, 3)
	fp2 := generateDistinctFingerprint(t, 233, 3)

	score, err := compare.Compare(fp1, fp2)
	if err != nil {
		t.Fatalf("Compare() failed: %v", err)
	}

	if score > 0.3 {
		t.Errorf("different tracks should have low score, got %f", score)
	}
}

func TestWithOffsetIdentical(t *testing.T) {
	t.Parallel()

	fp := generateFingerprint(t, 0, 3)

	score, offset, err := compare.WithOffset(fp, fp)
	if err != nil {
		t.Fatalf("WithOffset() failed: %v", err)
	}

	if score != 1.0 {
		t.Errorf("identical fingerprints should have score 1.0, got %f", score)
	}

	if offset != 0 {
		t.Errorf("identical fingerprints should have offset 0, got %d", offset)
	}
}

func TestBitErrorRateIdentical(t *testing.T) {
	t.Parallel()

	fp := generateFingerprint(t, 0, 3)

	ber, err := compare.BitErrorRate(fp, fp, 0)
	if err != nil {
		t.Fatalf("BitErrorRate() failed: %v", err)
	}

	if ber != 0.0 {
		t.Errorf("identical fingerprints should have BER 0.0, got %f", ber)
	}
}

func TestBitErrorRateInvalid(t *testing.T) {
	t.Parallel()

	_, err := compare.BitErrorRate("invalid!!!", "also-invalid!!!", 0)
	if err == nil {
		t.Error("BitErrorRate with invalid encoding should return error")
	}
}

func TestBitErrorRateNoOverlap(t *testing.T) {
	t.Parallel()

	fp := generateFingerprint(t, 0, 3)

	ber, err := compare.BitErrorRate(fp, fp, 100000)
	if err != nil {
		t.Fatalf("BitErrorRate() failed: %v", err)
	}

	if ber != 1.0 {
		t.Errorf("non-overlapping fingerprints should have BER 1.0, got %f", ber)
	}
}

func TestIsSameTrack(t *testing.T) {
	t.Parallel()

	fp := generateFingerprint(t, 0, 3)

	same, err := compare.IsSameTrack(fp, fp, 0.5)
	if err != nil {
		t.Fatalf("IsSameTrack() failed: %v", err)
	}

	if !same {
		t.Error("identical fingerprints should be same track at threshold 0.5")
	}
}

func TestIsSameTrackImpossibleThreshold(t *testing.T) {
	t.Parallel()

	fp := generateFingerprint(t, 0, 3)

	same, err := compare.IsSameTrack(fp, fp, 1.1)
	if err != nil {
		t.Fatalf("IsSameTrack() failed: %v", err)
	}

	if same {
		t.Error("threshold above 1.0 should never match")
	}
}

func TestIsSameTrackInvalid(t *testing.T) {
	t.Parallel()

	_, err := compare.IsSameTrack("invalid!!!", "also-invalid!!!", 0.5)
	if err == nil {
		t.Error("IsSameTrack with invalid encoding should return error")
	}
}

func BenchmarkCompare(b *testing.B) {
	// Generate two real fingerprints for benchmarking.
	ctx1 := chromaprint.New()
	defer ctx1.Free()

	ctx2 := chromaprint.New()
	defer ctx2.Free()

	multipliers := [2]int{17, 233}

	for i, ctx := range []*chromaprint.Context{ctx1, ctx2} {
		if err := ctx.Start(11025, 1); err != nil {
			b.Fatalf("Start() failed: %v", err)
		}

		samples := make([]int16, 11025*10)
		for j := range samples {
			samples[j] = int16((j * multipliers[i]) % 65536)
		}

		if err := ctx.Feed(samples); err != nil {
			b.Fatalf("Feed() failed: %v", err)
		}

		if err := ctx.Finish(); err != nil {
			b.Fatalf("Finish() failed: %v", err)
		}
	}

	fp1, err := ctx1.Fingerprint()
	if err != nil {
		b.Fatalf("Fingerprint() failed: %v", err)
	}

	fp2, err := ctx2.Fingerprint()
	if err != nil {
		b.Fatalf("Fingerprint() failed: %v", err)
	}

	for b.Loop() {
		_, _ = compare.Compare(fp1, fp2)
	}
}
