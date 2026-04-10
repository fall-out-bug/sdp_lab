package extract

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

// createTempDir creates a temporary directory and returns its path along with
// a cleanup function.
func createTempDir(t *testing.T) (string, func()) {
	t.Helper()
	dir, err := os.MkdirTemp("", "java-extract-test-")
	if err != nil {
		t.Fatal(err)
	}
	return dir, func() { os.RemoveAll(dir) }
}

// writeFile creates a file at the given path with content, creating any
// intermediate directories as needed.
func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// ---------------------------------------------------------------------------
// Import scanning tests
// ---------------------------------------------------------------------------

func TestScanJavaImports(t *testing.T) {
	dir, cleanup := createTempDir(t)
	defer cleanup()

	javaFile := filepath.Join(dir, "Example.java")
	content := `package com.example;

import java.util.List;
import java.util.ArrayList;
import static org.junit.Assert.assertEquals;
import com.example.service.UserService;
import com.example.model.*;
`
	writeFile(t, javaFile, content)

	imports, err := scanJavaImports(javaFile)
	if err != nil {
		t.Fatalf("scanJavaImports returned error: %v", err)
	}

	expected := []string{
		"java.util.List",
		"java.util.ArrayList",
		"org.junit.Assert.assertEquals",
		"com.example.service.UserService",
		"com.example.model.*",
	}

	if len(imports) != len(expected) {
		t.Fatalf("expected %d imports, got %d: %v", len(expected), len(imports), imports)
	}

	for i, imp := range imports {
		if imp != expected[i] {
			t.Errorf("import[%d] = %q, want %q", i, imp, expected[i])
		}
	}
}

func TestScanKotlinImports(t *testing.T) {
	dir, cleanup := createTempDir(t)
	defer cleanup()

	ktFile := filepath.Join(dir, "Example.kt")
	content := `package com.example

import java.util.List
import kotlin.collections.ArrayList
import com.example.service.UserService
`
	writeFile(t, ktFile, content)

	imports, err := scanKotlinImports(ktFile)
	if err != nil {
		t.Fatalf("scanKotlinImports returned error: %v", err)
	}

	expected := []string{
		"java.util.List",
		"kotlin.collections.ArrayList",
		"com.example.service.UserService",
	}

	if len(imports) != len(expected) {
		t.Fatalf("expected %d imports, got %d: %v", len(expected), len(imports), imports)
	}

	for i, imp := range imports {
		if imp != expected[i] {
			t.Errorf("import[%d] = %q, want %q", i, imp, expected[i])
		}
	}
}

// ---------------------------------------------------------------------------
// pom.xml parsing tests
// ---------------------------------------------------------------------------

func TestParsePomXML(t *testing.T) {
	dir, cleanup := createTempDir(t)
	defer cleanup()

	pomFile := filepath.Join(dir, "pom.xml")
	content := `<?xml version="1.0" encoding="UTF-8"?>
<project>
  <dependencies>
    <dependency>
      <groupId>org.springframework.boot</groupId>
      <artifactId>spring-boot-starter-web</artifactId>
      <scope>compile</scope>
    </dependency>
    <dependency>
      <groupId>org.projectlombok</groupId>
      <artifactId>lombok</artifactId>
      <scope>provided</scope>
    </dependency>
    <dependency>
      <groupId>junit</groupId>
      <artifactId>junit</artifactId>
      <scope>test</scope>
    </dependency>
  </dependencies>
</project>
`
	writeFile(t, pomFile, content)

	bs, err := parsePomXML(pomFile)
	if err != nil {
		t.Fatalf("parsePomXML returned error: %v", err)
	}

	if bs.Type != "maven" {
		t.Errorf("build system type = %q, want %q", bs.Type, "maven")
	}

	if len(bs.Dependencies) != 3 {
		t.Fatalf("expected 3 dependencies, got %d: %+v", len(bs.Dependencies), bs.Dependencies)
	}

	// Check first dependency.
	if bs.Dependencies[0].Group != "org.springframework.boot" {
		t.Errorf("dep[0].Group = %q, want %q", bs.Dependencies[0].Group, "org.springframework.boot")
	}
	if bs.Dependencies[0].Artifact != "spring-boot-starter-web" {
		t.Errorf("dep[0].Artifact = %q, want %q", bs.Dependencies[0].Artifact, "spring-boot-starter-web")
	}
	if bs.Dependencies[0].Scope != "compile" {
		t.Errorf("dep[0].Scope = %q, want %q", bs.Dependencies[0].Scope, "compile")
	}

	// Check test scope.
	if bs.Dependencies[2].Scope != "test" {
		t.Errorf("dep[2].Scope = %q, want %q", bs.Dependencies[2].Scope, "test")
	}
}

