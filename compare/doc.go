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

// Package compare provides Chromaprint fingerprint comparison.
//
// All public functions accept base64-encoded fingerprints as returned by
// [github.com/mycophonic/sporeprint/chromaprint.Context.Fingerprint].
// Decoding to raw uint32 arrays is handled internally via
// [github.com/mycophonic/sporeprint/chromaprint.Decode].
//
// Based on the AcoustID PostgreSQL matching function.
// Reference: https://oxygene.sk/2011/01/how-does-chromaprint-work/
package compare
