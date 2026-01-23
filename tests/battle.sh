#!/usr/bin/env bash

set -uo pipefail

folder="${1:?Usage: $0 <folder>}"

find "$folder" -type f \( -iname "*.flac" \) -print0 |
  while IFS= read -r -d '' file; do
    echo "$file"
    # fpcalc tend to not be happy with stdin (or closes stdin too abruptly for ffmpeg?), but still generates a fingerprint -\o/-
    fpfp="$(ffmpeg -nostdin -i "$file" -f s16le -ar 11025 -ac 1 pipe:1 2>/dev/null | fpcalc -format s16le -rate 11025 -channels 1  - | grep FINGERPRINT | sed 's/FINGERPRINT=//')" || true
    spfp="$(ffmpeg -nostdin -i "$file" -f s16le -ar 11025 -ac 1 pipe:1 2>/dev/null | ./bin/sporeprint)"
    if [ "$fpfp" != "$spfp" ]; then
      >&2 echo "-----------------------------------------------------"
      >&2 echo "FAIL: fingerprints differ"
      >&2 echo "$file"
      >&2 echo "-----------------------------------------------------"
#      echo "$fpfp"
#      echo "$spfp"
    fi
  done