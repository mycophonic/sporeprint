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

package resample_test

import (
	"encoding/binary"
	"math"
	"testing"

	"github.com/mycophonic/sporeprint/resample"
)

func TestParseFormat(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input   string
		want    resample.Format
		wantErr bool
	}{
		{"s16le", resample.FormatS16LE, false},
		{"s32le", resample.FormatS32LE, false},
		{"f32le", resample.FormatF32LE, false},
		{"s16be", 0, true},
		{"", 0, true},
		{"pcm", 0, true},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			t.Parallel()

			got, err := resample.ParseFormat(tc.input)
			if tc.wantErr {
				if err == nil {
					t.Errorf("ParseFormat(%q) expected error, got %v", tc.input, got)
				}

				return
			}

			if err != nil {
				t.Errorf("ParseFormat(%q) unexpected error: %v", tc.input, err)
			}

			if got != tc.want {
				t.Errorf("ParseFormat(%q) = %v, want %v", tc.input, got, tc.want)
			}
		})
	}
}

func TestBytesPerSample(t *testing.T) {
	t.Parallel()

	if resample.FormatS16LE.BytesPerSample() != 2 {
		t.Error("s16le should be 2 bytes")
	}

	if resample.FormatS32LE.BytesPerSample() != 4 {
		t.Error("s32le should be 4 bytes")
	}

	if resample.FormatF32LE.BytesPerSample() != 4 {
		t.Error("f32le should be 4 bytes")
	}
}

func TestDecodeS16LEMono(t *testing.T) {
	t.Parallel()

	// Encode known int16 values.
	samples := []int16{0, 1000, -1000, 32767, -32768}
	data := make([]byte, len(samples)*2)

	for idx, sample := range samples {
		binary.LittleEndian.PutUint16(data[idx*2:], uint16(sample))
	}

	got := resample.DecodeToFloat64(data, resample.FormatS16LE, 1)
	if len(got) != len(samples) {
		t.Fatalf("got %d samples, want %d", len(got), len(samples))
	}

	for idx, want := range samples {
		if got[idx] != float64(want) {
			t.Errorf("sample %d: got %f, want %f", idx, got[idx], float64(want))
		}
	}
}

func TestDecodeS16LEStereo(t *testing.T) {
	t.Parallel()

	// Stereo: L=1000, R=3000 → mono = 2000.
	data := make([]byte, 4)
	binary.LittleEndian.PutUint16(data[0:], uint16(int16(1000)))
	binary.LittleEndian.PutUint16(data[2:], uint16(int16(3000)))

	got := resample.DecodeToFloat64(data, resample.FormatS16LE, 2)
	if len(got) != 1 {
		t.Fatalf("got %d frames, want 1", len(got))
	}

	if got[0] != 2000.0 {
		t.Errorf("got %f, want 2000.0", got[0])
	}
}

func TestDecodeS32LEMono(t *testing.T) {
	t.Parallel()

	// s32le value 65536 should map to 1.0 in int16 scale (65536 / 65536 = 1).
	// s32le value 2147483647 (max int32) → 2147483647/65536 ≈ 32767.99998.
	values := []int32{0, 65536, -65536, 2147483647, -2147483648}
	expected := []float64{0, 1.0, -1.0, 32767.99998474121, -32768.0}

	data := make([]byte, len(values)*4)

	for idx, val := range values {
		binary.LittleEndian.PutUint32(data[idx*4:], uint32(val))
	}

	got := resample.DecodeToFloat64(data, resample.FormatS32LE, 1)
	if len(got) != len(values) {
		t.Fatalf("got %d samples, want %d", len(got), len(values))
	}

	for idx := range got {
		diff := math.Abs(got[idx] - expected[idx])
		if diff > 0.001 {
			t.Errorf("sample %d: got %f, want %f (diff=%f)", idx, got[idx], expected[idx], diff)
		}
	}
}

func TestDecodeF32LEMono(t *testing.T) {
	t.Parallel()

	// f32le 1.0 → 32767.0, f32le -1.0 → -32767.0, f32le 0.0 → 0.0.
	values := []float32{0.0, 1.0, -1.0, 0.5}
	expected := []float64{0.0, 32767.0, -32767.0, 16383.5}

	data := make([]byte, len(values)*4)

	for idx, val := range values {
		binary.LittleEndian.PutUint32(data[idx*4:], math.Float32bits(val))
	}

	got := resample.DecodeToFloat64(data, resample.FormatF32LE, 1)
	if len(got) != len(values) {
		t.Fatalf("got %d samples, want %d", len(got), len(values))
	}

	for idx := range got {
		diff := math.Abs(got[idx] - expected[idx])
		if diff > 0.01 {
			t.Errorf("sample %d: got %f, want %f", idx, got[idx], expected[idx])
		}
	}
}

func TestDecodeEmptyAndShort(t *testing.T) {
	t.Parallel()

	// Empty input.
	got := resample.DecodeToFloat64(nil, resample.FormatS16LE, 1)
	if got != nil {
		t.Errorf("nil input: got %v, want nil", got)
	}

	// Too short for one frame.
	got = resample.DecodeToFloat64([]byte{0x00}, resample.FormatS16LE, 1)
	if got != nil {
		t.Errorf("short input: got %v, want nil", got)
	}

	// Too short for one stereo frame (need 4 bytes, have 2).
	got = resample.DecodeToFloat64([]byte{0x00, 0x00}, resample.FormatS16LE, 2)
	if got != nil {
		t.Errorf("short stereo input: got %v, want nil", got)
	}
}
