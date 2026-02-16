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

package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/urfave/cli/v3"

	"github.com/mycophonic/primordium/app"

	"github.com/mycophonic/sporeprint/chromaprint"
	"github.com/mycophonic/sporeprint/compare"
	"github.com/mycophonic/sporeprint/resample"
	"github.com/mycophonic/sporeprint/version"
)

// See README.
const (
	targetRate      = 11025
	defaultDuration = 120
	bufferSize      = 8192

	// defaultThreshold is the minimum similarity score to consider two
	// fingerprints a match. Matches AcoustID's TRACK_GROUP_MERGE_THRESHOLD.
	// Reference: https://github.com/acoustid/acoustid-server
	defaultThreshold = 0.4
)

var (
	ErrChromaprintFailure = errors.New("chromaprint error")
	ErrCompareFailure     = errors.New("compare error")
	ErrInvalidArgs        = errors.New("invalid arguments")
	ErrReadFailure        = errors.New("read error")
	ErrNoMatch            = errors.New("no match")
)

func main() {
	ctx := context.Background()
	app.New(ctx, version.Name())

	appl := &cli.Command{
		Name:    version.Name(),
		Usage:   "Audio fingerprinting toolkit",
		Version: version.Version() + " (" + version.Commit() + " - " + version.Date() + " - chromaprint " + chromaprint.Version() + ")",
		Commands: []*cli.Command{
			{
				Name:  "fingerprint",
				Usage: "Generate a Chromaprint fingerprint from raw PCM via stdin",
				Description: `Reads raw PCM audio from stdin, resamples to 11025 Hz mono, and outputs a Chromaprint fingerprint.

Examples:
  # Let sporeprint handle resampling (recommended):
  ffmpeg -i track.flac -f s16le -ac 2 -ar 44100 pipe:1 | sporeprint fingerprint -r 44100 -c 2

  # Pre-resampled input (default flags):
  ffmpeg -i track.flac -f s16le -ac 1 -ar 11025 pipe:1 | sporeprint fingerprint`,
				Flags: []cli.Flag{
					&cli.IntFlag{
						Name:    "length",
						Aliases: []string{"l"},
						Value:   defaultDuration,
						Usage:   "max audio length in seconds (0 = unlimited)",
					},
					&cli.IntFlag{
						Name:    "rate",
						Aliases: []string{"r"},
						Value:   targetRate,
						Usage:   "source sample rate in Hz",
					},
					&cli.IntFlag{
						Name:    "channels",
						Aliases: []string{"c"},
						Value:   1,
						Usage:   "source channel count",
					},
					&cli.StringFlag{
						Name:    "format",
						Aliases: []string{"f"},
						Value:   "s16le",
						Usage:   "source sample format: s16le, s32le, f32le",
					},
				},
				Action: runFingerprint,
			},
			{
				Name:      "compare",
				Usage:     "Compare two encoded Chromaprint fingerprints",
				ArgsUsage: "FINGERPRINT1 FINGERPRINT2",
				Flags: []cli.Flag{
					&cli.FloatFlag{
						Name:    "threshold",
						Aliases: []string{"t"},
						Value:   defaultThreshold,
						Usage:   "minimum similarity score to consider a match (0.0-1.0)",
					},
				},
				Action: runCompare,
			},
		},
	}

	if err := appl.Run(ctx, os.Args); err != nil {
		if !errors.Is(err, ErrNoMatch) {
			_, _ = fmt.Fprintf(os.Stderr, "error: %v\n", err)
		}

		os.Exit(1)
	}
}

func runCompare(_ context.Context, cliCom *cli.Command) error {
	args := cliCom.Args()
	if args.Len() != 2 { //nolint:mnd
		return fmt.Errorf("%w: expected exactly 2 fingerprints, got %d", ErrInvalidArgs, args.Len())
	}

	fp1 := args.Get(0)
	fp2 := args.Get(1)
	threshold := cliCom.Float("threshold")

	score, err := compare.Compare(fp1, fp2)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrCompareFailure, err)
	}

	if score >= threshold {
		_, _ = fmt.Fprintf(os.Stdout, "score=%.3f match (threshold=%.2f)\n", score, threshold)

		return nil
	}

	_, _ = fmt.Fprintf(os.Stdout, "score=%.3f no match (threshold=%.2f)\n", score, threshold)

	return ErrNoMatch
}

