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
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/urfave/cli/v3"

	"github.com/mycophonic/primordium/app"

	"github.com/mycophonic/sporeprint/chromaprint"
	"github.com/mycophonic/sporeprint/compare"
	"github.com/mycophonic/sporeprint/version"
)

// See README.
const (
	sampleRate      = 11025
	channels        = 1
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
				Description: `Reads signed 16-bit PCM audio from stdin and outputs a Chromaprint fingerprint.

Chromaprint expects 11025 Hz mono input s16le. Example:

  ffmpeg -i track.flac -af "aresample=resampler=swr:filter_size=16:phase_shift=8:cutoff=0.8:linear_interp=1" -f s16le -ac 1 -ar 11025 pipe:1 2>/dev/null | sporeprint fingerprint

The aresample filter parameters ensure identical output to fpcalc.`,
				Flags: []cli.Flag{
					&cli.IntFlag{
						Name:    "length",
						Aliases: []string{"l"},
						Value:   defaultDuration,
						Usage:   "max audio length in seconds (0 = unlimited)",
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
	length := cliCom.Int("length")

	chroma := chromaprint.New()
	defer chroma.Free()

	if err := chroma.Start(sampleRate, channels); err != nil {
		return fmt.Errorf("%w: %w", ErrChromaprintFailure, err)
	}

	// Calculate sample limit: rate × channels × seconds
	var maxSamples int
	if length > 0 {
		maxSamples = sampleRate * channels * length
	}

	buf := make([]byte, bufferSize)
	samples := make([]int16, bufferSize/2)
	totalFed := 0

	for {
		nread, err := io.ReadFull(os.Stdin, buf)
		if nread == 0 {
			if errors.Is(err, io.EOF) {
				break
			}

			if err != nil {
				return fmt.Errorf("%w: %w", ErrReadFailure, err)
			}
		}

		// Convert bytes to int16 samples
		numSamples := nread / 2
		for i := 0; i+1 < nread; i += 2 {
			//nolint:gosec // samples size = 1/2 buffer size
			samples[i/2] = int16(binary.LittleEndian.Uint16(buf[i : i+2]))
		}

		// Apply length limit
		toFeed := numSamples
		if maxSamples > 0 && totalFed+toFeed > maxSamples {
			toFeed = maxSamples - totalFed
			if toFeed <= 0 {
				break
			}
		}

		if err = chroma.Feed(samples[:toFeed]); err != nil {
			return fmt.Errorf("%w: %w", ErrChromaprintFailure, err)
		}

		totalFed += toFeed

		if maxSamples > 0 && totalFed >= maxSamples {
			break
		}
	}

	if err := chroma.Finish(); err != nil {
		return fmt.Errorf("%w: %w", ErrChromaprintFailure, err)
	}

	fingerprint, err := chroma.Fingerprint()
	if err != nil {
		return fmt.Errorf("%w: %w", ErrChromaprintFailure, err)
	}

	_, _ = fmt.Fprintln(os.Stdout, fingerprint)

	return nil
}