func TestParsePomXMLEmpty(t *testing.T) {
	dir, cleanup := createTempDir(t)
	defer cleanup()

	pomFile := filepath.Join(dir, "pom.xml")
	writeFile(t, pomFile, `<project></project>`)

	bs, err := parsePomXML(pomFile)
	if err != nil {
		t.Fatalf("parsePomXML returned error: %v", err)
	}

	if bs.Type != "maven" {
		t.Errorf("type = %q, want %q", bs.Type, "maven")
	}
	if len(bs.Dependencies) != 0 {
		t.Errorf("expected 0 dependencies, got %d", len(bs.Dependencies))
	}
}

// ---------------------------------------------------------------------------
// Maven module detection tests
// ---------------------------------------------------------------------------

func TestParseModules(t *testing.T) {
	dir, cleanup := createTempDir(t)
	defer cleanup()

	pomFile := filepath.Join(dir, "pom.xml")
	content := `<project>
  <modules>
    <module>service-a</module>
    <module>service-b</module>
    <module>common</module>
  </modules>
</project>
`
	writeFile(t, pomFile, content)

	modules := parseModules(pomFile)
	if len(modules) != 3 {
		t.Fatalf("expected 3 modules, got %d: %v", len(modules), modules)
	}

	expected := []string{"service-a", "service-b", "common"}
	for i, m := range modules {
		if m != expected[i] {
			t.Errorf("module[%d] = %q, want %q", i, m, expected[i])
		}
	}
}

func TestParseModulesNone(t *testing.T) {
	dir, cleanup := createTempDir(t)
	defer cleanup()

	pomFile := filepath.Join(dir, "pom.xml")
	writeFile(t, pomFile, `<project></project>`)

	modules := parseModules(pomFile)
	if len(modules) != 0 {
		t.Errorf("expected 0 modules, got %d", len(modules))
	}
}

// ---------------------------------------------------------------------------
// build.gradle parsing tests
// ---------------------------------------------------------------------------

func TestParseBuildGradle(t *testing.T) {
	dir, cleanup := createTempDir(t)
	defer cleanup()

	gradleFile := filepath.Join(dir, "build.gradle")
	content := `
plugins {
    id 'java'
    id 'org.springframework.boot'
}

dependencies {
    implementation 'org.springframework.boot:spring-boot-starter-web'
    implementation 'com.google.guava:guava:31.1-jre'
    testImplementation 'org.junit.jupiter:junit-jupiter:5.8.2'
    compileOnly 'org.projectlombok:lombok'
}
`
	writeFile(t, gradleFile, content)

	bs, err := parseBuildGradle(gradleFile)
	if err != nil {
		t.Fatalf("parseBuildGradle returned error: %v", err)
	}

	if bs.Type != "gradle" {
		t.Errorf("type = %q, want %q", bs.Type, "gradle")
	}

	if len(bs.Dependencies) != 4 {
		t.Fatalf("expected 4 dependencies, got %d: %+v", len(bs.Dependencies), bs.Dependencies)
	}

	// Check first dependency.
	if bs.Dependencies[0].Group != "org.springframework.boot" {
		t.Errorf("dep[0].Group = %q, want %q", bs.Dependencies[0].Group, "org.springframework.boot")
	}
	if bs.Dependencies[0].Artifact != "spring-boot-starter-web" {
		t.Errorf("dep[0].Artifact = %q, want %q", bs.Dependencies[0].Artifact, "spring-boot-starter-web")
	}
}

