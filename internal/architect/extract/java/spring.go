package java

import (
	"bufio"
	"os"
	"regexp"
	"sort"
	"strings"
)

// Spring endpoint mapping patterns
var (
	requestMappingRe    = regexp.MustCompile(`@RequestMapping\s*\(\s*(?:value\s*=\s*)?["']([^"']+)["']`)
	getMappingRe        = regexp.MustCompile(`@GetMapping\s*\(\s*(?:value\s*=\s*)?["']([^"']+)["']`)
	postMappingRe       = regexp.MustCompile(`@PostMapping\s*\(\s*(?:value\s*=\s*)?["']([^"']+)["']`)
	putMappingRe        = regexp.MustCompile(`@PutMapping\s*\(\s*(?:value\s*=\s*)?["']([^"']+)["']`)
	deleteMappingRe     = regexp.MustCompile(`@DeleteMapping\s*\(\s*(?:value\s*=\s*)?["']([^"']+)["']`)
	patchMappingRe      = regexp.MustCompile(`@PatchMapping\s*\(\s*(?:value\s*=\s*)?["']([^"']+)["']`)
	classLevelMappingRe  = regexp.MustCompile(`@RequestMapping\s*\(\s*(?:value\s*=\s*)?["']([^"']+)["']`)
)

// extractSpringEndpoints scans a file for Spring endpoint annotations.
func extractSpringEndpoints(path, relPath, classPrefix string, classPrefixLine int) []JavaSpringEndpoint {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()

	var endpoints []JavaSpringEndpoint
	lineNum := 0

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "//") {
			continue
		}

		// Skip the class-level @RequestMapping line
		if lineNum == classPrefixLine {
			continue
		}

		if m := getMappingRe.FindStringSubmatch(line); m != nil {
			endpoints = append(endpoints, JavaSpringEndpoint{
				HTTPMethod: "GET",
				Path:       joinPaths(classPrefix, m[1]),
				File:       relPath,
				LineNumber: lineNum,
			})
			continue
		}
		if m := postMappingRe.FindStringSubmatch(line); m != nil {
			endpoints = append(endpoints, JavaSpringEndpoint{
				HTTPMethod: "POST",
				Path:       joinPaths(classPrefix, m[1]),
				File:       relPath,
				LineNumber: lineNum,
			})
			continue
		}
		if m := putMappingRe.FindStringSubmatch(line); m != nil {
			endpoints = append(endpoints, JavaSpringEndpoint{
				HTTPMethod: "PUT",
				Path:       joinPaths(classPrefix, m[1]),
				File:       relPath,
				LineNumber: lineNum,
			})
			continue
		}
		if m := deleteMappingRe.FindStringSubmatch(line); m != nil {
			endpoints = append(endpoints, JavaSpringEndpoint{
				HTTPMethod: "DELETE",
				Path:       joinPaths(classPrefix, m[1]),
				File:       relPath,
				LineNumber: lineNum,
			})
			continue
		}
		if m := patchMappingRe.FindStringSubmatch(line); m != nil {
			endpoints = append(endpoints, JavaSpringEndpoint{
				HTTPMethod: "PATCH",
				Path:       joinPaths(classPrefix, m[1]),
				File:       relPath,
				LineNumber: lineNum,
			})
			continue
		}

		// @RequestMapping at method level - default to GET unless method specified
		if m := requestMappingRe.FindStringSubmatch(line); m != nil {
			method := "GET" // default
			if strings.Contains(line, "method") {
				if strings.Contains(line, "POST") {
					method = "POST"
				} else if strings.Contains(line, "PUT") {
					method = "PUT"
				} else if strings.Contains(line, "DELETE") {
					method = "DELETE"
				} else if strings.Contains(line, "PATCH") {
					method = "PATCH"
				}
			}
			endpoints = append(endpoints, JavaSpringEndpoint{
				HTTPMethod: method,
				Path:       joinPaths(classPrefix, m[1]),
				File:       relPath,
				LineNumber: lineNum,
			})
		}
	}

	return endpoints
}

// extractClassLevelMapping returns the path and line number from a class-level
// @RequestMapping annotation. Returns ("", 0) if none found.
func extractClassLevelMapping(path string) (string, int) {
	f, err := os.Open(path)
	if err != nil {
		return "", 0
	}
	defer f.Close()

	lineNum := 0
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "//") {
			continue
		}
		// Look for @RequestMapping at class level (before any method definitions)
		if m := classLevelMappingRe.FindStringSubmatch(line); m != nil {
			return m[1], lineNum
		}
		// Once we hit a method definition, stop looking for class-level mapping
		if strings.Contains(line, "public ") && strings.Contains(line, "(") && !strings.Contains(line, "@") {
			break
		}
	}
	return "", 0
}

