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

package compare

import (
	"fmt"
	"math/bits"

	"github.com/mycophonic/sporeprint/chromaprint"
)

const (
	// MaxAlignOffset is the maximum offset to search when aligning fingerprints.
	// At ~0.1238s per hash, 120 hashes ≈ 15 seconds of drift tolerance.
	MaxAlignOffset = 120

	// MaxBitError is the maximum bit errors to consider two hashes as matching.
	// With 32-bit hashes, 2 bits = 93.75% similarity threshold per hash.
	MaxBitError = 2

	// scoreNoMatch is returned when fingerprints cannot be compared.
	scoreNoMatch = 0.0

	// scoreMaxDissimilarity is returned for maximum dissimilarity.
	scoreMaxDissimilarity = 1.0

	// bitsPerHash is the number of bits in each uint32 subfingerprint.
	bitsPerHash = 32
)

// Compare compares two encoded Chromaprint fingerprints and returns a
// similarity score between 0.0 (completely different) and 1.0 (identical).
//
// The algorithm:
//  1. For each hash in fp1, search within ±MaxAlignOffset hashes in fp2
//  2. If XOR bit error ≤ MaxBitError, record a match at that alignment offset
//  3. Build histogram of matches per offset
//  4. Score = best offset's match count / min(len(fp1), len(fp2))
func Compare(fp1, fp2 string) (float64, error) {
	raw1, raw2, err := decodePair(fp1, fp2)
	if err != nil {
		return scoreNoMatch, err
	}

	score, _ := compareRaw(raw1, raw2)

	return score, nil
}

// WithOffset is like [Compare] but also returns the best alignment
// offset. A positive offset means fp1 starts later than fp2. A negative
// offset means fp1 starts earlier than fp2.
func WithOffset(fp1, fp2 string) (score float64, offset int, err error) {
	raw1, raw2, err := decodePair(fp1, fp2)
	if err != nil {
		return scoreNoMatch, 0, err
	}

	score, offset = compareRaw(raw1, raw2)

	return score, offset, nil
}

// BitErrorRate computes the average bit error rate between two aligned
// encoded fingerprints. Use this after aligning with [WithOffset].
//
// Returns a value between 0.0 (identical) and 1.0 (completely different).
// For same track, expect < 0.1. For different tracks, expect ~0.5.
//
// If the fingerprints do not overlap at the given offset (offset exceeds
// either fingerprint's length), returns 1.0 (maximum dissimilarity).
func BitErrorRate(fp1, fp2 string, offset int) (float64, error) {
	raw1, raw2, err := decodePair(fp1, fp2)
	if err != nil {
		return scoreMaxDissimilarity, err
	}

	return bitErrorRateRaw(raw1, raw2, offset), nil
}

// IsSameTrack returns true if the encoded fingerprints likely represent the
// same audio track. Threshold is the minimum similarity score (0.0-1.0).
// Suggested: 0.5-0.7.
func IsSameTrack(fp1, fp2 string, threshold float64) (bool, error) {
	score, err := Compare(fp1, fp2)
	if err != nil {
		return false, err
	}

	return score >= threshold, nil
}

// decodePair decodes two encoded fingerprints into raw uint32 arrays.
func decodePair(fp1, fp2 string) (raw1, raw2 []uint32, err error) {
	raw1, err = chromaprint.Decode(fp1)
	if err != nil {
		return nil, nil, fmt.Errorf("decoding fp1: %w", err)
	}

	raw2, err = chromaprint.Decode(fp2)
	if err != nil {
		return nil, nil, fmt.Errorf("decoding fp2: %w", err)
	}

	return raw1, raw2, nil
}

// compareRaw compares two raw fingerprint arrays and returns the similarity
// score and best alignment offset.
func compareRaw(fp1, fp2 []uint32) (score float64, offset int) {
	if len(fp1) == 0 || len(fp2) == 0 {
		return scoreNoMatch, 0
	}

	// Histogram of matches per alignment offset.
	// Offset range: -(len(fp2)-1) to +(len(fp1)-1)
	// We shift by len(fp2) to make all indices positive.
	numCounts := len(fp1) + len(fp2) + 1
	counts := make([]int, numCounts)

	for idx1 := range fp1 {
		jBegin := max(0, idx1-MaxAlignOffset)
		jEnd := min(len(fp2), idx1+MaxAlignOffset)

		for idx2 := jBegin; idx2 < jEnd; idx2++ {
			bitError := bits.OnesCount32(fp1[idx1] ^ fp2[idx2])
			if bitError <= MaxBitError {
				offsetIdx := idx1 - idx2 + len(fp2)
				counts[offsetIdx]++
			}
		}
	}

	maxCount := 0
	bestIdx := len(fp2) // default: zero offset

	for idx, c := range counts {
		if c > maxCount {
			maxCount = c
			bestIdx = idx
		}
	}

	minLen := min(len(fp1), len(fp2))
	score = float64(maxCount) / float64(minLen)
	offset = bestIdx - len(fp2) // convert back to actual offset

	return score, offset
}

// bitErrorRateRaw computes the average bit error rate between two aligned
// raw fingerprint arrays.
func bitErrorRateRaw(fp1, fp2 []uint32, offset int) float64 {
	if len(fp1) == 0 || len(fp2) == 0 {
		return scoreMaxDissimilarity
	}

	// Determine overlapping region.
	var start1, start2, length int
	if offset >= 0 {
		start1 = offset
		start2 = 0
		length = min(len(fp1)-offset, len(fp2))
	} else {
		start1 = 0
		start2 = -offset
		length = min(len(fp1), len(fp2)+offset)
	}

	// No overlap: fingerprints are completely disjoint at this offset.
	if length <= 0 {
		return scoreMaxDissimilarity
	}

	totalBitErrors := 0
	for i := range length {
		totalBitErrors += bits.OnesCount32(fp1[start1+i] ^ fp2[start2+i])
	}

	// Each hash has 32 bits, so max errors per hash is 32.
	return float64(totalBitErrors) / float64(length*bitsPerHash)
}