func TestParseBuildGradleKts(t *testing.T) {
	dir, cleanup := createTempDir(t)
	defer cleanup()

	gradleFile := filepath.Join(dir, "build.gradle.kts")
	content := `
dependencies {
    implementation("org.springframework.boot:spring-boot-starter-data-jpa")
    implementation("org.jetbrains.kotlin:kotlin-stdlib")
}
`
	writeFile(t, gradleFile, content)

	bs, err := parseBuildGradle(gradleFile)
	if err != nil {
		t.Fatalf("parseBuildGradle returned error: %v", err)
	}

	if len(bs.Dependencies) != 2 {
		t.Fatalf("expected 2 dependencies, got %d: %+v", len(bs.Dependencies), bs.Dependencies)
	}
}

// ---------------------------------------------------------------------------
// settings.gradle parsing tests
// ---------------------------------------------------------------------------

func TestParseSettingsGradle(t *testing.T) {
	dir, cleanup := createTempDir(t)
	defer cleanup()

	settingsFile := filepath.Join(dir, "settings.gradle")
	writeFile(t, settingsFile, `
rootProject.name = 'my-project'
include 'service-a'
include 'service-b'
include 'common'
`)

	subprojects := parseSettingsGradle(dir)
	if len(subprojects) != 3 {
		t.Fatalf("expected 3 subprojects, got %d: %v", len(subprojects), subprojects)
	}

	expected := []string{"service-a", "service-b", "common"}
	for i, s := range subprojects {
		if s != expected[i] {
			t.Errorf("subproject[%d] = %q, want %q", i, s, expected[i])
		}
	}
}

func TestParseSettingsGradleKts(t *testing.T) {
	dir, cleanup := createTempDir(t)
	defer cleanup()

	settingsFile := filepath.Join(dir, "settings.gradle.kts")
	writeFile(t, settingsFile, `
rootProject.name = "my-project"
include("service-x")
include("service-y")
`)

	subprojects := parseSettingsGradle(dir)
	if len(subprojects) != 2 {
		t.Fatalf("expected 2 subprojects, got %d: %v", len(subprojects), subprojects)
	}
}

// ---------------------------------------------------------------------------
// Spring annotation and endpoint detection tests
// ---------------------------------------------------------------------------

func TestSpringAnnotationDetection(t *testing.T) {
	dir, cleanup := createTempDir(t)
	defer cleanup()

	javaFile := filepath.Join(dir, "UserController.java")
	content := `package com.example.controller;

import org.springframework.web.bind.annotation.RestController;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.PostMapping;

@RestController
public class UserController {

    @GetMapping("/users")
    public List<User> getUsers() {
        return userService.findAll();
    }

    @PostMapping("/users")
    public User createUser(@RequestBody User user) {
        return userService.save(user);
    }
}
`
	writeFile(t, javaFile, content)

	imports, annotations, pkgDecl, err := scanJavaFile(javaFile, "UserController.java")
	if err != nil {
		t.Fatalf("scanJavaFile returned error: %v", err)
	}

	// Check package declaration.
	if pkgDecl != "com.example.controller" {
		t.Errorf("package = %q, want %q", pkgDecl, "com.example.controller")
	}

	// Check imports.
	if len(imports) != 3 {
		t.Errorf("expected 3 imports, got %d: %v", len(imports), imports)
	}

	// Check annotations.
	if len(annotations) < 1 {
		t.Fatal("expected at least 1 annotation")
	}

	foundRestController := false
	for _, a := range annotations {
		if a.Annotation == "@RestController" {
			foundRestController = true
		}
	}
	if !foundRestController {
		t.Error("expected @RestController annotation to be detected")
	}
}

