package java

import (
	"bufio"
	"os"
	"regexp"
	"strings"
)

// Import scanning patterns
var (
	javaImportRe   = regexp.MustCompile(`^import\s+(static\s+)?([a-zA-Z0-9_.]+\*?);`)
	kotlinImportRe = regexp.MustCompile(`^import\s+([a-zA-Z0-9_.]+)`)
	scalaImportRe  = regexp.MustCompile(`^import\s+(.+)`)
)

// scanJavaImports reads only import statements from a Java file.
func scanJavaImports(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var imports []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		m := javaImportRe.FindStringSubmatch(line)
		if m != nil && len(m) > 2 {
			imports = append(imports, m[2])
		}
	}
	return imports, scanner.Err()
}

// scanKotlinImports reads only import statements from a Kotlin file.
func scanKotlinImports(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var imports []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		m := kotlinImportRe.FindStringSubmatch(line)
		if m != nil {
			imports = append(imports, m[1])
		}
	}
	return imports, scanner.Err()
}

// scanJavaFile performs full scanning of a Java file: imports, annotations,
// and package declaration.
func scanJavaFile(path, relPath string) ([]string, []JavaAnnotationSighting, string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, nil, "", err
	}
	defer f.Close()

	return scanJavaFromReader(f, relPath)
}

// scanJavaFromReader reads from a reader and extracts imports, annotations,
// and package declaration.
func scanJavaFromReader(f *os.File, relPath string) ([]string, []JavaAnnotationSighting, string, error) {
	var imports []string
	var annotations []JavaAnnotationSighting
	var packageDecl string
	lineNum := 0

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())

		// Skip comments
		if strings.HasPrefix(line, "//") || strings.HasPrefix(line, "/*") || strings.HasPrefix(line, "*") {
			continue
		}

		// Package declaration
		if m := javaPackageRe.FindStringSubmatch(line); m != nil {
			packageDecl = m[1]
			continue
		}

		// Import statements
		if m := javaImportRe.FindStringSubmatch(line); m != nil && len(m) > 2 {
			imports = append(imports, m[2])
			continue
		}

		// Spring annotations
		for _, match := range springAnnotRe.FindAllStringSubmatch(line, -1) {
			annotations = append(annotations, JavaAnnotationSighting{
				Annotation: "@" + match[1],
				File:       relPath,
				LineNumber: lineNum,
			})
		}

		// @Bean detection
		if springBeanRe.MatchString(line) {
			annotations = append(annotations, JavaAnnotationSighting{
				Annotation: "@Bean",
				File:       relPath,
				LineNumber: lineNum,
			})
		}

		// Lombok annotations
		for _, match := range lombokAnnotRe.FindAllStringSubmatch(line, -1) {
			annotations = append(annotations, JavaAnnotationSighting{
				Annotation: "@" + match[1],
				File:       relPath,
				LineNumber: lineNum,
			})
		}
	}

	return imports, annotations, packageDecl, scanner.Err()
}

// scanKotlinFile performs full scanning of a Kotlin file.
func scanKotlinFile(path, relPath string) ([]string, []JavaAnnotationSighting, string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, nil, "", err
	}
	defer f.Close()

	var imports []string
	var annotations []JavaAnnotationSighting
	var packageDecl string
	lineNum := 0

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())

		// Skip comments
		if strings.HasPrefix(line, "//") || strings.HasPrefix(line, "/*") || strings.HasPrefix(line, "*") {
			continue
		}

		// Package declaration
		if m := kotlinPackageRe.FindStringSubmatch(line); m != nil {
			packageDecl = m[1]
			continue
		}

		// Import statements
		if m := kotlinImportRe.FindStringSubmatch(line); m != nil {
			imports = append(imports, m[1])
			continue
		}

		// Spring annotations (Kotlin uses same annotations)
		for _, match := range springAnnotRe.FindAllStringSubmatch(line, -1) {
			annotations = append(annotations, JavaAnnotationSighting{
				Annotation: "@" + match[1],
				File:       relPath,
				LineNumber: lineNum,
			})
		}

		// @Bean detection
		if springBeanRe.MatchString(line) {
			annotations = append(annotations, JavaAnnotationSighting{
				Annotation: "@Bean",
				File:       relPath,
				LineNumber: lineNum,
			})
		}
	}

	return imports, annotations, packageDecl, scanner.Err()
}
