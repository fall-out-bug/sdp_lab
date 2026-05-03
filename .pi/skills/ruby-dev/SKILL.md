---
name: ruby-dev
description: Ruby / Rails development, testing, and quality assurance. Use for web apps (Rails, Sinatra), automation, DevOps tools, and scripting.
---

# Ruby Development

## Top 10 Patterns

1. **RSpec + FactoryBot + Faker** — the golden trinity
2. **Capybara + Cuprite / Selenium** — feature / system tests
3. **VCR + WebMock** — HTTP interaction recording
4. **Shoulda Matchers** — one-liner validations and associations
5. **SimpleCov** — code coverage
6. **RuboCop + StandardRB** — linting and formatting
7. **Brakeman** — security static analysis for Rails
8. **Bullet** — N+1 query detection
9. **Parallel Tests** — `parallel_tests` gem for CI speed
10. **Dry-rb ecosystem** — dry-validation, dry-transaction, dry-monads

## Quality Gates

```bash
bundle exec rubocop -P
bundle exec brakeman -q -w2
bundle exec rspec
bundle exec rails db:environment:set RAILS_ENV=test db:drop db:create db:migrate
```

## Key Tools

| Tool | Purpose |
|------|---------|
| RSpec / Minitest | Test frameworks |
| FactoryBot | Test data |
| Capybara | Feature tests |
| Cuprite / Selenium | Browser drivers |
| VCR | HTTP mocking |
| SimpleCov | Coverage |
| RuboCop / Standard | Lint/format |
| Brakeman | Security |
| Flog / Flay | Complexity detection |