func TestLombokAnnotationDetection(t *testing.T) {
	dir, cleanup := createTempDir(t)
	defer cleanup()

	javaFile := filepath.Join(dir, "User.java")
	content := `package com.example.model;

import lombok.Data;
import lombok.Builder;
import lombok.Getter;

@Data
@Builder
public class User {
    @Getter
    private String name;
}
`
	writeFile(t, javaFile, content)

	_, annotations, _, err := scanJavaFile(javaFile, "User.java")
	if err != nil {
		t.Fatalf("scanJavaFile returned error: %v", err)
	}

	expectedAnnotations := map[string]bool{
		"@Data":   false,
		"@Builder": false,
		"@Getter": false,
	}

	for _, a := range annotations {
		if _, ok := expectedAnnotations[a.Annotation]; ok {
			expectedAnnotations[a.Annotation] = true
		}
	}

	for annot, found := range expectedAnnotations {
		if !found {
			t.Errorf("expected %s annotation not found", annot)
		}
	}
}

func TestSpringEndpointExtraction(t *testing.T) {
	dir, cleanup := createTempDir(t)
	defer cleanup()

	javaFile := filepath.Join(dir, "OrderController.java")
	content := `package com.example.controller;

import org.springframework.web.bind.annotation.*;

@RestController
@RequestMapping("/api/v1")
public class OrderController {

    @GetMapping("/orders")
    public List<Order> getOrders() {
        return orderService.findAll();
    }

    @PostMapping("/orders")
    public Order createOrder(@RequestBody Order order) {
        return orderService.save(order);
    }

    @PutMapping("/orders/{id}")
    public Order updateOrder(@PathVariable Long id, @RequestBody Order order) {
        return orderService.update(id, order);
    }

    @DeleteMapping("/orders/{id}")
    public void deleteOrder(@PathVariable Long id) {
        orderService.delete(id);
    }

    @PatchMapping("/orders/{id}/status")
    public Order patchStatus(@PathVariable Long id) {
        return orderService.patchStatus(id);
    }
}
`
	writeFile(t, javaFile, content)

	// First extract class-level mapping.
	prefix, prefixLine := extractClassLevelMapping(javaFile)
	if prefix != "/api/v1" {
		t.Errorf("class-level mapping = %q, want %q", prefix, "/api/v1")
	}

	// Extract endpoints.
	endpoints := extractSpringEndpoints(javaFile, "OrderController.java", prefix, prefixLine)
	if len(endpoints) != 5 {
		t.Fatalf("expected 5 endpoints, got %d: %+v", len(endpoints), endpoints)
	}

	expectedEndpoints := []SpringEndpoint{
		{HTTPMethod: "GET", Path: "/api/v1/orders"},
		{HTTPMethod: "POST", Path: "/api/v1/orders"},
		{HTTPMethod: "PUT", Path: "/api/v1/orders/{id}"},
		{HTTPMethod: "DELETE", Path: "/api/v1/orders/{id}"},
		{HTTPMethod: "PATCH", Path: "/api/v1/orders/{id}/status"},
	}

	for i, ep := range expectedEndpoints {
		if i >= len(endpoints) {
			t.Errorf("missing endpoint %s %s", ep.HTTPMethod, ep.Path)
			continue
		}
		if endpoints[i].HTTPMethod != ep.HTTPMethod {
			t.Errorf("endpoint[%d].HTTPMethod = %q, want %q", i, endpoints[i].HTTPMethod, ep.HTTPMethod)
		}
		if endpoints[i].Path != ep.Path {
			t.Errorf("endpoint[%d].Path = %q, want %q", i, endpoints[i].Path, ep.Path)
		}
	}
}

func TestSpringEndpointsNoClassPrefix(t *testing.T) {
	dir, cleanup := createTempDir(t)
	defer cleanup()

	javaFile := filepath.Join(dir, "SimpleController.java")
	content := `package com.example;

@RestController
public class SimpleController {

    @GetMapping("/health")
    public String health() {
        return "OK";
    }
}
`
	writeFile(t, javaFile, content)

	prefix, prefixLine := extractClassLevelMapping(javaFile)
	if prefix != "" {
		t.Errorf("expected empty class-level mapping, got %q", prefix)
	}

	endpoints := extractSpringEndpoints(javaFile, "SimpleController.java", "", prefixLine)
	if len(endpoints) != 1 {
		t.Fatalf("expected 1 endpoint, got %d", len(endpoints))
	}
	if endpoints[0].HTTPMethod != "GET" {
		t.Errorf("method = %q, want %q", endpoints[0].HTTPMethod, "GET")
	}
	if endpoints[0].Path != "/health" {
		t.Errorf("path = %q, want %q", endpoints[0].Path, "/health")
	}
}

