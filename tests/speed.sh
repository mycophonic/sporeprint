#!/usr/bin/env bash

set -uo pipefail

folder="${1:?Usage: $0 <folder>}"
bin="${2:?Usage: $0 <fpcalc|sporeprint>}"

# Cross-platform CPU count: nproc (Linux) or sysctl (macOS)
ncpu=$(nproc 2>/dev/null || sysctl -n hw.ncpu 2>/dev/null || echo 4)

process_fpcalc() {
  ffmpeg -nostdin -i "$1" -f s16le -ar 11025 -ac 1 pipe:1 2>/dev/null |
    fpcalc -format s16le -rate 11025 -channels 1 - |
    grep FINGERPRINT |
    sed 's/FINGERPRINT=//' >/dev/null
}

process_sporeprint() {
  ffmpeg -nostdin -i "$1" -f s16le -ar 11025 -ac 1 pipe:1 2>/dev/null |
    ./bin/sporeprint >/dev/null
}

export -f process_fpcalc process_sporeprint

if [ "$bin" == "fpcalc" ]; then
  find "$folder" -type f \( -iname "*.flac" -o -iname "*.mp3" -o -iname "*.m4a" -o -iname "*.mp4" \) -print0 |
    xargs -0 -P "$ncpu" -n 1 bash -c 'process_fpcalc "$1"' _
else
  find "$folder" -type f \( -iname "*.flac" -o -iname "*.mp3" -o -iname "*.m4a" -o -iname "*.mp4" \) -print0 |
    xargs -0 -P "$ncpu" -n 1 bash -c 'process_sporeprint "$1"' _
fi
