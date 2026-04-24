/**
 * @file test_main.c
 * @brief Test runner entry point for codec module
 */

#include "test_framework.h"
#include <stdio.h>
#include <stdlib.h>

/* External test registration functions */
extern void register_codec_tests(void);

int main(int argc, char** argv) {
    /* Initialize test framework with command line arguments */
    fc_test_init_with_args(argc, argv);

    /* Register all test suites */
    register_codec_tests();

    int result = fc_test_run_all();

    /* Generate coverage report if requested */
    fc_test_generate_coverage_report();

    fc_test_cleanup();

    printf("\ncodec self-test: %s\n", result == 0 ? "PASS" : "FAIL");
    return result;
}