// ---------------------------------------------------------------------------
// Package structure detection tests
// ---------------------------------------------------------------------------

func TestDetectPackageStructure(t *testing.T) {
	dir, cleanup := createTempDir(t)
	defer cleanup()

	// Create standard Maven directory structure.
	writeFile(t, filepath.Join(dir, "src/main/java/com/example/App.java"), "package com.example;")
	writeFile(t, filepath.Join(dir, "src/test/java/com/example/AppTest.java"), "package com.example;")
	writeFile(t, filepath.Join(dir, "src/main/kotlin/com/example/Utils.kt"), "package com.example")

	ps := detectPackageStructure(dir)

	if len(ps.SourceDirs) != 2 {
		t.Errorf("expected 2 source dirs (java + kotlin), got %d: %v", len(ps.SourceDirs), ps.SourceDirs)
	}
	if len(ps.TestDirs) != 1 {
		t.Errorf("expected 1 test dir, got %d: %v", len(ps.TestDirs), ps.TestDirs)
	}
	if !ps.HasKotlin {
		t.Error("expected HasKotlin = true")
	}
}

// ---------------------------------------------------------------------------
// Kotlin-specific pattern detection tests
// ---------------------------------------------------------------------------

func TestKotlinPatternDetection(t *testing.T) {
	dir, cleanup := createTempDir(t)
	defer cleanup()

	ktFile := filepath.Join(dir, "Models.kt")
	content := `package com.example.models

data class User(val name: String, val email: String)

class UserService {
    companion object {
        fun create(): UserService = UserService()
    }

    fun String.isEmail(): Boolean = this.contains("@")
}
`
	writeFile(t, ktFile, content)

	result := &JavaExtractionResult{
		Metadata: make(map[string]string),
	}
	detectKotlinPatterns(ktFile, result)

	if result.Metadata["kotlin_data_class"] != "true" {
		t.Error("expected kotlin_data_class to be detected")
	}
	if result.Metadata["kotlin_companion_object"] != "true" {
		t.Error("expected kotlin_companion_object to be detected")
	}
	if result.Metadata["kotlin_extension_function"] != "true" {
		t.Error("expected kotlin_extension_function to be detected")
	}
}

// ---------------------------------------------------------------------------
// Full integration test
// ---------------------------------------------------------------------------

