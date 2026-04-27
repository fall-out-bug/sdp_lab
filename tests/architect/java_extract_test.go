package architect_test

import (
	"testing"

	"github.com/fall-out-bug/sdp_lab/internal/architect/extract"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// TestJavaExtractor_BasicImports
// ---------------------------------------------------------------------------

func TestJavaExtractor_BasicImports(t *testing.T) {
	dir := t.TempDir()

	// Java source file with several imports (including a static import).
	writeFile(t, dir, "src/main/java/com/example/App.java", `
package com.example;

import java.util.List;
import java.util.Map;
import static java.util.Collections.emptyList;
import com.google.common.collect.ImmutableList;
`)

	// Kotlin source file.
	writeFile(t, dir, "src/main/kotlin/com/example/Util.kt", `
package com.example

import kotlin.collections.listOf
import java.io.File
`)

	ext := &extract.JavaExtractor{}
	require.Equal(t, "java", ext.Name())

	result, err := ext.Extract(dir)
	require.NoError(t, err)

	// Verify metadata fields.
	assert.Equal(t, "java/kotlin/scala", result.Language)
	assert.Equal(t, "regex", result.ExtractionMethod)
	assert.InDelta(t, 0.70, result.AccuracyEstimate, 0.001)

	// Java imports grouped by package directory.
	javaPkg := "src/main/java/com/example"
	javaImports := result.ImportGraph.PackageImports[javaPkg]
	require.NotNil(t, javaImports, "expected imports for %s", javaPkg)
	assert.Contains(t, javaImports, "java.util.List")
	assert.Contains(t, javaImports, "java.util.Map")
	assert.Contains(t, javaImports, "java.util.Collections.emptyList")
	assert.Contains(t, javaImports, "com.google.common.collect.ImmutableList")

	// Kotlin imports grouped by package directory.
	ktPkg := "src/main/kotlin/com/example"
	ktImports := result.ImportGraph.PackageImports[ktPkg]
	require.NotNil(t, ktImports, "expected imports for %s", ktPkg)
	assert.Contains(t, ktImports, "kotlin.collections.listOf")
	assert.Contains(t, ktImports, "java.io.File")
}

// ---------------------------------------------------------------------------
// TestJavaExtractor_PomXml
// ---------------------------------------------------------------------------

func TestJavaExtractor_PomXml(t *testing.T) {
	dir := t.TempDir()

	// Minimal Java file so the extractor does not return "no source files".
	writeFile(t, dir, "src/main/java/App.java", `
package app;
import java.util.List;
`)

	writeFile(t, dir, "pom.xml", `<?xml version="1.0"?>
<project>
  <modelVersion>4.0.0</modelVersion>
  <groupId>com.example</groupId>
  <artifactId>my-app</artifactId>
  <version>1.0.0</version>
  <dependencies>
    <dependency>
      <groupId>org.springframework.boot</groupId>
      <artifactId>spring-boot-starter-web</artifactId>
    </dependency>
    <dependency>
      <groupId>com.google.guava</groupId>
      <artifactId>guava</artifactId>
      <scope>compile</scope>
    </dependency>
    <dependency>
      <groupId>junit</groupId>
      <artifactId>junit</artifactId>
      <scope>test</scope>
    </dependency>
  </dependencies>
</project>
`)

	ext := &extract.JavaExtractor{}
	result, err := ext.Extract(dir)
	require.NoError(t, err)

	require.NotNil(t, result.BuildSystem)
	assert.Equal(t, "maven", result.BuildSystem.Type)
	require.Len(t, result.BuildSystem.Dependencies, 3)

	// Verify first dependency.
	dep0 := result.BuildSystem.Dependencies[0]
	assert.Equal(t, "org.springframework.boot", dep0.Group)
	assert.Equal(t, "spring-boot-starter-web", dep0.Artifact)

	// Verify scoped dependency.
	dep2 := result.BuildSystem.Dependencies[2]
	assert.Equal(t, "junit", dep2.Group)
	assert.Equal(t, "junit", dep2.Artifact)
	assert.Equal(t, "test", dep2.Scope)
}

// ---------------------------------------------------------------------------
// TestJavaExtractor_BuildGradle
// ---------------------------------------------------------------------------

func TestJavaExtractor_BuildGradle(t *testing.T) {
	dir := t.TempDir()

	writeFile(t, dir, "src/main/java/App.java", `
package app;
import java.util.Map;
`)

	writeFile(t, dir, "build.gradle", `
plugins {
    id 'java'
}

dependencies {
    implementation 'org.springframework.boot:spring-boot-starter:3.2.0'
    api 'com.google.guava:guava:33.0.0-jre'
    testImplementation 'org.junit.jupiter:junit-jupiter:5.10.1'
    compileOnly 'org.projectlombok:lombok:1.18.30'
}
`)

	ext := &extract.JavaExtractor{}
	result, err := ext.Extract(dir)
	require.NoError(t, err)

	require.NotNil(t, result.BuildSystem)
	assert.Equal(t, "gradle", result.BuildSystem.Type)
	require.Len(t, result.BuildSystem.Dependencies, 4)

	groups := make(map[string]string)
	for _, d := range result.BuildSystem.Dependencies {
		groups[d.Artifact] = d.Group
	}
	assert.Equal(t, "org.springframework.boot", groups["spring-boot-starter"])
	assert.Equal(t, "com.google.guava", groups["guava"])
	assert.Equal(t, "org.junit.jupiter", groups["junit-jupiter"])
	assert.Equal(t, "org.projectlombok", groups["lombok"])
}

// ---------------------------------------------------------------------------
// TestJavaExtractor_SpringBootDetection
// ---------------------------------------------------------------------------

func TestJavaExtractor_SpringBootDetection(t *testing.T) {
	dir := t.TempDir()

	writeFile(t, dir, "src/main/java/com/example/Application.java", `
package com.example;

import org.springframework.boot.SpringApplication;
import org.springframework.boot.autoconfigure.SpringBootApplication;

public class Application {
    public static void main(String[] args) {
        SpringApplication.run(Application.class, args);
    }
}
`)

	writeFile(t, dir, "src/main/java/com/example/controller/UserController.java", `
package com.example.controller;

import org.springframework.web.bind.annotation.RestController;
import org.springframework.web.bind.annotation.GetMapping;
import java.util.List;

@RestController
public class UserController {
    @GetMapping("/users")
    public List<String> getUsers() { return List.of(); }
}
`)

	writeFile(t, dir, "src/main/java/com/example/service/UserService.java", `
package com.example.service;

import org.springframework.stereotype.Service;

@Service
public class UserService {}
`)

	writeFile(t, dir, "src/main/java/com/example/repo/UserRepo.java", `
package com.example.repo;

import org.springframework.stereotype.Repository;
import jakarta.persistence.Entity;

@Repository
public class UserRepo {}
`)

	ext := &extract.JavaExtractor{}
	result, err := ext.Extract(dir)
	require.NoError(t, err)

	require.NotEmpty(t, result.Frameworks, "expected at least one framework detected")

	var spring *extract.JavaFramework
	for i := range result.Frameworks {
		if result.Frameworks[i].Name == "Spring Boot" {
			spring = &result.Frameworks[i]
			break
		}
	}
	require.NotNil(t, spring, "Spring Boot framework should be detected")
	assert.Greater(t, spring.Confidence, 0.0)

	// Check specific evidence items are present.
	allEvidence := make(map[string]bool)
	for _, e := range spring.Evidence {
		allEvidence[e] = true
	}
	assert.True(t, allEvidence["Spring Boot application"], "should detect @SpringBootApplication")
	assert.True(t, allEvidence["Spring MVC REST controller"], "should detect @RestController")
	assert.True(t, allEvidence["Spring service component"], "should detect @Service")
	assert.True(t, allEvidence["Spring data repository"], "should detect @Repository")
	assert.True(t, allEvidence["JPA/Hibernate entity"], "should detect @Entity")
}

// ---------------------------------------------------------------------------
// TestJavaExtractor_MultiModule
// ---------------------------------------------------------------------------

func TestJavaExtractor_MultiModule(t *testing.T) {
	dir := t.TempDir()

	// Root pom.xml with <modules> section.
	writeFile(t, dir, "pom.xml", `<?xml version="1.0"?>
<project>
  <modelVersion>4.0.0</modelVersion>
  <groupId>com.example</groupId>
  <artifactId>parent</artifactId>
  <packaging>pom</packaging>
  <modules>
    <module>core</module>
    <module>web</module>
    <module>data</module>
  </modules>
</project>
`)

	// Sub-module pom with its own dependencies.
	writeFile(t, dir, "core/pom.xml", `<?xml version="1.0"?>
<project>
  <parent>
    <groupId>com.example</groupId>
    <artifactId>parent</artifactId>
  </parent>
  <artifactId>core</artifactId>
  <dependencies>
    <dependency>
      <groupId>com.google.guava</groupId>
      <artifactId>guava</artifactId>
    </dependency>
  </dependencies>
</project>
`)

	// Java source in a sub-module.
	writeFile(t, dir, "core/src/main/java/com/example/core/CoreService.java", `
package com.example.core;

import java.util.Optional;
`)

	writeFile(t, dir, "web/src/main/java/com/example/web/WebApp.java", `
package com.example.web;

import java.net.URI;
`)

	ext := &extract.JavaExtractor{}
	result, err := ext.Extract(dir)
	require.NoError(t, err)

	// Multi-module detection.
	assert.Equal(t, "true", result.Metadata["multi_module"])
	require.Len(t, result.Modules, 3)
	assert.Contains(t, result.Modules, "core")
	assert.Contains(t, result.Modules, "web")
	assert.Contains(t, result.Modules, "data")

	// Build system should aggregate dependencies from sub-module pom.
	require.NotNil(t, result.BuildSystem)
	assert.Equal(t, "maven", result.BuildSystem.Type)

	foundGuava := false
	for _, dep := range result.BuildSystem.Dependencies {
		if dep.Artifact == "guava" && dep.Group == "com.google.guava" {
			foundGuava = true
		}
	}
	assert.True(t, foundGuava, "should find guava dependency from sub-module pom")
}

// ---------------------------------------------------------------------------
// TestJavaExtractor_NoJavaFiles
// ---------------------------------------------------------------------------

func TestJavaExtractor_NoJavaFiles(t *testing.T) {
	dir := t.TempDir()

	// Create a non-Java file so the directory is not empty.
	writeFile(t, dir, "README.md", "# Hello")

	ext := &extract.JavaExtractor{}
	_, err := ext.Extract(dir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no Java or Kotlin source files")
}

// ---------------------------------------------------------------------------
// TestJavaExtractor_DuplicateImports
// ---------------------------------------------------------------------------

func TestJavaExtractor_DuplicateImports(t *testing.T) {
	dir := t.TempDir()

	// Two files in the same package importing the same thing.
	writeFile(t, dir, "src/main/java/pkg/A.java", `
package pkg;
import java.util.List;
import java.util.Map;
`)
	writeFile(t, dir, "src/main/java/pkg/B.java", `
package pkg;
import java.util.List;
import java.util.Set;
`)

	ext := &extract.JavaExtractor{}
	result, err := ext.Extract(dir)
	require.NoError(t, err)

	imports := result.ImportGraph.PackageImports["src/main/java/pkg"]
	require.NotNil(t, imports)

	// Count occurrences of List -- should be exactly 1 after dedup.
	count := 0
	for _, imp := range imports {
		if imp == "java.util.List" {
			count++
		}
	}
	assert.Equal(t, 1, count, "duplicate imports should be removed")
	assert.Contains(t, imports, "java.util.Map")
	assert.Contains(t, imports, "java.util.Set")
}
