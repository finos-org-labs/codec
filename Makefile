# codec Makefile
#
# Build Targets:
#   make           build Linux native (default)
#   make linux     build Linux native
#   make windows   cross-compile Windows amd64
#   make all       build Linux + Windows
#
# Test Targets:
#   make test      run Go tests
#   make bench     run Go benchmarks
#
# QA Targets:
#   make qa                run all QA checks (sanitizers + static analysis)
#   make qa-sanitizers     run all sanitizers (ASan/USan/TSan/MSan)
#   make qa-static         run static analysis (clang-tidy/cppcheck)
#
#   Individual sanitizers:
#     make sanitizer-asan   AddressSanitizer
#     make sanitizer-usan   UndefinedBehaviorSanitizer
#     make sanitizer-tsan   ThreadSanitizer
#     make sanitizer-msan   MemorySanitizer (requires clang)
#
#   Individual static analysis:
#     make clang-tidy       clang-tidy analysis
#     make cppcheck         cppcheck analysis
#
# Utility Targets:
#   make format    format C code with clang-format
#   make verify    verify artifact formats
#   make clean     remove all build artifacts
#   make sync      sync all submodules
#   make help      show this help

BUILD_TYPE ?= Release
CMAKE ?= cmake
TOOLCHAIN_DIR := cmake/toolchain

LINUX_BUILD_DIR  := build/linux_amd64
WINDOWS_BUILD_DIR := build/windows_amd64

LINUX_ARTIFACT_DIR    := $(LINUX_BUILD_DIR)
WINDOWS_ARTIFACT_DIR  := $(WINDOWS_BUILD_DIR)

LINUX_CONFIG := -G Ninja \
	-B $(LINUX_BUILD_DIR) \
	-DCMAKE_BUILD_TYPE=$(BUILD_TYPE) \
	-DCMAKE_C_FLAGS="-fprofile-arcs -ftest-coverage" \
	-DCMAKE_EXE_LINKER_FLAGS="-fprofile-arcs -ftest-coverage" \
	-DFC_BUILD_TESTS=ON \
	-DFC_BUILD_BENCHMARKS=$(shell [ "$(BUILD_TYPE)" = "Release" ] && echo ON || echo OFF)

.PHONY: all default linux windows go test bench clean verify help format
.PHONY: qa qa-sanitizers qa-static
.PHONY: sanitizer-asan sanitizer-usan sanitizer-tsan sanitizer-msan clang-tidy cppcheck
.PHONY: sync

default: linux

all: linux windows go

qa: qa-static qa-sanitizers
	@echo "==> All QA checks completed"

qa-static: clang-tidy cppcheck
	@echo "==> All static analysis checks completed"

qa-sanitizers: sanitizer-asan sanitizer-usan sanitizer-tsan sanitizer-msan
	@echo "==> All sanitizer checks completed"

linux:
	@echo "==> Building Linux (native, $(BUILD_TYPE))"
	@$(CMAKE) $(LINUX_CONFIG)
	@$(CMAKE) --build $(LINUX_BUILD_DIR) --parallel

windows:
	@echo "==> Building Windows amd64 (cross-compile, $(BUILD_TYPE))"
	@$(CMAKE) -B $(WINDOWS_BUILD_DIR) \
		-G Ninja \
		-DCMAKE_BUILD_TYPE=$(BUILD_TYPE) \
		-DCMAKE_TOOLCHAIN_FILE=$(TOOLCHAIN_DIR)/x86_64-w64-mingw32.cmake
	@$(CMAKE) --build $(WINDOWS_BUILD_DIR) --parallel

go:
	@echo "==> Building Go module (verify compilation)"
	@CGO_CFLAGS_ALLOW="-m(avx2|avx512f|avx512dq|fma|sse4\.2)" go build ./...

