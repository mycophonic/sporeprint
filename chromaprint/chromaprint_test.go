package chromaprint_test

import (
	"errors"
	"testing"

	"github.com/mycophonic/sporeprint/chromaprint"
)

func TestVersion(t *testing.T) {
	t.Parallel()

	version := chromaprint.Version()
	if version == "" {
		t.Error("Version() returned empty string")
	}
}

func TestContextLifecycle(t *testing.T) {
	t.Parallel()

	ctx := chromaprint.New()
	if ctx == nil {
		t.Fatal("New() returned nil")
	}

	// Free should not panic
	ctx.Free()

	// Double Free should not panic
	ctx.Free()
}

func TestErrFreedAfterFree(t *testing.T) {
	t.Parallel()

	ctx := chromaprint.New()
	ctx.Free()

	// All methods should return ErrFreed after Free()
	t.Run("Start", func(t *testing.T) {
		t.Parallel()

		err := ctx.Start(11025, 1)
		if !errors.Is(err, chromaprint.ErrFreed) {
			t.Errorf("Start() after Free() = %v, want ErrFreed", err)
		}
	})

	t.Run("Feed", func(t *testing.T) {
		t.Parallel()

		err := ctx.Feed([]int16{0, 0, 0})
		if !errors.Is(err, chromaprint.ErrFreed) {
			t.Errorf("Feed() after Free() = %v, want ErrFreed", err)
		}
	})

	t.Run("Finish", func(t *testing.T) {
		t.Parallel()

		err := ctx.Finish()
		if !errors.Is(err, chromaprint.ErrFreed) {
			t.Errorf("Finish() after Free() = %v, want ErrFreed", err)
		}
	})

	t.Run("Fingerprint", func(t *testing.T) {
		t.Parallel()

		_, err := ctx.Fingerprint()
		if !errors.Is(err, chromaprint.ErrFreed) {
			t.Errorf("Fingerprint() after Free() = %v, want ErrFreed", err)
		}
	})
}

func TestFeedEmptyData(t *testing.T) {
	t.Parallel()

	ctx := chromaprint.New()
	defer ctx.Free()

	if err := ctx.Start(11025, 1); err != nil {
		t.Fatalf("Start() failed: %v", err)
	}

	// Empty slice should succeed without error
	if err := ctx.Feed([]int16{}); err != nil {
		t.Errorf("Feed(empty) = %v, want nil", err)
	}

	// Nil slice should also succeed
	if err := ctx.Feed(nil); err != nil {
		t.Errorf("Feed(nil) = %v, want nil", err)
	}
}

func TestBasicFingerprint(t *testing.T) {
	t.Parallel()

	ctx := chromaprint.New()
	defer ctx.Free()

	// Start with expected parameters (11025 Hz mono)
	if err := ctx.Start(11025, 1); err != nil {
		t.Fatalf("Start() failed: %v", err)
	}

	// Generate a test signal with some variation (not silence)
	// 3 seconds of pseudo-random audio to get meaningful fingerprint
	samples := make([]int16, 11025*3)
	for i := range samples {
		// Create a varying signal that's not pure silence
		samples[i] = int16(((i * 17) % 65536) - 32768)
	}

	if err := ctx.Feed(samples); err != nil {
		t.Fatalf("Feed() failed: %v", err)
	}

	if err := ctx.Finish(); err != nil {
		t.Fatalf("Finish() failed: %v", err)
	}

	fingerprint, err := ctx.Fingerprint()
	if err != nil {
		t.Fatalf("Fingerprint() failed: %v", err)
	}

	if fingerprint == "" {
		t.Error("Fingerprint() returned empty string")
	}

	// Chromaprint fingerprints are base64-encoded
	// With 3 seconds of non-silent audio, should get a reasonable fingerprint
	if len(fingerprint) < 10 {
		t.Errorf("Fingerprint too short: %q", fingerprint)
	}
}

func TestFingerprintDeterministic(t *testing.T) {
	t.Parallel()

	// Same input should produce same fingerprint
	samples := make([]int16, 11025)
	for i := range samples {
		// Simple sine-ish pattern for variation
		samples[i] = int16((i % 256) - 128)
	}

	var fingerprints [2]string

	for i := range 2 {
		ctx := chromaprint.New()

		if err := ctx.Start(11025, 1); err != nil {
			t.Fatalf("Start() failed: %v", err)
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

		fingerprints[i] = fp

		ctx.Free()
	}

	if fingerprints[0] != fingerprints[1] {
		t.Errorf("Fingerprints not deterministic:\n  first:  %s\n  second: %s", fingerprints[0], fingerprints[1])
	}
}

func TestMultipleFeeds(t *testing.T) {
	t.Parallel()

	ctx := chromaprint.New()
	defer ctx.Free()

	if err := ctx.Start(11025, 1); err != nil {
		t.Fatalf("Start() failed: %v", err)
	}

	// Feed in chunks
	chunk := make([]int16, 1024)
	for i := range 10 {
		// Vary the data slightly
		for j := range chunk {
			chunk[j] = int16(i*100 + j%100)
		}

		if err := ctx.Feed(chunk); err != nil {
			t.Fatalf("Feed() chunk %d failed: %v", i, err)
		}
	}

	if err := ctx.Finish(); err != nil {
		t.Fatalf("Finish() failed: %v", err)
	}

	fingerprint, err := ctx.Fingerprint()
	if err != nil {
		t.Fatalf("Fingerprint() failed: %v", err)
	}

	if fingerprint == "" {
		t.Error("Fingerprint() returned empty string after multiple feeds")
	}
}
