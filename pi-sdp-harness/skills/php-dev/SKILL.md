---
name: php-dev
description: PHP development, testing, and quality assurance. Use for Laravel, Symfony, WordPress plugins, and modern PHP 8+ applications.
---

# PHP Development

## Top 10 Patterns

1. **PHPUnit + Pest** — Pest for elegance, PHPUnit for power
2. **Pest Architecture Plugin** — enforce layer boundaries
3. **Laravel Pint / PHP CS Fixer** — formatting
4. **PHPStan / Psalm** — static analysis at max level
5. **Laravel Dusk / Symfony Panther** — browser testing
6. **Mockery** — advanced mocking
7. **Infection** — mutation testing for PHP
8. **PHPBench** — benchmarking
9. **Laravel Sail / Docker** — dev environment parity
10. **FrankenPHP / RoadRunner** — modern PHP servers

## Quality Gates

```bash
vendor/bin/pest --parallel
vendor/bin/phpstan analyse --memory-limit=512M
vendor/bin/pint
vendor/bin/infection --min-msi=80
```

## Key Tools

| Tool | Purpose |
|------|---------|
| PHPUnit / Pest | Testing |
| PHPStan / Psalm | Static analysis |
| PHP CS Fixer / Pint | Formatting |
| Infection | Mutation testing |
| Mockery | Mocking |
| Laravel Dusk / Panther | E2E |
| PHPBench | Benchmarks |
| Deptrac | Architecture testing |
