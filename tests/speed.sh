#!/usr/bin/env bash

set -uo pipefail

folder="${1:?Usage: $0 <folder>}"
bin="${2:?Usage: $0 <fpcalc|sporeprint>}"

# find "$folder" -type f \( -iname "*.flac" -o -iname "*.mp3" -o -iname "*.m4a" -o -iname "*.mp4" \) -print0 |
#  while IFS= read -r -d '' file; do
#    echo "$file"
#    # fpcalc tend to return a not happy exit code with stdin, but still generate a fingerprint -\o/-
#    if [ "$bin" == "fpcalc" ]; then
#      ffmpeg -nostdin -i "$file" -f s16le -ar 11025 -ac 1 pipe:1 2>/dev/null | fpcalc -format s16le -rate 11025 -channels 1  - | grep FINGERPRINT | sed 's/FINGERPRINT=//' >/dev/null
#    else
#      ffmpeg -nostdin -i "$file" -f s16le -ar 11025 -ac 1 pipe:1 2>/dev/null | ./bin/sporeprint >/dev/null
#    fi
#  done

# Parallel version using xargs -P
# Cross-platform CPU count: nproc (Linux) or sysctl (macOS)
ncpu=$(nproc 2>/dev/null || sysctl -n hw.ncpu 2>/dev/null || echo 4)

if [ "$bin" == "fpcalc" ]; then
  find "$folder" -type f \( -iname "*.flac" -o -iname "*.mp3" -o -iname "*.m4a" -o -iname "*.mp4" \) -print0 |
    xargs -0 -P "$ncpu" -n 1 sh -c '
      ffmpeg -nostdin -i "$1" -f s16le -ar 11025 -ac 1 pipe:1 2>/dev/null | fpcalc -format s16le -rate 11025 -channels 1 - | grep FINGERPRINT | sed "s/FINGERPRINT=//" >/dev/null
    ' _
else
  find "$folder" -type f \( -iname "*.flac" -o -iname "*.mp3" -o -iname "*.m4a" -o -iname "*.mp4" \) -print0 |
    xargs -0 -P "$ncpu" -n 1 sh -c '
      ffmpeg -nostdin -i "$1" -f s16le -ar 11025 -ac 1 pipe:1 2>/dev/null | ./bin/sporeprint >/dev/null
    ' _
fi
