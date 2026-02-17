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
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/containerd/nerdctl/mod/tigron/test"
	"github.com/containerd/nerdctl/mod/tigron/tig"

	"github.com/mycophonic/agar/pkg/agar"
)

// binaryPath and fpcalcPath are resolved once and shared across all test
// functions to avoid a data race when tigron calls AmbientRequirements and
// CustomCommand concurrently.
//
//nolint:gochecknoglobals
var (
	binaryOnce sync.Once
	binaryPath string
	fpcalcOnce sync.Once
	fpcalcPath string
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

func (*sporeprintSetup) AmbientRequirements(_ *test.Case, tester tig.T) {
	if _, err := agar.LookFor(ffmpegBinary); err != nil {
		tester.Skip(ffmpegBinary + " not found")
	}

	fpcalcOnce.Do(func() {
		fpcalcPath = findProjectFpcalc()
		if fpcalcPath == "" {
			p, err := agar.LookFor(fpcalcBinary)
			if err != nil {
				tester.Skip(fpcalcBinary + " not found: run 'make fpcalc'")
			}

			fpcalcPath = p
		}
	})

	binaryOnce.Do(func() {
		path, err := agar.LookFor("sporeprint")
		if err != nil {
			tester.Log("sporeprint not found: run 'make build'")
			tester.FailNow()
		}

		binaryPath = path
	})
}

// findProjectFpcalc looks for the project-built fpcalc binary at bin/tests/fpcalc
// relative to the project root (identified by go.mod).
func findProjectFpcalc() string {
	dir, err := os.Getwd()
	if err != nil {
		return ""
	}

	name := fpcalcBinary
	if runtime.GOOS == "windows" {
		name += ".exe"
	}

	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			candidate := filepath.Join(dir, "bin", "tests", name)
			if _, err := os.Stat(candidate); err == nil {
				return candidate
			}

			return ""
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}

		dir = parent
	}
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
func FpcalcFingerprint(tester tig.T, filePath string) string {
	tester.Helper()

	out, err := exec.CommandContext(context.Background(), fpcalcPath, "-plain", filePath).Output()
	if err != nil {
		tester.Log("fpcalc -plain " + filePath + ": " + err.Error())
		tester.FailNow()
	}

	return strings.TrimSpace(string(out))
}

// FpcalcFingerprintPCM runs fpcalc on a raw s16le 11025Hz mono PCM file and returns the fingerprint.
func FpcalcFingerprintPCM(tester tig.T, pcmPath string) string {
	tester.Helper()

	out, err := exec.CommandContext(context.Background(), fpcalcPath,
		"-format", PCMFormat,
		"-rate", PCMSampleRate,
		"-channels", PCMChannels,
		"-plain", pcmPath,
	).Output()
	if err != nil {
		tester.Log("fpcalc PCM " + pcmPath + ": " + err.Error())
		tester.FailNow()
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

func sporeprintFP(tester tig.T, pcmPath string, extraFlags []string) string {
	tester.Helper()

	bin, err := agar.LookFor("sporeprint")
	if err != nil {
		tester.Log("sporeprint: " + err.Error())
		tester.FailNow()
	}

	pcmFile, err := os.Open(pcmPath)
	if err != nil {
		tester.Log("open PCM: " + err.Error())
		tester.FailNow()
	}

	defer pcmFile.Close()

	args := []string{"fingerprint", "-l", "0"}
	args = append(args, extraFlags...)

	cmd := exec.CommandContext(context.Background(), bin, args...)
	cmd.Stdin = pcmFile

	out, err := cmd.Output()
	if err != nil {
		tester.Log("sporeprint: " + err.Error())
		tester.FailNow()
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