func TestJavaExtractorExtract(t *testing.T) {
	dir, cleanup := createTempDir(t)
	defer cleanup()

	// Create a realistic Maven project structure.
	writeFile(t, filepath.Join(dir, "pom.xml"), `<?xml version="1.0"?>
<project>
  <modules>
    <module>user-service</module>
    <module>order-service</module>
  </modules>
  <dependencies>
    <dependency>
      <groupId>org.springframework.boot</groupId>
      <artifactId>spring-boot-starter-web</artifactId>
    </dependency>
  </dependencies>
</project>
`)

	writeFile(t, filepath.Join(dir, "src/main/java/com/example/Application.java"), `
package com.example;
import org.springframework.boot.SpringApplication;
import org.springframework.boot.autoconfigure.SpringBootApplication;
@SpringBootApplication
public class Application {
    public static void main(String[] args) {
        SpringApplication.run(Application.class, args);
    }
}
`)

	writeFile(t, filepath.Join(dir, "src/main/java/com/example/controller/UserController.java"), `
package com.example.controller;
import org.springframework.web.bind.annotation.RestController;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.RequestMapping;
@RestController
@RequestMapping("/api/users")
public class UserController {
    @GetMapping("/")
    public String list() { return "users"; }
}
`)

	writeFile(t, filepath.Join(dir, "src/main/java/com/example/model/User.java"), `
package com.example.model;
import lombok.Data;
import lombok.Builder;
@Data
@Builder
public class User {
    private String name;
}
`)

	writeFile(t, filepath.Join(dir, "src/main/java/com/example/service/UserService.java"), `
package com.example.service;
import org.springframework.stereotype.Service;
@Service
public class UserService {
}
`)

	writeFile(t, filepath.Join(dir, "src/main/java/com/example/repository/UserRepository.java"), `
package com.example.repository;
import org.springframework.stereotype.Repository;
@Repository
public interface UserRepository {
}
`)

	writeFile(t, filepath.Join(dir, "src/main/kotlin/com/example/utils/Extensions.kt"), `
package com.example.utils
data class Result(val ok: Boolean)
`)

	e := &JavaExtractor{}
	result, err := e.Extract(dir)
	if err != nil {
		t.Fatalf("Extract returned error: %v", err)
	}

	// Check basic fields.
	if result.ExtractionMethod != "regex" {
		t.Errorf("extraction method = %q, want %q", result.ExtractionMethod, "regex")
	}
	if result.AccuracyEstimate != 0.70 {
		t.Errorf("accuracy = %f, want %f", result.AccuracyEstimate, 0.70)
	}

	// Check package structure.
	if result.PackageStructure == nil {
		t.Fatal("expected non-nil PackageStructure")
	}
	if !result.PackageStructure.HasKotlin {
		t.Error("expected HasKotlin = true")
	}
	if !result.PackageStructure.MultiModule {
		t.Error("expected MultiModule = true")
	}
	if result.PackageStructure.BuildTool != "maven" {
		t.Errorf("build tool = %q, want %q", result.PackageStructure.BuildTool, "maven")
	}

	// Check modules.
	if len(result.Modules) != 2 {
		t.Errorf("expected 2 modules, got %d: %v", len(result.Modules), result.Modules)
	}

	// Check build system.
	if result.BuildSystem == nil {
		t.Fatal("expected non-nil BuildSystem")
	}
	if result.BuildSystem.Type != "maven" {
		t.Errorf("build system type = %q, want %q", result.BuildSystem.Type, "maven")
	}

	// Check frameworks (should detect Spring Boot and Lombok).
	if len(result.Frameworks) < 2 {
		t.Errorf("expected at least 2 frameworks (Spring + Lombok), got %d: %+v",
			len(result.Frameworks), result.Frameworks)
	}

	fwNames := make(map[string]bool)
	for _, fw := range result.Frameworks {
		fwNames[fw.Name] = true
	}
	if !fwNames["Spring Boot"] {
		t.Error("expected Spring Boot framework detection")
	}
	if !fwNames["Lombok"] {
		t.Error("expected Lombok framework detection")
	}

	// Check endpoints.
	if len(result.SpringEndpoints) < 1 {
		t.Error("expected at least 1 Spring endpoint")
	} else {
		foundUsersEndpoint := false
		for _, ep := range result.SpringEndpoints {
			// joinPaths("/api/users", "/") = "/api/users"
			if (ep.Path == "/api/users" || ep.Path == "/api/users/") && ep.HTTPMethod == "GET" {
				foundUsersEndpoint = true
			}
		}
		if !foundUsersEndpoint {
			t.Errorf("expected GET /api/users endpoint, got: %+v", result.SpringEndpoints)
		}
		// Verify no duplicate from class-level @RequestMapping.
		for _, ep := range result.SpringEndpoints {
			if ep.Path == "/api/users/api/users" {
				t.Errorf("found duplicate class-level mapping as endpoint: %+v", ep)
			}
		}
	}

	// Check root packages.
	if len(result.PackageStructure.RootPackages) == 0 {
		t.Error("expected root packages to be detected")
	}
	hasComExample := false
	for _, pkg := range result.PackageStructure.RootPackages {
		if pkg == "com.example" {
			hasComExample = true
		}
	}
	if !hasComExample {
		t.Errorf("expected 'com.example' in root packages, got: %v", result.PackageStructure.RootPackages)
	}

	// Check import graph.
	if len(result.ImportGraph.PackageImports) == 0 {
		t.Error("expected import graph to contain packages")
	}

	// Check annotations.
	if len(result.Annotations) == 0 {
		t.Error("expected annotations to be detected")
	}

	// Check metadata.
	if result.Metadata["multi_module"] != "true" {
		t.Error("expected multi_module metadata")
	}
}

