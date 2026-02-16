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

// Package testutil provides test helpers for sporeprint integration tests.
package testutil

import (
	"os"
	"os/exec"
	"strings"
	"sync"

	"github.com/containerd/nerdctl/mod/tigron/test"
	"github.com/containerd/nerdctl/mod/tigron/tig"

	"github.com/mycophonic/agar/pkg/agar"
)

// binaryPath is resolved once and shared across all test functions to avoid
// a data race when tigron calls AmbientRequirements and CustomCommand concurrently.
//
//nolint:gochecknoglobals
var (
	binaryOnce sync.Once
	binaryPath string
)

type sporeprintSetup struct{}

func (*sporeprintSetup) CustomCommand(_ *test.Case, _ tig.T) test.CustomizableCommand {
	cmd := test.NewGenericCommand()
	cmd.WithBinary(binaryPath)

	gen := *(cmd.(*test.GenericCommand))
	gen.WithWhitelist([]string{
		"PATH",
		"HOME",
		"XDG_*",
		// Windows
		"SYSTEMROOT",
		"SYSTEMDRIVE",
		"COMSPEC",
		"TEMP",
		"TMP",
		"USERPROFILE",
		"PATHEXT",
	})

	return &gen
}

func (*sporeprintSetup) AmbientRequirements(_ *test.Case, t tig.T) {
	for _, bin := range []string{"ffmpeg", "fpcalc"} {
		if _, err := agar.LookFor(bin); err != nil {
			t.Skip(bin + " not found")
		}
	}

	binaryOnce.Do(func() {
		path, err := agar.LookFor("sporeprint")
		if err != nil {
			t.Log("sporeprint not found: run 'make build'")
			t.FailNow()
		}

		binaryPath = path
	})
}

// Setup creates a test case configured to run the sporeprint binary.
func Setup() *test.Case {
	test.Customize(&sporeprintSetup{})

	return &test.Case{
		Env: map[string]string{},
	}
}

const (
	fpcalcBinary = "fpcalc"
	ffmpegBinary = "ffmpeg"

	// AresampleFilter is the ffmpeg aresample filter that matches fpcalc's SetCompatibleMode()
	// for identical fingerprints. See chromaprint documentation for details.
	AresampleFilter = "aresample=resampler=swr:filter_size=16:phase_shift=8:cutoff=0.8:linear_interp=1"

	// PCMFormat is the chromaprint-compatible sample format (signed 16-bit little-endian).
	PCMFormat = "s16le"
	// PCMSampleRate is the chromaprint-compatible sample rate in Hz.
	PCMSampleRate = "11025"
	// PCMChannels is the chromaprint-compatible channel count.
	PCMChannels = "1"
)

// FpcalcFingerprint runs fpcalc directly on an audio file and returns the fingerprint string.
func FpcalcFingerprint(t tig.T, filePath string) string {
	t.Helper()

	fpcalc, err := agar.LookFor(fpcalcBinary)
	if err != nil {
		t.Log(fpcalcBinary + ": " + err.Error())
		t.FailNow()
	}

	out, err := exec.Command(fpcalc, "-plain", filePath).Output()
	if err != nil {
		t.Log("fpcalc -plain " + filePath + ": " + err.Error())
		t.FailNow()
	}

	return strings.TrimSpace(string(out))
}

// FpcalcFingerprintPCM runs fpcalc on a raw s16le 11025Hz mono PCM file and returns the fingerprint.
func FpcalcFingerprintPCM(t tig.T, pcmPath string) string {
	t.Helper()

	fpcalc, err := agar.LookFor(fpcalcBinary)
	if err != nil {
		t.Log(fpcalcBinary + ": " + err.Error())
		t.FailNow()
	}

	out, err := exec.Command(fpcalc,
		"-format", PCMFormat,
		"-rate", PCMSampleRate,
		"-channels", PCMChannels,
		"-plain", pcmPath,
	).Output()
	if err != nil {
		t.Log("fpcalc PCM " + pcmPath + ": " + err.Error())
		t.FailNow()
	}

	return strings.TrimSpace(string(out))
}

// PreprocessPCM converts an audio file to chromaprint-compatible s16le 11025Hz mono PCM using ffmpeg.
func PreprocessPCM(helpers test.Helpers, inputPath, outputPath string) {
	helpers.T().Helper()

	ffmpeg, err := agar.LookFor(ffmpegBinary)
	if err != nil {
		helpers.T().Log(ffmpegBinary + ": " + err.Error())
		helpers.T().FailNow()
	}

	helpers.Custom(ffmpeg,
		"-i", inputPath,
		"-af", AresampleFilter,
		"-f", PCMFormat,
		"-ac", PCMChannels,
		"-ar", PCMSampleRate,
		"-y", outputPath,
	).Run(&test.Expected{})
}

// SporeprintFingerprint feeds a PCM file to sporeprint via stdin and returns the fingerprint.
func SporeprintFingerprint(t tig.T, pcmPath string) string {
	t.Helper()

	return sporeprintFP(t, pcmPath, nil)
}

// SporeprintFingerprintWithResample feeds a raw PCM file to sporeprint with
// --rate, --channels, and --format flags, letting sporeprint handle resampling internally.
func SporeprintFingerprintWithResample(t tig.T, pcmPath, rate, channels, format string) string {
	t.Helper()

	return sporeprintFP(t, pcmPath, []string{"-r", rate, "-c", channels, "-f", format})
}

func sporeprintFP(t tig.T, pcmPath string, extraFlags []string) string {
	t.Helper()

	bin, err := agar.LookFor("sporeprint")
	if err != nil {
		t.Log("sporeprint: " + err.Error())
		t.FailNow()
	}

	f, err := os.Open(pcmPath)
	if err != nil {
		t.Log("open PCM: " + err.Error())
		t.FailNow()
	}

	defer f.Close()

	args := []string{"fingerprint", "-l", "0"}
	args = append(args, extraFlags...)

	cmd := exec.Command(bin, args...)
	cmd.Stdin = f

	out, err := cmd.Output()
	if err != nil {
		t.Log("sporeprint: " + err.Error())
		t.FailNow()
	}

	return strings.TrimSpace(string(out))
}

// DecodePCM converts an audio file to raw PCM at the given rate/channels/format using ffmpeg.
// No resampling is applied — ffmpeg only decodes and converts sample format.
func DecodePCM(helpers test.Helpers, inputPath, outputPath, rate, channels, format string) {
	helpers.T().Helper()

	ffmpeg, err := agar.LookFor(ffmpegBinary)
	if err != nil {
		helpers.T().Log(ffmpegBinary + ": " + err.Error())
		helpers.T().FailNow()
	}

	helpers.Custom(ffmpeg,
		"-i", inputPath,
		"-f", format,
		"-ac", channels,
		"-ar", rate,
		"-y", outputPath,
	).Run(&test.Expected{})
}
