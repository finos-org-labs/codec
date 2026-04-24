cmake_minimum_required(VERSION 3.16)
project(finkit-codec VERSION 1.0.0 LANGUAGES C)

set(CMAKE_C_STANDARD 11)
set(CMAKE_C_STANDARD_REQUIRED ON)

# Compile options
add_compile_options(-Wall -Wextra -Wpedantic -ffp-contract=off)

# Add platform submodule
add_subdirectory(modules/platform EXCLUDE_FROM_ALL)

# Codec module sources
set(CODEC_SOURCES
    src/fix_codec.c
)

# Create object library
add_library(finkit_codec OBJECT ${CODEC_SOURCES})

target_include_directories(finkit_codec
    PUBLIC
        $<BUILD_INTERFACE:${CMAKE_CURRENT_SOURCE_DIR}/include>
        $<BUILD_INTERFACE:${CMAKE_CURRENT_SOURCE_DIR}/modules/platform/include>
        $<INSTALL_INTERFACE:include>
)

target_compile_definitions(finkit_codec PRIVATE FC_STATIC_BUILD)

# Set compile flags for SIMD variants in fix_codec.c
if(CMAKE_C_COMPILER_ID MATCHES "GNU|Clang")
    set_source_files_properties(src/fix_codec.c PROPERTIES COMPILE_FLAGS "-msse4.2 -mavx2")
elseif(MSVC)
    set_source_files_properties(src/fix_codec.c PROPERTIES COMPILE_FLAGS "/arch:AVX2")
endif()

# Create static library
add_library(finkit_codec_static STATIC $<TARGET_OBJECTS:finkit_codec>)
target_include_directories(finkit_codec_static
    PUBLIC
        $<BUILD_INTERFACE:${CMAKE_CURRENT_SOURCE_DIR}/include>
        $<INSTALL_INTERFACE:include>
)

# Link platform and math library
target_link_libraries(finkit_codec_static PUBLIC finkit_platform_static m)

# Create alias
add_library(finkit::codec ALIAS finkit_codec_static)

# Install
install(TARGETS finkit_codec_static
    EXPORT finkit-codec-targets
    ARCHIVE DESTINATION lib
)

install(DIRECTORY include/
    DESTINATION include/codec
    FILES_MATCHING PATTERN "*.h"
)

install(EXPORT finkit-codec-targets
    FILE finkit-codecTargets.cmake
    NAMESPACE finkit::
    DESTINATION lib/cmake/finkit-codec
)

# Config files
include(CMakePackageConfigHelpers)
write_basic_package_version_file(
    "${CMAKE_CURRENT_BINARY_DIR}/finkit-codecConfigVersion.cmake"
    VERSION ${PROJECT_VERSION}
    COMPATIBILITY AnyNewerVersion
)

configure_package_config_file(
    "${CMAKE_CURRENT_SOURCE_DIR}/cmake/finkit-codecConfig.cmake.in"
    "${CMAKE_CURRENT_BINARY_DIR}/finkit-codecConfig.cmake"
    INSTALL_DESTINATION lib/cmake/finkit-codec
)

install(FILES
    "${CMAKE_CURRENT_BINARY_DIR}/finkit-codecConfig.cmake"
    "${CMAKE_CURRENT_BINARY_DIR}/finkit-codecConfigVersion.cmake"
    DESTINATION lib/cmake/finkit-codec
)

# Tests
option(FC_BUILD_TESTS "Build tests" ON)
if(FC_BUILD_TESTS AND EXISTS ${CMAKE_CURRENT_SOURCE_DIR}/tests)
    enable_testing()
    add_subdirectory(tests)
endif()

# Benchmarks
option(FC_BUILD_BENCHMARKS "Build benchmarks" ON)
if(FC_BUILD_BENCHMARKS AND EXISTS ${CMAKE_CURRENT_SOURCE_DIR}/benchmarks)
    add_subdirectory(benchmarks)
endif()

# Print configuration
message(STATUS "")
message(STATUS "============================================================")
message(STATUS " finkit-codec ${PROJECT_VERSION}")
message(STATUS "============================================================")
message(STATUS "  C Compiler: ${CMAKE_C_COMPILER_ID} ${CMAKE_C_COMPILER_VERSION}")
message(STATUS "  Build Type: ${CMAKE_BUILD_TYPE}")
message(STATUS "  System: ${CMAKE_SYSTEM_NAME}")
message(STATUS "============================================================")
message(STATUS "")
