# Audio Fingerprinting Solutions

## Open Source Libraries

| Name | Language | Stars | Notes |
|------|----------|-------|-------|
| **Chromaprint** | C/C++ | ~2k | The standard. Powers AcoustID/MusicBrainz. Analyzes first 2 min, 12 pitch classes, 8x/sec. Compact fingerprints (~2.5KB). Very fast (<100ms for 2 min). |
| **Olaf** | C | ~380 | "Overly Lightweight Acoustic Fingerprinting." <1MB memory footprint. Runs on ESP32, WASM. Shazam-style algorithm. Part of Panako project. |
| **Panako** | Java | — | Research platform with multiple algorithms. Handles time-stretch and pitch-shift (unique). AGPL licensed. Patent concerns on some algorithms (US7627477, US6990453). |
| **Dejavu** | Python | — | Real-time recognition. Shazam-style spectrogram peaks. Good for learning, not production-grade. |
| **audfprint** | Python/Matlab | — | Dan Ellis (Columbia). Research-oriented. Two implementations available. |
| **NeuralFP** | Python | ~200 | Deep learning approach (ICASSP 2021). Neural network-based fingerprinting. |
| **Echoprint** | C/C++ | — | Was Echo Nest, acquired by Spotify. **Abandoned.** |
| **LibOFA** | C | — | Open Fingerprint Architecture. Old, basically dead. |
| **SoundFingerprinting** | C# (.NET) | — | Aurio-based. Windows-focused. |
| **JAudioScout** | Java | — | Based on pHash / Philips "Robust Audio Hashing" algorithm. |

---

## Commercial APIs

| Name | Notes |
|------|-------|
| **Shazam** | The original. Now Apple. No public API. |
| **ACRCloud** | Commercial API. Huge database. Pay per query. |
| **AudD** | Commercial API. Music recognition service. |
| **Gracenote** | Sony-owned. Powers car systems, TVs. Enterprise pricing. |
| **AcoustID** | Free API with 30M+ fingerprints linked to MusicBrainz. Rate limited. Commercial tier available via acoustid.biz. |

---

## Dead / Deprecated

| Name | Status |
|------|--------|
| **MusicIP / PUID** | Dead. Was MusicBrainz's first fingerprinting system. Proprietary, slow, operators abandoned it. |
| **Echoprint** | Abandoned after Spotify acquisition of Echo Nest. |
| **Freetantrum** | Dead. Used eTantrum's Songprint service (defunct). |

---

## Technical Comparison

| Library | Speed | Fingerprint Size | Pitch/Time Shift | Offline Index | Embedded |
|---------|-------|------------------|------------------|---------------|----------|
| Chromaprint | Very fast | ~2.5KB | ❌ | ❌ (needs AcoustID) | ❌ |
| Olaf | Fast | Small | ❌ | ✅ | ✅ (ESP32, WASM) |
| Panako | Medium | Medium | ✅ | ✅ | ❌ |
| Dejavu | Slow | Large | ❌ | ✅ | ❌ |
| NeuralFP | Slow (GPU) | Small | Partial | ✅ | ❌ |

---

## Algorithm Families

### Shazam-style (Spectrogram Peaks)
- Identify prominent peaks in spectrogram
- Create "constellation" of time-frequency points
- Hash pairs of peaks for matching
- Used by: Shazam, Olaf, Dejavu, Echoprint

### Chromaprint-style (Chroma Features)
- Extract pitch class strength over time
- 12 bins (one per semitone in octave)
- Compact representation, good for near-identical matching
- Used by: Chromaprint/AcoustID

### Philips Robust Hashing
- Described in Haitsma & Kalker paper
- 32-bit sub-fingerprints from energy differences
- Used by: JAudioScout, some Panako algorithms

### Neural / Deep Learning
- CNN or RNN-based feature extraction
- More robust to distortion but slower
- Used by: NeuralFP

---

## Patent Concerns

Several fingerprinting techniques are covered by patents:
- **US7627477 B2** — Shazam's algorithm
- **US6990453** — Related techniques

Panako documentation explicitly warns about this. Chromaprint uses a different approach (chroma-based) that avoids these patents.

---

## Sporeprint

### Current Use Case
Identify files in local library, match to MusicBrainz, help with duplicate detection.

### Primary Choice: Chromaprint + AcoustID
- Battle-tested, huge database (30M+ fingerprints)
- Direct MusicBrainz integration
- Fast, compact fingerprints
- Well-maintained

### Secondary Choice: Olaf
- For embedded/mobile (iOS, resource-constrained devices)
- Can run entirely offline
- Small memory footprint (<1MB)
- Good for local duplicate detection without network

---

## Resources

- [Chromaprint: How it works](https://oxygene.sk/2011/01/how-does-chromaprint-work/)
- [AcoustID](https://acoustid.org/)
- [Panako](https://github.com/JorenSix/Panako)
- [Olaf](https://github.com/JorenSix/Olaf)
- [MusicBrainz Fingerprinting docs](https://musicbrainz.org/doc/Fingerprinting)