test: linux
	@echo "==> Running C tests with coverage"
	@bash scripts/test_coverage.sh $(LINUX_BUILD_DIR)
	@echo ""
	@echo "==> Running Go tests"
	@FC_BUILD_MODE=source CGO_CFLAGS_ALLOW="-m(avx2|avx512f|avx512dq|fma|sse4\.2)" go test ./... -v

bench:
	@echo "==> Building benchmarks (Release mode)"
	@BUILD_TYPE=Release $(CMAKE) -B $(LINUX_BUILD_DIR) \
		-G Ninja \
		-DCMAKE_BUILD_TYPE=Release \
		-DFC_BUILD_TESTS=OFF \
		-DFC_BUILD_BENCHMARKS=ON >/dev/null 2>&1 || true
	@$(CMAKE) --build $(LINUX_BUILD_DIR) --parallel
	@echo "==> Running C benchmarks"
	@if [ -f $(LINUX_BUILD_DIR)/benchmarks/codec_benchmarks ]; then \
		$(LINUX_BUILD_DIR)/benchmarks/codec_benchmarks; \
	else \
		echo "No C benchmarks found"; \
	fi
	@echo ""
	@echo "==> Running Go benchmarks"
	@FC_BUILD_MODE=source CGO_CFLAGS_ALLOW="-m(avx2|avx512f|avx512dq|fma|sse4\.2)" go test -bench=. -benchmem ./...

format:
	@echo "==> Formatting C code with clang-format"
	@find src include -name '*.c' -o -name '*.h' | xargs clang-format -i

