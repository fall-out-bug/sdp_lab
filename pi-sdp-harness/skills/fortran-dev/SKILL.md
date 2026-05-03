---
name: fortran-dev
description: Fortran development, testing, and quality assurance. Use for HPC, numerical weather prediction, computational physics, legacy modernization, and array-intensive computing.
---

# Fortran Development

## Top 10 Patterns

1. **CMake + Fortran** — cross-platform builds
2. **pFUnit** — unit testing framework
3. **Vegetables** — BDD testing for Fortran
4. **fprettify** — code formatting
5. **Fortls / fortls** — LSP for modern Fortran
6. **Coarrays / MPI / OpenMP** — parallel computing
7. **derived types + OOP** — `type, extends`, `class`
8. **modules + `use, only`** — explicit dependencies
9. **assumed-shape arrays** — `(:,:)` for safe interfaces
10. **FORD** — documentation generation

## Quality Gates

```bash
mkdir build && cd build
cmake .. -DCMAKE_Fortran_COMPILER=gfortran
make -j$(nproc)
ctest --output-on-failure
```

## Key Tools

| Tool | Purpose |
|------|---------|
| `pFUnit` | Unit testing |
| `Vegetables` | BDD testing |
| `fprettify` | Formatting |
| `ford` | Documentation |
| `gcov` | Coverage |
| `valgrind` | Memory |
| `CMake/CTest` | Build/test |
