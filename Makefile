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
