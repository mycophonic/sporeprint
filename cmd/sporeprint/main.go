// Package main provides a sample cli implementation for a fingerprinting tool using chromaprint.
// See main README.md for details.
package main

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/urfave/cli/v3"

	"github.com/farcloser/sporeprint/pkg/chromaprint"
)

// See README.
const (
	sampleRate      = 11025
	channels        = 1
	defaultDuration = 120
	bufferSize      = 8192
	samplesSize     = 4096
)

var (
	ErrEncodeFailure      = errors.New("encode failure")
	ErrChromaprintFailure = errors.New("chromaprint error")
	ErrReadFailure        = errors.New("read error")
)

func main() {
	cmd := &cli.Command{
		Name:  "sporeprint",
		Usage: "Generate audio fingerprints from raw PCM via stdin",
		Description: `Reads signed 16-bit PCM audio from stdin and outputs a Chromaprint fingerprint.

Chromaprint expects 11025 Hz mono input s16le. Example:

  ffmpeg -i track.flac -f s16le -ar 11025 -ac 1 pipe:1 2>/dev/null | sporeprint`,
		Flags: []cli.Flag{
			&cli.IntFlag{
				Name:    "length",
				Aliases: []string{"l"},
				Value:   defaultDuration,
				Usage:   "max audio length in seconds (0 = unlimited)",
			},
			&cli.BoolFlag{
				Name:    "version",
				Aliases: []string{"v"},
				Usage:   "print version and exit",
			},
		},
		Action: run,
	}

	if err := cmd.Run(context.Background(), os.Args); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "error: %v\n", err)

		os.Exit(1)
	}
}

func run(_ context.Context, cliCom *cli.Command) error {
	if cliCom.Bool("version") {
		_, _ = fmt.Fprintf(os.Stderr, "sporeprint using chromaprint %s\n", chromaprint.Version())

		return nil
	}

	length := cliCom.Int("length")
	byteOrder := binary.LittleEndian

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
	samples := make([]int16, samplesSize)
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
		for i := range numSamples {
			samples[i] = int16(byteOrder.Uint16(buf[i*2:]))
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
