package java

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Test helpers

func createTempDir(t *testing.T) (string, func()) {
	t.Helper()
	dir, err := os.MkdirTemp("", "java-extract-test-")
	if err != nil {
		t.Fatal(err)
	}
	return dir, func() { os.RemoveAll(dir) }
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestExtractSpringEndpoints(t *testing.T) {
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

	prefix, prefixLine := extractClassLevelMapping(javaFile)
	if prefix != "/api/v1" {
		t.Errorf("class-level mapping = %q, want %q", prefix, "/api/v1")
	}

	endpoints := extractSpringEndpoints(javaFile, "OrderController.java", prefix, prefixLine)
	if len(endpoints) != 5 {
		t.Fatalf("expected 5 endpoints, got %d: %+v", len(endpoints), endpoints)
	}

	expectedEndpoints := []JavaSpringEndpoint{
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

func TestExtractSpringEndpointsNoClassPrefix(t *testing.T) {
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

func TestExtractSpringEndpointsRequestMappingMethod(t *testing.T) {
	dir, cleanup := createTempDir(t)
	defer cleanup()

	javaFile := filepath.Join(dir, "MixedController.java")
	content := `package com.example;

import org.springframework.web.bind.annotation.*;

@RestController
@RequestMapping("/api")
public class MixedController {

    @RequestMapping(value = "/resource", method = RequestMethod.POST)
    public String create() {
        return "created";
    }

    @RequestMapping(value = "/resource", method = RequestMethod.GET)
    public String get() {
        return "ok";
    }
}
`
	writeFile(t, javaFile, content)

	prefix, prefixLine := extractClassLevelMapping(javaFile)
	if prefix != "/api" {
		t.Errorf("class-level mapping = %q, want %q", prefix, "/api")
	}

	endpoints := extractSpringEndpoints(javaFile, "MixedController.java", prefix, prefixLine)
	if len(endpoints) != 2 {
		t.Fatalf("expected 2 endpoints, got %d: %+v", len(endpoints), endpoints)
	}

	// Check POST endpoint
	if endpoints[0].HTTPMethod != "POST" {
		t.Errorf("endpoint[0].HTTPMethod = %q, want %q", endpoints[0].HTTPMethod, "POST")
	}
	if endpoints[0].Path != "/api/resource" {
		t.Errorf("endpoint[0].Path = %q, want %q", endpoints[0].Path, "/api/resource")
	}
}

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

	// Should include both import-based and annotation-based evidence
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

func TestExtractClassLevelMapping(t *testing.T) {
	dir, cleanup := createTempDir(t)
	defer cleanup()

	javaFile := filepath.Join(dir, "BaseController.java")
	content := `package com.example;

import org.springframework.web.bind.annotation.RequestMapping;

@RequestMapping("/api/v1")
public class BaseController {
}
`
	writeFile(t, javaFile, content)

	prefix, line := extractClassLevelMapping(javaFile)
	if prefix != "/api/v1" {
		t.Errorf("prefix = %q, want %q", prefix, "/api/v1")
	}
	if line == 0 {
		t.Error("expected non-zero line number")
	}
}

func TestExtractClassLevelMappingNoMapping(t *testing.T) {
	dir, cleanup := createTempDir(t)
	defer cleanup()

	javaFile := filepath.Join(dir, "NoMappingController.java")
	content := `package com.example;

public class NoMappingController {
}
`
	writeFile(t, javaFile, content)

	prefix, line := extractClassLevelMapping(javaFile)
	if prefix != "" {
		t.Errorf("expected empty prefix, got %q", prefix)
	}
	if line != 0 {
		t.Errorf("expected zero line number, got %d", line)
	}
}

func TestJoinPaths(t *testing.T) {
	tests := []struct {
		prefix, suffix, want string
	}{
		{"", "/users", "/users"},
		{"/api", "/users", "/api/users"},
		{"/api/", "/users", "/api/users"},
		{"/api", "/", "/api"},
		{"/api", "", "/api"},
		{"/api/v1", "/users", "/api/v1/users"},
		{"/api/v1/", "/users/", "/api/v1/users"},
	}
	for _, tt := range tests {
		got := joinPaths(tt.prefix, tt.suffix)
		if got != tt.want {
			t.Errorf("joinPaths(%q, %q) = %q, want %q", tt.prefix, tt.suffix, got, tt.want)
		}
	}
}

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

	// Check package
	if pkgDecl != "com.example.controller" {
		t.Errorf("package = %q, want %q", pkgDecl, "com.example.controller")
	}

	// Check imports
	if len(imports) != 3 {
		t.Errorf("expected 3 imports, got %d: %v", len(imports), imports)
	}

	// Check annotations
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

func TestDetectFrameworks(t *testing.T) {
	ig := JavaImportGraph{
		PackageImports: map[string][]string{
			"src/main/java/com/example": {
				"org.springframework.web.bind.annotation.RestController",
				"org.springframework.stereotype.Service",
				"org.hibernate.Session",
			},
		},
	}

	annotations := map[string]bool{
		"@RestController": true,
		"@Service":        true,
	}

	lombokAnnotations := map[string]bool{
		"@Data": true,
	}

	frameworks := detectFrameworks(ig, annotations, lombokAnnotations)

	if len(frameworks) < 2 {
		t.Errorf("expected at least 2 frameworks (Spring + Hibernate), got %d: %+v", len(frameworks), frameworks)
	}

	// Check Spring Boot framework
	foundSpring := false
	for _, fw := range frameworks {
		if fw.Name == "Spring Boot" {
			foundSpring = true
			if fw.Confidence <= 0 {
				t.Errorf("Spring Boot confidence should be > 0, got %f", fw.Confidence)
			}
		}
	}
	if !foundSpring {
		t.Error("expected Spring Boot framework to be detected")
	}

	// Check Hibernate framework
	foundHibernate := false
	for _, fw := range frameworks {
		if fw.Name == "Hibernate" {
			foundHibernate = true
		}
	}
	if !foundHibernate {
		t.Error("expected Hibernate framework to be detected")
	}
}
