package tests_test

import (
	"path/filepath"
	"testing"

	"github.com/containerd/nerdctl/mod/tigron/test"

	"github.com/mycophonic/agar/pkg/agar"

	"github.com/mycophonic/sporeprint/tests/testutils"
)

// nolint:paralleltest
func TestFingerprintMatchesFpcalc(t *testing.T) {
	testCase := testutils.Setup()

	testCase.SubTests = []*test.Case{
		fingerprintSubtest("FLAC 16-bit 44.1kHz stereo", agar.Genuine16bit44k),
		fingerprintSubtest("FLAC 24-bit 96kHz stereo", agar.Genuine24bit96k),
		fingerprintSubtest("FLAC 24-bit 48kHz stereo", agar.Genuine24bit48k),
		fingerprintSubtest("FLAC mono 16-bit 44.1kHz", agar.GenuineMono16bit44k),
		fingerprintSubtest("MP3 320k", agar.FormatMP3320k),
		fingerprintSubtest("MP3 96k", agar.FormatMP396k),
		fingerprintSubtest("AAC 256k", agar.FormatAAC256k),
		fingerprintSubtest("AAC 64k", agar.FormatAAC64k),
		fingerprintSubtest("OGG Vorbis", agar.FormatOggVorbis),
		fingerprintSubtest("ALAC", agar.FormatALAC),
	}

	testCase.Run(t)
}

type audioGenerator func(test.Data, test.Helpers) string

func fingerprintSubtest(description string, gen audioGenerator) *test.Case {
	return &test.Case{
		Description: description,
		Setup: func(data test.Data, helpers test.Helpers) {
			// Generate audio file.
			audioFile := gen(data, helpers)
			data.Labels().Set("audio", audioFile)

			// Preprocess to chromaprint-compatible PCM.
			pcmFile := filepath.Join(data.Temp().Dir(), "preprocessed.pcm")
			testutils.PreprocessPCM(helpers, audioFile, pcmFile)
			data.Labels().Set("pcm", pcmFile)

			// 1. fpcalc direct on the original audio file.
			data.Labels().Set("fp-direct", testutils.FpcalcFingerprint(helpers.T(), audioFile))

			// 2. fpcalc on the preprocessed PCM.
			data.Labels().Set("fp-pcm", testutils.FpcalcFingerprintPCM(helpers.T(), pcmFile))
		},
		Command: func(data test.Data, helpers test.Helpers) test.TestableCommand {
			pcmFile := data.Labels().Get("pcm")

			// 3. sporeprint on the preprocessed PCM via stdin.
			fpSporeprint := testutils.SporeprintFingerprint(helpers.T(), pcmFile)

			fpDirect := data.Labels().Get("fp-direct")
			fpPCM := data.Labels().Get("fp-pcm")

			if fpDirect != fpSporeprint {
				helpers.T().Log("fpcalc direct vs sporeprint: MISMATCH")
				helpers.T().Log("  fpcalc:     " + fpDirect)
				helpers.T().Log("  sporeprint: " + fpSporeprint)
				helpers.T().Fail()
			}

			if fpPCM != fpSporeprint {
				helpers.T().Log("fpcalc PCM vs sporeprint: MISMATCH")
				helpers.T().Log("  fpcalc:     " + fpPCM)
				helpers.T().Log("  sporeprint: " + fpSporeprint)
				helpers.T().Fail()
			}

			return helpers.Custom("true")
		},
		Expected: test.Expects(0, nil, nil),
	}
}
