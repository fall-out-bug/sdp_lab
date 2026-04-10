# Discovery Hypothesis

**Raw idea:** AI Architect — компонент платформы SDP (Software Development Platform), который анализирует ВНЕШНИЕ репозитории на ЛЮБОМ языке программирования. Ключевые возможности: (1) Определение типа архитектуры целевого проекта — monolith, microservices, hexagonal, event-driven, clean architecture, CQRS, pipe-and-filter, plugin/extension и др. (2) Обнаружение паттернов проектирования — GoF, DDD (Aggregate Root, Value Object, Domain Event), инфраструктурные (Circuit Breaker, Saga, Outbox). (3) Детекция существующих нотаций и спецификаций — OpenAPI, AsyncAPI, Protobuf/gRPC, GraphQL, ADR, Dockerfile, Terraform, CI/CD configs. (4) Генерация C4 диаграмм в Mermaid для любого языка. (5) Создание и валидация контрактов на интеграции и данные. (6) Поддержка greenfield (ведение архитектурного разговора до написания кода) и brownfield (анализ существующей кодовой базы). (7) Разбиение больших кодовых баз на участки анализа (codebase > context window LLM). (8) Отслеживание нового кода и проверка на соответствие референсной архитектуре (conformance checking). (9) Подготовка brownfield проектов к AI Assisted разработке, а затем к полному AI Native. Целевая аудитория: команды разработки, архитекторы, tech leads, CTO. Компонент должен быть language-agnostic и работать с Go, Python, TypeScript, Java, Rust, C# и любыми другими языками.

## Test Card (Strategyzer)

**We believe** Software development teams need to quickly understand the architecture of new or unfamiliar external codebases because it accelerates onboarding and contribution.

**To verify this**, we will Conduct 5-7 customer interviews with software development team leads and architects, presenting mockups of an automated C4 diagram generation tool for polyglot repositories and asking them to describe how it would impact their onboarding process and architectural understanding.

**We'll measure** The percentage of interviewees who express a strong desire for such a tool and identify it as a significant improvement over current methods.

**We are right if** At least 70% of interviewees state that an automated C4 diagram generation tool for polyglot repositories would significantly improve their team's onboarding efficiency within 3 months.

## Assumptions (RAT-Ranked)

| Rank | Assumption | Risk | Uncertainty | RAT Score |
|------|-----------|------|-------------|----------|
| 1 | Software development teams are actively seeking solutions to improve onboarding efficiency for external codebases. | high | high | 9 |
| 2 | The effort required to integrate such a tool into existing workflows is acceptable to target users. | medium | high | 6 |
| 3 | Teams are willing to invest in a new tool specifically for architectural understanding and documentation. | high | medium | 6 |
| 4 | Automated C4 diagram generation for polyglot environments is perceived as a valuable and accurate representation of architecture. | medium | medium | 4 |
| 5 | Existing tools do not adequately address the need for language-agnostic architectural understanding and C4 diagram generation. | low | low | 1 |

**Riskiest assumption (rank 1):** Software development teams are actively seeking solutions to improve onboarding efficiency for external codebases.

## Requirements

- User can upload or connect to an external code repository (e.g., Git URL).
- System automatically analyzes code in multiple programming languages (e.g., Java, Python, JavaScript, Go).
- System generates C4 diagrams (Context, Container, Component, Code) based on the analyzed architecture.
- User can view and navigate generated C4 diagrams within a web interface.
- User can export generated C4 diagrams in common formats (e.g., PNG, SVG, PlantUML).
