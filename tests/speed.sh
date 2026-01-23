#!/usr/bin/env bash

set -uo pipefail

folder="${1:?Usage: $0 <folder>}"
bin="${2:?Usage: $0 <fpcalc|sporeprint>}"

find "$folder" -type f \( -iname "*.flac" \) -print0 |
  while IFS= read -r -d '' file; do
    echo "$file"
    # fpcalc tend to not be happy with stdin, but still generate a fingerprint -\o/-
    if [ "$bin" == "fpcalc" ]; then
      ffmpeg -nostdin -i "$file" -f s16le -ar 11025 -ac 1 pipe:1 2>/dev/null | fpcalc -format s16le -rate 11025 -channels 1  - | grep FINGERPRINT | sed 's/FINGERPRINT=//' >/dev/null
    else
      ffmpeg -nostdin -i "$file" -f s16le -ar 11025 -ac 1 pipe:1 2>/dev/null | ./bin/sporeprint >/dev/null
    fi
  done