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

package tests_test

import (
	"path/filepath"
	"testing"

	"github.com/containerd/nerdctl/mod/tigron/test"

	"github.com/mycophonic/agar/pkg/agar"

	"github.com/mycophonic/sporeprint/tests/testutil"
)

//nolint:paralleltest
func TestFingerprintMatchesFpcalc(t *testing.T) {
	testCase := testutil.Setup()

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

// TestFingerprintWithInternalResample verifies that sporeprint produces identical
// fingerprints when resampling internally (--rate/--channels flags) vs fpcalc.
// ffmpeg is only used for decoding (no aresample filter).
//
//nolint:paralleltest
func TestFingerprintWithInternalResample(t *testing.T) {
	testCase := testutil.Setup()

	testCase.SubTests = []*test.Case{
		resampleSubtest("44.1kHz stereo s16le", agar.Genuine16bit44k, "44100", "2", "s16le"),
		resampleSubtest("96kHz stereo s32le", agar.Genuine24bit96k, "96000", "2", "s32le"),
		resampleSubtest("48kHz stereo s32le", agar.Genuine24bit48k, "48000", "2", "s32le"),
		resampleSubtest("44.1kHz mono s16le", agar.GenuineMono16bit44k, "44100", "1", "s16le"),
	}

	testCase.Run(t)
}

type audioGenerator func(test.Data, test.Helpers) string

func resampleSubtest(description string, gen audioGenerator, rate, channels, format string) *test.Case {
	return &test.Case{
		Description: description,
		Setup: func(data test.Data, helpers test.Helpers) {
			// Generate audio file.
			audioFile := gen(data, helpers)
			data.Labels().Set("audio", audioFile)

			// Decode to raw PCM at native rate (no resampling).
			pcmFile := filepath.Join(data.Temp().Dir(), "decoded.pcm")
			testutil.DecodePCM(helpers, audioFile, pcmFile, rate, channels, format)
			data.Labels().Set("pcm", pcmFile)

			// Reference: fpcalc directly on the original file.
			data.Labels().Set("fp-ref", testutil.FpcalcFingerprint(helpers.T(), audioFile))
		},
		Command: func(data test.Data, helpers test.Helpers) test.TestableCommand {
			pcmFile := data.Labels().Get("pcm")

			// sporeprint with internal resampling.
			fpSporeprint := testutil.SporeprintFingerprintWithResample(
				helpers.T(), pcmFile, rate, channels, format,
			)

			fpRef := data.Labels().Get("fp-ref")

			if fpRef != fpSporeprint {
				helpers.T().Log("fpcalc vs sporeprint (internal resample): MISMATCH")
				helpers.T().Log("  fpcalc:     " + fpRef)
				helpers.T().Log("  sporeprint: " + fpSporeprint)
				helpers.T().Fail()
			}

			return helpers.Custom("true")
		},
		Expected: test.Expects(0, nil, nil),
	}
}

func fingerprintSubtest(description string, gen audioGenerator) *test.Case {
	return &test.Case{
		Description: description,
		Setup: func(data test.Data, helpers test.Helpers) {
			// Generate audio file.
			audioFile := gen(data, helpers)
			data.Labels().Set("audio", audioFile)

			// Preprocess to chromaprint-compatible PCM.
			pcmFile := filepath.Join(data.Temp().Dir(), "preprocessed.pcm")
			testutil.PreprocessPCM(helpers, audioFile, pcmFile)
			data.Labels().Set("pcm", pcmFile)

			// 1. fpcalc direct on the original audio file.
			data.Labels().Set("fp-direct", testutil.FpcalcFingerprint(helpers.T(), audioFile))

			// 2. fpcalc on the preprocessed PCM.
			data.Labels().Set("fp-pcm", testutil.FpcalcFingerprintPCM(helpers.T(), pcmFile))
		},
		Command: func(data test.Data, helpers test.Helpers) test.TestableCommand {
			pcmFile := data.Labels().Get("pcm")

			// 3. sporeprint on the preprocessed PCM via stdin.
			fpSporeprint := testutil.SporeprintFingerprint(helpers.T(), pcmFile)

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
