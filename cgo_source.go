//go:build !lib

// Source mode: compile C sources directly with CGO.
// Usage: go build (default)
// Requires: CGO_CFLAGS_ALLOW="-mavx.*|-msse.*|-mfma|-mavx512.*"
package codec

/*
#cgo CFLAGS: -I${SRCDIR}/include -I${SRCDIR}/modules/platform/include -I${SRCDIR}/src -I${SRCDIR}/modules/platform/src -O2 -Wall -std=c11 -D_POSIX_C_SOURCE=200112L
#cgo LDFLAGS: -lm

#include "codec.h"
#include "fix_codec.h"

// Platform sources (dependency)
#include "modules/platform/src/simd_detect.c"
#include "modules/platform/src/mem_aligned.c"
#include "modules/platform/src/error.c"
#include "modules/platform/src/fc_init.c"

// Platform-specific sources
#if defined(__linux__)
  #include "modules/platform/src/platform_linux.c"
#elif defined(__APPLE__)
  #include "modules/platform/src/platform_macos.c"
#elif defined(_WIN32)
  #include "modules/platform/src/platform_win.c"
#endif

// Codec sources
#include "fix_codec.c"

// Forward declarations for fc_init/fc_cleanup (no public header)
int fc_init(void);
void fc_cleanup(void);
*/
import "C"
