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

// Package resample implements a polyphase sinc resampler compatible with
// ffmpeg's libswresample (swr). It converts PCM audio at arbitrary sample
// rates to 11025 Hz mono signed 16-bit, matching the exact filter parameters
// used by fpcalc for Chromaprint fingerprinting:
//
//	filter_size=16, phase_shift=8, cutoff=0.8, linear_interp=1, kaiser_beta=9
//
// The algorithm uses a Kaiser-windowed sinc filter partitioned into 256
// polyphase banks with linear interpolation between adjacent phases.
//
// Reference: ffmpeg libswresample/resample.c (LGPL-2.1).
// This is a clean-room reimplementation of the algorithm in pure Go.
package resample