func runFingerprint(_ context.Context, cliCom *cli.Command) error {
	srcRate := cliCom.Int("rate")
	srcChannels := cliCom.Int("channels")
	length := cliCom.Int("length")

	format, err := resample.ParseFormat(cliCom.String("format"))
	if err != nil {
		return fmt.Errorf("%w: %w", ErrInvalidArgs, err)
	}

	if srcRate <= 0 {
		return fmt.Errorf("%w: rate must be positive", ErrInvalidArgs)
	}

	if srcChannels <= 0 {
		return fmt.Errorf("%w: channels must be positive", ErrInvalidArgs)
	}

	chroma := chromaprint.New()
	defer chroma.Free()

	if err = chroma.Start(targetRate, 1); err != nil {
		return fmt.Errorf("%w: %w", ErrChromaprintFailure, err)
	}

	// Create resampler (nil when srcRate == 11025).
	resampler, err := resample.NewResampler(srcRate)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrInvalidArgs, err)
	}

	if err = feedPCM(chroma, resampler, format, srcRate, srcChannels, length); err != nil {
		return err
	}

	if err = chroma.Finish(); err != nil {
		return fmt.Errorf("%w: %w", ErrChromaprintFailure, err)
	}

	fingerprint, err := chroma.Fingerprint()
	if err != nil {
		return fmt.Errorf("%w: %w", ErrChromaprintFailure, err)
	}

	_, _ = fmt.Fprintln(os.Stdout, fingerprint)

	return nil
}

// feedPCM reads raw PCM from stdin, decodes, resamples, and feeds chromaprint.
func feedPCM(
	chroma *chromaprint.Context,
	resampler *resample.Resampler,
	format resample.Format,
	srcRate, srcChannels, length int,
) error {
	// Length limit in source frames (before resampling).
	var maxFrames int
	if length > 0 {
		maxFrames = srcRate * length
	}

	buf := make([]byte, bufferSize)
	totalFrames := 0

	for {
		mono, done, err := readChunk(buf, format, srcChannels)
		if err != nil {
			return err
		}

		if len(mono) > 0 {
			// Apply length limit in source frames.
			mono = limitFrames(mono, maxFrames, totalFrames)
			totalFrames += len(mono)

			if feedErr := feedChunk(chroma, resampler, mono); feedErr != nil {
				return feedErr
			}
		}

		if done || (maxFrames > 0 && totalFrames >= maxFrames) {
			break
		}
	}

	return flushResampler(chroma, resampler)
}

// readChunk reads one buffer of raw PCM from stdin and decodes to mono float64.
// Returns the decoded samples, whether EOF was reached, and any read error.
func readChunk(buf []byte, format resample.Format, channels int) ([]float64, bool, error) {
	nread, readErr := io.ReadFull(os.Stdin, buf)

	if nread == 0 {
		if errors.Is(readErr, io.EOF) {
			return nil, true, nil
		}

		if readErr != nil {
			return nil, false, fmt.Errorf("%w: %w", ErrReadFailure, readErr)
		}
	}

	mono := resample.DecodeToFloat64(buf[:nread], format, channels)
	done := errors.Is(readErr, io.EOF) || errors.Is(readErr, io.ErrUnexpectedEOF)

	return mono, done, nil
}

// limitFrames truncates mono to respect the maximum source frame count.
func limitFrames(mono []float64, maxFrames, totalFrames int) []float64 {
	if maxFrames <= 0 {
		return mono
	}

	remaining := maxFrames - totalFrames
	if remaining <= 0 {
		return nil
	}

	if len(mono) > remaining {
		return mono[:remaining]
	}

	return mono
}

// flushResampler drains remaining samples from the resampler into chromaprint.
func flushResampler(chroma *chromaprint.Context, resampler *resample.Resampler) error {
	if resampler == nil {
		return nil
	}

	flushed := resampler.Flush()
	if len(flushed) == 0 {
		return nil
	}

	if err := chroma.Feed(flushed); err != nil {
		return fmt.Errorf("%w: %w", ErrChromaprintFailure, err)
	}

	return nil
}

// feedChunk resamples a chunk of mono float64 samples and feeds it to chromaprint.
func feedChunk(chroma *chromaprint.Context, resampler *resample.Resampler, mono []float64) error {
	var out []int16
	if resampler != nil {
		out = resampler.Write(mono)
	} else {
		out = resample.FloatToInt16(mono)
	}

	if len(out) > 0 {
		if err := chroma.Feed(out); err != nil {
			return fmt.Errorf("%w: %w", ErrChromaprintFailure, err)
		}
	}

	return nil
}
