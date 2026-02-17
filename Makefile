#   Copyright Mycophonic.

#   Licensed under the Apache License, Version 2.0 (the "License");
#   you may not use this file except in compliance with the License.
#   You may obtain a copy of the License at

#       http://www.apache.org/licenses/LICENSE-2.0

#   Unless required by applicable law or agreed to in writing, software
#   distributed under the License is distributed on an "AS IS" BASIS,
#   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
#   See the License for the specific language governing permissions and
#   limitations under the License.

CGO_ENABLED := 1
NAME := sporeprint
GOFLAGS := -tags=cgo,netgo,osusergo,static_build
COVER_MIN := 35

include hack/common.mk

##########################
# Chromaprint
##########################

CHROMAPRINT_VERSION := 1.6.0
CHROMAPRINT_BUILD_DIR := bin/tmp/chromaprint
CHROMAPRINT_LIB := bin/libchromaprint.a
CHROMAPRINT_HEADER := bin/chromaprint.h

# Build targets depend on Chromaprint being built first.
build-sporeprint: $(CHROMAPRINT_LIB)
build-debug-sporeprint: $(CHROMAPRINT_LIB)
build-static-sporeprint: $(CHROMAPRINT_LIB)

chromaprint: $(CHROMAPRINT_LIB) $(CHROMAPRINT_HEADER) ## Build Chromaprint static library (MIT, KissFFT)

$(CHROMAPRINT_LIB) $(CHROMAPRINT_HEADER):
	@echo "=== Fetching Chromaprint $(CHROMAPRINT_VERSION) ==="
	@rm -rf $(CHROMAPRINT_BUILD_DIR)
	@mkdir -p bin
	@git clone --branch v$(CHROMAPRINT_VERSION) --depth 1 \
		https://github.com/acoustid/chromaprint.git \
		$(CHROMAPRINT_BUILD_DIR)/chromaprint-$(CHROMAPRINT_VERSION)
	@echo "=== Building Chromaprint (static, KissFFT) ==="
	@cd $(CHROMAPRINT_BUILD_DIR)/chromaprint-$(CHROMAPRINT_VERSION) && \
		mkdir -p build && \
		cd build && \
		cmake .. \
			$(CMAKE_GENERATOR) \
			-DCMAKE_BUILD_TYPE=Release \
			-DCMAKE_C_FLAGS="$(C_CFLAGS_RELEASE)" \
			-DCMAKE_CXX_FLAGS="$(C_CXXFLAGS_RELEASE)" \
			-DCMAKE_EXE_LINKER_FLAGS="$(C_LDFLAGS)" \
			-DCMAKE_SHARED_LINKER_FLAGS="$(C_LDFLAGS)" \
			-DBUILD_SHARED_LIBS=OFF \
			-DBUILD_TOOLS=OFF \
			-DBUILD_TESTS=OFF \
			-DFFT_LIB=kissfft && \
		cmake --build . --config Release
	@cp $(CHROMAPRINT_BUILD_DIR)/chromaprint-$(CHROMAPRINT_VERSION)/build/src/libchromaprint.a bin/
	@cp $(CHROMAPRINT_BUILD_DIR)/chromaprint-$(CHROMAPRINT_VERSION)/src/chromaprint.h bin/
	@echo "=== Chromaprint built: $(CHROMAPRINT_LIB) $(CHROMAPRINT_HEADER) ==="

clean-chromaprint: ## Clean Chromaprint build artifacts
	@rm -rf $(CHROMAPRINT_BUILD_DIR) $(CHROMAPRINT_LIB) $(CHROMAPRINT_HEADER)

##########################
# fpcalc (test tooling)
##########################

# fpcalc is built from the same Chromaprint source but with BUILD_TOOLS=ON.
# It requires FFmpeg libraries for audio decoding.
# This is test-only tooling — it is NOT part of the production build.
#
# Set FFMPEG_DIR to override FFmpeg location (e.g. brew --prefix ffmpeg-full).

FPCALC_BUILD_DIR := $(CHROMAPRINT_BUILD_DIR)/chromaprint-$(CHROMAPRINT_VERSION)/build-fpcalc

ifeq ($(OS),Windows_NT)
    FPCALC_BIN := bin/tests/fpcalc.exe
    # Windows: use MSYS2 MinGW64 FFmpeg if available.
    FFMPEG_DIR ?= $(firstword $(wildcard C:/msys64/mingw64))
else
    FPCALC_BIN := bin/tests/fpcalc
    # macOS: prefer ffmpeg-full over ffmpeg to avoid dyld version skew
    # between the two Homebrew formulas. Linux: leave empty so cmake
    # finds FFmpeg in system paths via apt-installed dev packages.
    BREW_PREFIX := $(shell brew --prefix 2>/dev/null)
    FFMPEG_DIR ?= $(firstword $(wildcard $(BREW_PREFIX)/opt/ffmpeg-full $(BREW_PREFIX)/opt/ffmpeg))
endif

fpcalc: $(FPCALC_BIN) ## Build fpcalc test tool (requires FFmpeg)

$(FPCALC_BIN): $(CHROMAPRINT_LIB)
	@echo "=== Building fpcalc (test tooling, requires FFmpeg) ==="
	@mkdir -p bin/tests
	@cd $(CHROMAPRINT_BUILD_DIR)/chromaprint-$(CHROMAPRINT_VERSION) && \
		rm -rf build-fpcalc && \
		mkdir -p build-fpcalc && \
		cd build-fpcalc && \
		cmake .. \
			$(CMAKE_GENERATOR) \
			-DCMAKE_BUILD_TYPE=Release \
			-DCMAKE_C_FLAGS="$(C_CFLAGS_RELEASE)" \
			-DCMAKE_CXX_FLAGS="$(C_CXXFLAGS_RELEASE)" \
			-DCMAKE_EXE_LINKER_FLAGS="$(C_LDFLAGS)" \
			-DCMAKE_SHARED_LINKER_FLAGS="$(C_LDFLAGS)" \
			$(if $(FFMPEG_DIR),-DFFMPEG_ROOT=$(FFMPEG_DIR)) \
			-DBUILD_SHARED_LIBS=OFF \
			-DBUILD_TOOLS=ON \
			-DBUILD_TESTS=OFF \
			-DFFT_LIB=kissfft && \
		cmake --build . --config Release --target fpcalc
	@cp $(FPCALC_BUILD_DIR)/src/cmd/fpcalc$(if $(filter Windows_NT,$(OS)),.exe) $(FPCALC_BIN)
	@echo "=== fpcalc built: $(FPCALC_BIN) ==="

# Test targets depend on fpcalc being built first.
test-unit: $(FPCALC_BIN)
test-unit-race: $(FPCALC_BIN)
test-unit-cover: $(FPCALC_BIN)

clean-fpcalc: ## Clean fpcalc build artifacts
	@rm -rf $(FPCALC_BUILD_DIR) $(FPCALC_BIN)
