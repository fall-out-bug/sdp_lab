# Discovery Frame

**Raw idea:** AI Architect — компонент платформы SDP (Software Development Platform), который анализирует ВНЕШНИЕ репозитории на ЛЮБОМ языке программирования. Ключевые возможности: (1) Определение типа архитектуры целевого проекта — monolith, microservices, hexagonal, event-driven, clean architecture, CQRS, pipe-and-filter, plugin/extension и др. (2) Обнаружение паттернов проектирования — GoF, DDD (Aggregate Root, Value Object, Domain Event), инфраструктурные (Circuit Breaker, Saga, Outbox). (3) Детекция существующих нотаций и спецификаций — OpenAPI, AsyncAPI, Protobuf/gRPC, GraphQL, ADR, Dockerfile, Terraform, CI/CD configs. (4) Генерация C4 диаграмм в Mermaid для любого языка. (5) Создание и валидация контрактов на интеграции и данные. (6) Поддержка greenfield (ведение архитектурного разговора до написания кода) и brownfield (анализ существующей кодовой базы). (7) Разбиение больших кодовых баз на участки анализа (codebase > context window LLM). (8) Отслеживание нового кода и проверка на соответствие референсной архитектуре (conformance checking). (9) Подготовка brownfield проектов к AI Assisted разработке, а затем к полному AI Native. Целевая аудитория: команды разработки, архитекторы, tech leads, CTO. Компонент должен быть language-agnostic и работать с Go, Python, TypeScript, Java, Rust, C# и любыми другими языками.

## Problem Statement

Software development teams, architects, tech leads, and CTOs struggle to efficiently understand, document, and maintain the architecture of external code repositories, especially in polyglot environments. This leads to difficulties in onboarding, architectural drift, inconsistent integration contracts, and challenges in preparing brownfield projects for AI-assisted development. Existing tools often lack language-agnostic capabilities, comprehensive architectural pattern detection, and automated C4 diagram generation, making it time-consuming and error-prone to gain a holistic view of a system's design and ensure architectural conformance.

## Jobs to Be Done

- Development teams need to quickly understand the architecture of new or unfamiliar external codebases to accelerate onboarding and contribution.
- Architects need to automatically identify architectural styles, design patterns, and existing specifications within diverse code repositories to ensure consistency and guide future development.
- Tech leads need to track new code and validate its adherence to reference architectures to prevent architectural drift and maintain system integrity.
- CTOs need to prepare brownfield projects for AI-assisted and AI-native development by gaining a structured understanding of their existing architecture and integration points.
- Developers need to generate and validate integration and data contracts across different services and languages to ensure seamless communication.
- Anyone involved in software development needs to visualize complex system architectures through automatically generated C4 diagrams to improve communication and understanding.

**Appetite:** large

**Scope:** A language-agnostic AI-powered component within an SDP that analyzes external code repositories to identify architectural styles, design patterns, and specifications, generates C4 diagrams, creates/validates contracts, and supports both greenfield and brownfield architectural analysis and conformance checking, with a focus on preparing projects for AI-assisted development.
