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

package chromaprint

/*
#cgo CFLAGS: -I${SRCDIR}/../bin
#cgo windows CFLAGS: -DCHROMAPRINT_NODLL
#cgo LDFLAGS: ${SRCDIR}/../bin/libchromaprint.a -lstdc++ -lm
#include "chromaprint.h"
#include <stdlib.h>
*/
import "C"

import (
	"errors"
	"unsafe"
)

var (
	// ErrFingerprint happens on a fingerprinting error.
	ErrFingerprint = errors.New("chromaprint: fingerprinting failed")
	// ErrFreed happens when the context has been already freed.
	ErrFreed = errors.New("chromaprint: context already freed")
	// ErrDecode happens when decoding an encoded fingerprint fails.
	ErrDecode = errors.New("chromaprint: decode failed")
)

// Context wraps a ChromaprintContext.
type Context struct {
	ctx *C.ChromaprintContext
}

// New creates a new Chromaprint context using the default algorithm.
func New() *Context {
	return &Context{
		ctx: C.chromaprint_new(C.CHROMAPRINT_ALGORITHM_DEFAULT),
	}
}

// Free releases the context resources. Call when done.
func (c *Context) Free() {
	if c.ctx != nil {
		C.chromaprint_free(c.ctx)
		c.ctx = nil
	}
}

// Start initializes fingerprinting for the given audio format.
// For best results, use SampleRate() and NumChannels() to get the expected values.
func (c *Context) Start(sampleRate, channels int) error {
	if c.ctx == nil {
		return ErrFreed
	}

	if C.chromaprint_start(c.ctx, C.int(sampleRate), C.int(channels)) != 1 {
		return ErrFingerprint
	}

	return nil
}

// Feed sends PCM audio data to the fingerprinter.
// Data must be signed 16-bit integers, interleaved for stereo.
// Size is the total number of samples (frames × channels).
func (c *Context) Feed(data []int16) error {
	if c.ctx == nil {
		return ErrFreed
	}

	if len(data) == 0 {
		return nil
	}

	if C.chromaprint_feed(c.ctx, (*C.int16_t)(unsafe.Pointer(&data[0])), C.int(len(data))) != 1 {
		return ErrFingerprint
	}

	return nil
}

// Finish signals end of audio data and finalizes the fingerprint.
func (c *Context) Finish() error {
	if c.ctx == nil {
		return ErrFreed
	}

	if C.chromaprint_finish(c.ctx) != 1 {
		return ErrFingerprint
	}

	return nil
}

// Fingerprint returns the calculated fingerprint as a compressed
// base64-encoded string. Use [Decode] to obtain the raw uint32 array
// for comparison operations.
func (c *Context) Fingerprint() (string, error) {
	if c.ctx == nil {
		return "", ErrFreed
	}

	var encoded *C.char
	if C.chromaprint_get_fingerprint(c.ctx, &encoded) != 1 {
		return "", ErrFingerprint
	}

	fp := C.GoString(encoded)

	C.chromaprint_dealloc(unsafe.Pointer(encoded))

	return fp, nil
}

// Decode converts a base64-encoded Chromaprint fingerprint (as returned by
// [Context.Fingerprint]) into a raw uint32 subfingerprint array suitable for
// comparison operations.
func Decode(encoded string) ([]uint32, error) {
	cEncoded := C.CString(encoded)
	defer C.free(unsafe.Pointer(cEncoded))

	var rawPtr *C.uint32_t
	var rawSize C.int
	var algorithm C.int

	if C.chromaprint_decode_fingerprint(cEncoded, C.int(len(encoded)), &rawPtr, &rawSize, &algorithm, 1) != 1 {
		return nil, ErrDecode
	}

	defer C.chromaprint_dealloc(unsafe.Pointer(rawPtr))

	raw := make([]uint32, int(rawSize))
	copy(raw, unsafe.Slice((*uint32)(unsafe.Pointer(rawPtr)), int(rawSize)))

	return raw, nil
}

// Version returns the Chromaprint library version string.
func Version() string {
	return C.GoString(C.chromaprint_get_version())
}