// detectFrameworks analyzes imports and annotations to detect frameworks.
func detectFrameworks(ig JavaImportGraph, annotations map[string]bool, lombokAnnotations map[string]bool) []JavaFramework {
	var frameworks []JavaFramework

	// Collect all imports into a single set
	allImports := make(map[string]bool)
	for _, imports := range ig.PackageImports {
		for _, imp := range imports {
			allImports[imp] = true
		}
	}

	// Spring Boot detection
	springEvidence := detectSpringEvidence(allImports, annotations)
	if len(springEvidence) > 0 {
		confidence := float64(len(springEvidence)) / float64(len(springAnnotations))
		if confidence > 1.0 {
			confidence = 1.0
		}
		if confidence < 0.5 {
			confidence = 0.5
		}
		frameworks = append(frameworks, JavaFramework{
			Name:       "Spring Boot",
			Confidence: confidence,
			Evidence:   springEvidence,
		})
	}

	// Lombok detection
	if len(lombokAnnotations) > 0 {
		var evidence []string
		for annot := range lombokAnnotations {
			if desc, ok := lombokAnnotationLabels[annot]; ok {
				evidence = append(evidence, desc)
			}
		}
		sort.Strings(evidence)
		confidence := float64(len(lombokAnnotations)) / float64(len(lombokAnnotationLabels))
		if confidence > 1.0 {
			confidence = 1.0
		}
		if confidence < 0.5 {
			confidence = 0.5
		}
		frameworks = append(frameworks, JavaFramework{
			Name:       "Lombok",
			Confidence: confidence,
			Evidence:   evidence,
		})
	}

	// Hibernate/JPA detection
	hibernateEvidence := detectHibernateEvidence(allImports)
	if len(hibernateEvidence) > 0 {
		frameworks = append(frameworks, JavaFramework{
			Name:       "Hibernate",
			Confidence: 0.8,
			Evidence:   hibernateEvidence,
		})
	}

	return frameworks
}

// detectSpringEvidence scans imports and annotations for Spring-related patterns.
func detectSpringEvidence(imports map[string]bool, annotations map[string]bool) []string {
	var evidence []string
	seen := make(map[string]bool)

	// Evidence from import paths
	springImportPrefixes := map[string]string{
		"org.springframework.web.bind.annotation":                      "@RestController",
		"org.springframework.stereotype.Service":                       "@Service",
		"org.springframework.stereotype.Repository":                    "@Repository",
		"org.springframework.stereotype.Component":                     "@Component",
		"org.springframework.context.annotation.Configuration":         "@Configuration",
		"org.springframework.boot.autoconfigure.SpringBootApplication": "@SpringBootApplication",
		"org.springframework.boot.SpringApplication":                   "@SpringBootApplication",
		"org.springframework.context.annotation.Bean":                  "@Bean",
		"javax.persistence.Entity":                                     "@Entity",
		"jakarta.persistence.Entity":                                   "@Entity",
	}

	for imp := range imports {
		for prefix, annotation := range springImportPrefixes {
			if strings.HasPrefix(imp, prefix) || imp == prefix {
				if !seen[annotation] {
					seen[annotation] = true
					if desc, ok := springAnnotations[annotation]; ok {
						evidence = append(evidence, desc)
					}
				}
			}
		}
	}

	// Evidence from directly-observed annotations
	annotSeen := make(map[string]bool)
	for annot := range annotations {
		if desc, ok := springAnnotations[annot]; ok {
			if !annotSeen[annot] {
				annotSeen[annot] = true
				evidence = append(evidence, desc+" (annotation in source)")
			}
		}
	}

	sort.Strings(evidence)
	return evidence
}

// detectHibernateEvidence scans imports for Hibernate/JPA patterns.
func detectHibernateEvidence(imports map[string]bool) []string {
	var evidence []string
	seen := make(map[string]bool)

	hibernatePatterns := map[string]string{
		"org.hibernate":                "Hibernate session/entity imports",
		"javax.persistence":            "JPA persistence annotations",
		"jakarta.persistence":          "Jakarta persistence annotations",
		"org.springframework.data.jpa": "Spring Data JPA",
	}

	for imp := range imports {
		for prefix, desc := range hibernatePatterns {
			if strings.HasPrefix(imp, prefix) && !seen[prefix] {
				seen[prefix] = true
				evidence = append(evidence, desc)
			}
		}
	}

	sort.Strings(evidence)
	return evidence
}

// joinPaths combines a class-level prefix with a method-level path.
func joinPaths(prefix, suffix string) string {
	if prefix == "" {
		return suffix
	}
	if suffix == "" || suffix == "/" {
		return prefix
	}
	prefix = strings.TrimSuffix(prefix, "/")
	suffix = strings.TrimPrefix(suffix, "/")
	result := prefix + "/" + suffix
	return strings.TrimSuffix(result, "/")
}
