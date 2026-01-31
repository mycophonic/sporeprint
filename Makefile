CGO_ENABLED := 1
NAME := sporeprint
GOFLAGS := -tags=cgo,netgo,osusergo,static_build

include hack/common.mk

##########################
# Chromaprint
##########################

CHROMAPRINT_VERSION := 1.6.0
CHROMAPRINT_BUILD_DIR := tmp/chromaprint
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
	@mkdir -p $(CHROMAPRINT_BUILD_DIR) bin
	@curl -fsSL "https://github.com/acoustid/chromaprint/releases/download/v$(CHROMAPRINT_VERSION)/chromaprint-$(CHROMAPRINT_VERSION).tar.gz" \
		| tar xz -C $(CHROMAPRINT_BUILD_DIR)
	@echo "=== Building Chromaprint (static, KissFFT) ==="
	@cd $(CHROMAPRINT_BUILD_DIR)/chromaprint-$(CHROMAPRINT_VERSION) && \
		mkdir -p build && \
		cd build && \
		cmake .. \
			-DCMAKE_BUILD_TYPE=Release \
			-DCMAKE_C_FLAGS="$(C_CFLAGS_RELEASE)" \
			-DCMAKE_CXX_FLAGS="$(C_CXXFLAGS_RELEASE)" \
			-DCMAKE_EXE_LINKER_FLAGS="$(C_LDFLAGS)" \
			-DCMAKE_SHARED_LINKER_FLAGS="$(C_LDFLAGS)" \
			-DBUILD_SHARED_LIBS=OFF \
			-DBUILD_TOOLS=OFF \
			-DBUILD_TESTS=OFF \
			-DFFT_LIB=kissfft && \
		$(MAKE)
	@cp $(CHROMAPRINT_BUILD_DIR)/chromaprint-$(CHROMAPRINT_VERSION)/build/src/libchromaprint.a bin/
	@cp $(CHROMAPRINT_BUILD_DIR)/chromaprint-$(CHROMAPRINT_VERSION)/src/chromaprint.h bin/
	@echo "=== Chromaprint built: $(CHROMAPRINT_LIB) $(CHROMAPRINT_HEADER) ==="

clean-chromaprint: ## Clean Chromaprint build artifacts
	@rm -rf $(CHROMAPRINT_BUILD_DIR) $(CHROMAPRINT_LIB) $(CHROMAPRINT_HEADER)