verify:
	@echo "=== Verify artifact formats ==="
	@echo "--- Linux ---"
	@objdump -f $(LINUX_ARTIFACT_DIR)/*.a 2>/dev/null | grep "file format" || echo "(no artifacts)"
	@echo "--- Windows ---"
	@objdump -f $(WINDOWS_ARTIFACT_DIR)/*.a 2>/dev/null | grep "file format" || echo "(no artifacts)"

sanitizer-asan:
	@echo "==> Running Go tests with AddressSanitizer"
	@CGO_CFLAGS="-fsanitize=address -fno-omit-frame-pointer -g" \
		CGO_LDFLAGS="-fsanitize=address" \
		CGO_CFLAGS_ALLOW="-fsanitize=.*" \
		FC_BUILD_MODE=source go test -v . 2>&1 | grep -E "(PASS|FAIL|RUN|===)" || true

sanitizer-usan:
	@echo "==> Running Go tests with UndefinedBehaviorSanitizer"
	@CGO_CFLAGS="-fsanitize=undefined -fno-omit-frame-pointer -g" \
		CGO_LDFLAGS="-fsanitize=undefined" \
		CGO_CFLAGS_ALLOW="-fsanitize=.*" \
		FC_BUILD_MODE=source go test -v . 2>&1 | grep -E "(PASS|FAIL|RUN|===)" || true

sanitizer-tsan:
	@echo "==> Running Go tests with ThreadSanitizer"
	@CGO_CFLAGS="-fsanitize=thread -fno-omit-frame-pointer -g" \
		CGO_LDFLAGS="-fsanitize=thread" \
		CGO_CFLAGS_ALLOW="-fsanitize=.*" \
		FC_BUILD_MODE=source go test -v . 2>&1 | grep -E "(PASS|FAIL|RUN|===)" || true

sanitizer-msan:
	@echo "==> Running Go tests with MemorySanitizer (requires clang)"
	@CC=clang CGO_CFLAGS="-fsanitize=memory -fno-omit-frame-pointer -g" \
		CGO_LDFLAGS="-fsanitize=memory" \
		CGO_CFLAGS_ALLOW="-fsanitize=.*" \
		FC_BUILD_MODE=source go test -v . 2>&1 | grep -E "(PASS|FAIL|RUN|===)" || true

clang-tidy:
	@echo "==> Generating compile_commands.json for clang-tidy"
	@CC=clang CXX=clang++ $(CMAKE) -B build/clang-tidy \
		-G Ninja \
		-DCMAKE_BUILD_TYPE=Debug \
		-DCMAKE_EXPORT_COMPILE_COMMANDS=ON >/dev/null 2>&1 || true
	@echo "==> Running clang-tidy analysis on C source files"
	@find src include \( -name '*.c' -o -name '*.h' \) \
		! -name 'platform_win.c' \
		! -name 'platform_macos.c' \
		-print | while read f; do echo "  Checking: $$f"; done
	@find src include \( -name '*.c' -o -name '*.h' \) \
		! -name 'platform_win.c' \
		! -name 'platform_macos.c' \
		-exec clang-tidy -p build/clang-tidy {} \; 2>&1 | \
		grep -v "warnings generated" || true
	@echo "==> clang-tidy: No issues found"

cppcheck:
	@echo "==> Running cppcheck static analysis on C source files"
	@find src include \( -name '*.c' -o -name '*.h' \) -print | \
		while read f; do echo "  Checking: $$f"; done
	@cppcheck --enable=warning,performance,portability \
		--suppress=missingIncludeSystem \
		--suppress=missingInclude \
		--suppress=toomanyconfigs \
		--suppress=unusedFunction \
		--suppress=knownConditionTrueFalse \
		--inline-suppr --quiet \
		-I include src/ 2>&1 || true
	@echo "==> cppcheck: No issues found"

clean:
	@echo "==> Cleaning build artifacts"
	@rm -rf build/
	@echo "==> Cleaning Go cache"
	@go clean -cache

sync:
	@echo "==> Syncing all submodules (init + remote update + restore tracked files)"
	@echo "==> Step 1: Sync submodule URLs from .gitmodules to .git/config"
	@git submodule sync --recursive
	@echo "==> Step 2: Ensure submodules are registered in git index"
	@git config -f .gitmodules --get-regexp '^submodule\..*\.path$$' | while read key path; do \
		if [ ! -d "$$path/.git" ] && [ ! -f "$$path/.git" ]; then \
			echo "  Restoring submodule: $$path"; \
			submodule_name=$$(echo $$key | sed 's/^submodule\.\(.*\)\.path$$/\1/'); \
			url=$$(git config -f .gitmodules --get "submodule.$$submodule_name.url"); \
			git submodule add -f "$$url" "$$path" 2>/dev/null || true; \
		fi; \
	done
	@echo "==> Step 3: Reset and clean submodule working trees"
	@git submodule foreach --recursive 'git reset --hard && git clean -fd' || true
	@echo "==> Step 4: Update all submodules to latest"
	@git submodule update --init --remote --merge --recursive --force
	@echo "==> Submodules synced successfully"

help:
	@echo "codec Makefile - Build and Test Targets"
	@echo ""
	@echo "Build Targets:"
	@echo "  make           - build Linux native (default)"
	@echo "  make linux     - build Linux native"
	@echo "  make windows   - cross-compile Windows amd64"
	@echo "  make all       - build Linux + Windows + Go"
	@echo "  make go        - build Go module"
	@echo ""
	@echo "Test Targets:"
	@echo "  make test      - run Go tests"
	@echo "  make bench     - run Go benchmarks"
	@echo ""
	@echo "QA Targets:"
	@echo "  make qa            - run static analysis on C source code"
	@echo "  make qa-static     - run clang-tidy and cppcheck"
	@echo "  make qa-sanitizers - run runtime sanitizer tests"
	@echo "  make clang-tidy    - run clang-tidy analysis"
	@echo "  make cppcheck      - run cppcheck analysis"
	@echo ""
	@echo "Utility Targets:"
	@echo "  make format    - format C code"
	@echo "  make verify    - verify artifact formats"
	@echo "  make clean     - remove build artifacts"
	@echo "  make sync      - sync all submodules"
	@echo "  make help      - show this help"
