#!/usr/bin/env bash

set -e

# Minimal Chromaprint build - MIT compatible only (KissFFT, no FFmpeg/FFTW3)

VERSION="1.6.0"
DEST_DIR="bin"
BUILD_DIR="tmp"
rm -Rf "$BUILD_DIR"
mkdir "$BUILD_DIR"
mkdir "$DEST_DIR"
cd "$BUILD_DIR"

echo "=== Fetching Chromaprint ${VERSION} ==="
curl -fsSL "https://github.com/acoustid/chromaprint/releases/download/v${VERSION}/chromaprint-${VERSION}.tar.gz" -o chromaprint.tar.gz
tar xzf chromaprint.tar.gz
cd "chromaprint-${VERSION}"

mkdir -p build && cd build

cmake .. \
    -DCMAKE_BUILD_TYPE=Release \
    -DBUILD_SHARED_LIBS=OFF \
    -DBUILD_TOOLS=OFF \
    -DBUILD_TESTS=OFF \
    -DFFT_LIB=kissfft

make

cp "$(pwd)/src/libchromaprint.a" ../../../"$DEST_DIR"/
cp "$(dirname "$(pwd)")/src/chromaprint.h" ../../../"$DEST_DIR"/
