#!/usr/bin/env bash

set -uo pipefail

folder="${1:?Usage: $0 <folder>}"

find "$folder" -type f \( -iname "*.flac" -o -iname "*.mp3" -o -iname "*.m4a" -o -iname "*.mp4" \) -print0 |
  while IFS= read -r -d '' file; do
    echo "$file"
    # aresample parameters must match fpcalc's SetCompatibleMode() for identical fingerprints
    fpfpd="$(fpcalc "$file" | grep FINGERPRINT | sed 's/FINGERPRINT=//')"
    fpfp="$(ffmpeg -nostdin -i "$file" -af "aresample=resampler=swr:filter_size=16:phase_shift=8:cutoff=0.8:linear_interp=1" -f s16le -ac 1 -ar 11025 pipe:1 2>/dev/null | fpcalc -format s16le -rate 11025 -channels 1 - | grep FINGERPRINT | sed 's/FINGERPRINT=//')" || true
    spfp="$(ffmpeg -nostdin -i "$file" -af "aresample=resampler=swr:filter_size=16:phase_shift=8:cutoff=0.8:linear_interp=1" -f s16le -ac 1 -ar 11025 pipe:1 2>/dev/null | ./bin/sporeprint)"
    if [ "$fpfpd" != "$spfp" ]; then
      >&2 echo "-----------------------------------------------------"
      >&2 echo "FAIL: fingerprints differ (direct)"
      >&2 echo "$file"
      >&2 echo "-----------------------------------------------------"
    fi
    if [ "$fpfp" != "$spfp" ]; then
      >&2 echo "-----------------------------------------------------"
      >&2 echo "FAIL: fingerprints differ (stdin)"
      >&2 echo "$file"
      >&2 echo "-----------------------------------------------------"
      >&2 echo "$fpfpd"
      >&2 echo "$fpfp"
      >&2 echo "$spfp"
    fi

  done