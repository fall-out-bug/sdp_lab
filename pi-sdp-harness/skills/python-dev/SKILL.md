---
name: python-dev
description: Python development, testing, and quality assurance. Use for web backends, data science, ML, scripting, and automation.
---

# Python Development

## Top 10 Patterns

1. **pytest fixtures + parametrize** — reusable test setup
2. **pytest-xdist** — parallel test execution
3. **hypothesis** — property-based testing
4. **mypy / pyright** — static type checking
5. **ruff** — ultra-fast linter and formatter (replaces flake8, black, isort)
6. **pydantic v2** — runtime validation + serialization
7. **tox / nox** — test across Python versions
8. **pytest-cov + coverage.py** — coverage reporting
9. **Factory Boy / Faker** — test data generation
10. **Locust / k6** — load testing for Python web apps

## Quality Gates

```bash
ruff check . && ruff format --check .
mypy .
pytest -xvs --cov=src --cov-report=term-missing
```

## Key Tools

| Category | Tools |
|----------|-------|
| Test | pytest, unittest, doctest |
| Mock | unittest.mock, pytest-mock, responses, httpxmock |
| Integration | testcontainers-python, pytest-docker |
| E2E | Selenium, Playwright, Behave |
| Perf | pytest-benchmark, locust, k6 |
| Type | mypy, pyright, pydantic |
| Lint | ruff, pylint, bandit |
| Env | tox, nox, hatch, poetry |
