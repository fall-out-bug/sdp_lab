---
name: jvm-dev
description: Full-spectrum JVM development (Java, Kotlin, Scala, Groovy, Clojure). Use for Spring, Quarkus, Micronaut, Android, Gradle plugins, and multi-module builds.
---

# JVM Development

## When to Use

- Any `.java`, `.kt`, `.scala`, `.groovy`, `.clj` file changes
- Maven or Gradle build configuration
- Spring Boot / Quarkus / Micronaut / Vert.x applications
- Android development
- Multi-module / multi-platform builds
- JVM library publishing

## Languages

| Language | Primary Use | Build Tool |
|----------|-------------|------------|
| Java | Enterprise, Android | Maven, Gradle |
| Kotlin | Android, server, DSL | Gradle (KTS) |
| Scala | Big data, FP | SBT, Mill |
| Groovy | Gradle scripts, testing | Gradle |
| Clojure | FP, REPL-driven | Leiningen, deps.edn |

## Top 10 Development Patterns

### 1. TestContainers for Integration Tests
```kotlin
@Testcontainers
class PostgresTest {
    @Container
    val postgres = PostgreSQLContainer("postgres:16")
}
```

### 2. Architecture Testing (ArchUnit)
```java
ArchRuleDefinition.classes()
    .that().resideInAPackage("..service..")
    .should().onlyBeAccessed().byAnyPackage("..controller..", "..service..");
```

### 3. Mutation Testing (PIT)
```bash
./mvnw org.pitest:pitest-maven:mutationCoverage
```

### 4. Property-Based Testing (jqwik / JUnit-Quickcheck)
```java
@Property
boolean absoluteValueOfIntegerIsPositive(@ForAll int anInteger) {
    return Math.abs(anInteger) >= 0;
}
```

### 5. Contract Testing (Spring Cloud Contract / Pact)
```bash
./mvnw spring-cloud-contract:generateTests
```

### 6. Benchmarking (JMH)
```java
@BenchmarkMode(Mode.Throughput)
@State(Scope.Thread)
public class MyBenchmark { ... }
```

### 7. GraalVM Native Image
```bash
./mvnw native:compile -Pnative
```

### 8. Gradle Version Catalogs
```toml
[versions]
kotlin = "1.9.22"
[libraries]
kotlin-stdlib = { module = "org.jetbrains.kotlin:kotlin-stdlib", version.ref = "kotlin" }
```

### 9. Gradle Convention Plugins
Write `buildSrc/src/main/kotlin/java-conventions.gradle.kts` for shared config.

### 10. Kotest / Spek / Spock for Expressive Tests
```kotlin
class MyTests : FunSpec({
    test("String length returns correct value") {
        "sammy".length shouldBe 5
    }
})
```

## Testing Stack

| Layer | Java | Kotlin | Scala |
|-------|------|--------|-------|
| Unit | JUnit 5, TestNG | JUnit 5, Kotest | ScalaTest, MUnit |
| Integration | TestContainers, Spring Test | TestContainers, MockK | ScalaTest + Docker |
| E2E | Selenium, Playwright | Playwright, RestAssured | Playwright |
| Contract | Pact, Spring Cloud Contract | Pact | Pact |
| Mutation | PIT | PIT | N/A |
| Perf | JMH, Gatling | JMH, k6 | JMH |
| A11y | Deque axe Selenium | Same | Same |
| Property | jqwik, JUnit-Quickcheck | jqwik, Kotest property | ScalaCheck |

## Quality Gates

```bash
# Maven
./mvnw verify -Pit

# Gradle
./gradlew check koverXmlReport

# Include: compile, test, integrationTest, pitest, detekt/ktlint, spotbugs
```

## Key Files

- `pom.xml` / `build.gradle.kts` / `build.sbt` — build config
- `src/test/resources/application-test.yml` — test properties
- `gradle/libs.versions.toml` — version catalog
- `src/test/java/../ArchitectureTest.java` — ArchUnit rules
- `.kover/report.xml` — coverage