func TestJavaExtractorGradleProject(t *testing.T) {
	dir, cleanup := createTempDir(t)
	defer cleanup()

	writeFile(t, filepath.Join(dir, "build.gradle"), `
dependencies {
    implementation 'org.springframework.boot:spring-boot-starter-web'
    implementation 'org.projectlombok:lombok'
}
`)

	writeFile(t, filepath.Join(dir, "settings.gradle"), `
rootProject.name = 'my-app'
include 'module-a'
include 'module-b'
`)

	writeFile(t, filepath.Join(dir, "src/main/java/com/example/App.java"), `
package com.example;
public class App {}
`)

	e := &JavaExtractor{}
	result, err := e.Extract(dir)
	if err != nil {
		t.Fatalf("Extract returned error: %v", err)
	}

	if result.BuildSystem == nil || result.BuildSystem.Type != "gradle" {
		t.Errorf("expected gradle build system, got: %+v", result.BuildSystem)
	}

	if !result.PackageStructure.MultiModule {
		t.Error("expected MultiModule = true from settings.gradle")
	}

	if len(result.Modules) != 2 {
		t.Errorf("expected 2 modules from settings.gradle, got %d: %v", len(result.Modules), result.Modules)
	}
}

func TestJavaExtractorNoSourceFiles(t *testing.T) {
	dir, cleanup := createTempDir(t)
	defer cleanup()

	// Only pom.xml, no source files.
	writeFile(t, filepath.Join(dir, "pom.xml"), `<project></project>`)

	e := &JavaExtractor{}
	_, err := e.Extract(dir)
	if err == nil {
		t.Error("expected error when no source files found")
	}
}

// ---------------------------------------------------------------------------
// Helper function tests
// ---------------------------------------------------------------------------

func TestJoinPaths(t *testing.T) {
	tests := []struct {
		prefix, suffix, want string
	}{
		{"", "/users", "/users"},
		{"/api", "/users", "/api/users"},
		{"/api/", "/users", "/api/users"},
		{"/api", "/", "/api"},
		{"/api", "", "/api"},
	}
	for _, tt := range tests {
		got := joinPaths(tt.prefix, tt.suffix)
		if got != tt.want {
			t.Errorf("joinPaths(%q, %q) = %q, want %q", tt.prefix, tt.suffix, got, tt.want)
		}
	}
}

func TestDedup(t *testing.T) {
	input := []string{"a", "b", "a", "c", "b"}
	result := dedup(input)

	if len(result) != 3 {
		t.Fatalf("expected 3 unique items, got %d: %v", len(result), result)
	}

	// Check all expected values present.
	seen := make(map[string]bool)
	for _, s := range result {
		seen[s] = true
	}
	for _, expected := range []string{"a", "b", "c"} {
		if !seen[expected] {
			t.Errorf("expected %q in deduped result", expected)
		}
	}
}

func TestRecordRootPackage(t *testing.T) {
	ps := &PackageStructure{}

	recordRootPackage(ps, "com.example.service")
	recordRootPackage(ps, "com.example.model")
	recordRootPackage(ps, "org.other.utils")

	if len(ps.RootPackages) != 2 {
		t.Fatalf("expected 2 root packages, got %d: %v", len(ps.RootPackages), ps.RootPackages)
	}

	sort.Strings(ps.RootPackages)
	if ps.RootPackages[0] != "com.example" || ps.RootPackages[1] != "org.other" {
		t.Errorf("unexpected root packages: %v", ps.RootPackages)
	}
}

