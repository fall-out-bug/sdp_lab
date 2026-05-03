---
name: cpp-dev
description: C/C++ development, testing, and quality assurance. Use for systems, embedded, game engines, high-performance computing, and native libraries.
---

# C/C++ Development

## Top 10 Patterns

1. **Google Test + Google Mock** — industry standard
2. **Catch2 / doctest** — lightweight, header-only alternatives
3. **CMake + CTest** — build and test orchestration
4. **Clang Static Analyzer / Clang-Tidy** — static analysis
5. **Valgrind / AddressSanitizer** — memory error detection
6. **cppcheck** — fast static analysis
7. **gcov + lcov** — code coverage
8. **Google Benchmark** — performance benchmarks
9. **Conan / vcpkg** — dependency management
10. **Doxygen** — documentation generation

## Quality Gates

```bash
mkdir build && cd build
cmake .. -DCMAKE_BUILD_TYPE=Debug
make -j$(nproc)
ctest --output-on-failure
clang-tidy ../src/*.cpp -- -I../include
cppcheck --enable=all --suppress=missingIncludeSystem ../src
```

## Key Tools

| Tool | Purpose |
|------|---------|
| Google Test / Catch2 | Testing |
| CMake / Meson / Bazel | Build |
| clang-tidy / cppcheck | Static analysis |
| Valgrind / ASan | Memory safety |
| gcov / lcov | Coverage |
| Google Benchmark | Benchmarks |
| Conan / vcpkg | Dependencies |
| Doxygen | Docs |