func TestIsJavaFile(t *testing.T) {
	if !isJavaFile("src/main/java/App.java") {
		t.Error("expected .java file to be recognized")
	}
	if isJavaFile("src/main/java/App.kt") {
		t.Error("expected .kt file to not be recognized as Java")
	}
}

func TestIsKotlinFile(t *testing.T) {
	if !isKotlinFile("src/main/kotlin/App.kt") {
		t.Error("expected .kt file to be recognized")
	}
	if isKotlinFile("src/main/java/App.java") {
		t.Error("expected .java file to not be recognized as Kotlin")
	}
	if isKotlinFile("build.gradle.kts") {
		t.Error(".kts build files should not be treated as Kotlin source files")
	}
}

// ---------------------------------------------------------------------------
// Spring evidence detection tests
// ---------------------------------------------------------------------------

func TestDetectSpringEvidence(t *testing.T) {
	imports := map[string]bool{
		"org.springframework.web.bind.annotation.RestController": true,
		"org.springframework.stereotype.Service":                true,
	}

	annotations := map[string]bool{
		"@RestController": true,
		"@Service":        true,
	}

	evidence := detectSpringEvidence(imports, annotations)
	if len(evidence) == 0 {
		t.Error("expected Spring evidence to be detected")
	}

	// Should include both import-based and annotation-based evidence.
	foundImportEvidence := false
	foundAnnotEvidence := false
	for _, e := range evidence {
		if strings.Contains(e, "annotation in source") {
			foundAnnotEvidence = true
		} else {
			foundImportEvidence = true
		}
	}

	if !foundImportEvidence {
		t.Error("expected import-based evidence")
	}
	if !foundAnnotEvidence {
		t.Error("expected annotation-based evidence")
	}
}

func TestDetectHibernateEvidence(t *testing.T) {
	imports := map[string]bool{
		"org.hibernate.Session":           true,
		"javax.persistence.EntityManager": true,
	}

	evidence := detectHibernateEvidence(imports)
	if len(evidence) != 2 {
		t.Errorf("expected 2 Hibernate evidence strings, got %d: %v", len(evidence), evidence)
	}
}

// ---------------------------------------------------------------------------
// Build system detection via full walk tests
// ---------------------------------------------------------------------------

func TestBuildSystemMavenDepsParsed(t *testing.T) {
	dir, cleanup := createTempDir(t)
	defer cleanup()

	writeFile(t, filepath.Join(dir, "pom.xml"), `<project>
  <dependencies>
    <dependency>
      <groupId>com.google.guava</groupId>
      <artifactId>guava</artifactId>
    </dependency>
  </dependencies>
</project>
`)
	writeFile(t, filepath.Join(dir, "src/main/java/Foo.java"), "class Foo {}")

	e := &JavaExtractor{}
	result, err := e.Extract(dir)
	if err != nil {
		t.Fatalf("Extract error: %v", err)
	}

	if result.BuildSystem == nil {
		t.Fatal("expected BuildSystem to be set")
	}
	found := false
	for _, dep := range result.BuildSystem.Dependencies {
		if dep.Artifact == "guava" {
			found = true
		}
	}
	if !found {
		t.Error("expected guava dependency to be parsed from pom.xml")
	}
}

func TestSkipTargetDirectory(t *testing.T) {
	dir, cleanup := createTempDir(t)
	defer cleanup()

	// Source file.
	writeFile(t, filepath.Join(dir, "src/main/java/App.java"), "class App {}")

	// File inside target/ should be ignored.
	writeFile(t, filepath.Join(dir, "target/classes/Generated.java"), `
package com.generated;
@RestController
public class Generated {}
`)

	e := &JavaExtractor{}
	result, err := e.Extract(dir)
	if err != nil {
		t.Fatalf("Extract error: %v", err)
	}

	// The @RestController inside target/ should not be found.
	for _, a := range result.Annotations {
		if a.Annotation == "@RestController" {
			t.Error("expected @RestController in target/ to be skipped")
		}
	}
}
