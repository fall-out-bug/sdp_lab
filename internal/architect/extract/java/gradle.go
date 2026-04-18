package java

import (
	"os"
	"path/filepath"
	"regexp"
)

// Gradle dependency parsing patterns
var (
	// Groovy DSL: implementation 'group:artifact:version'
	// Kotlin DSL: implementation("group:artifact:version")
	gradleDepRe = regexp.MustCompile(
		`(?:implementation|api|compile|compileOnly|runtimeOnly|testImplementation|testRuntimeOnly)\s*\(?` +
			`['"]([a-zA-Z0-9._-]+):([a-zA-Z0-9._-]+)(?::([a-zA-Z0-9._-]+))?['"]\)?`)

	// Gradle settings include patterns
	gradleIncludeRe       = regexp.MustCompile(`include\s*\(\s*['"]([^'"]+)['"]`)
	gradleIncludeSingleRe = regexp.MustCompile(`include\s+['"]([^'"]+)['"]`)
)

// parseBuildGradle reads a build.gradle or build.gradle.kts file and returns dependencies.
func parseBuildGradle(path string) (*JavaBuildSystem, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	content := string(data)

	matches := gradleDepRe.FindAllStringSubmatch(content, -1)
	bs := &JavaBuildSystem{Type: "gradle"}

	for _, m := range matches {
		dep := JavaDependency{
			Group:    m[1],
			Artifact: m[2],
		}
		bs.Dependencies = append(bs.Dependencies, dep)
	}

	return bs, nil
}

// parseSettingsGradle reads settings.gradle or settings.gradle.kts and extracts
// include directives to identify subprojects.
func parseSettingsGradle(rootDir string) []string {
	var subprojects []string

	for _, filename := range []string{"settings.gradle", "settings.gradle.kts"} {
		path := filepath.Join(rootDir, filename)
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		content := string(data)

		// include "project-name" style
		for _, m := range gradleIncludeRe.FindAllStringSubmatch(content, -1) {
			subprojects = append(subprojects, m[1])
		}

		// include 'project-name' style (single quote, no parens)
		for _, m := range gradleIncludeSingleRe.FindAllStringSubmatch(content, -1) {
			subprojects = append(subprojects, m[1])
		}

		break // only read the first one found
	}

	return subprojects
}

// extractProjectName extracts the rootProject.name from settings.gradle.
func extractProjectName(rootDir string) string {
	for _, filename := range []string{"settings.gradle", "settings.gradle.kts"} {
		path := filepath.Join(rootDir, filename)
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		content := string(data)

		// Match: rootProject.name = 'my-project'
		nameRe := regexp.MustCompile(`rootProject\.name\s*=\s*['"]([^'"]+)['"]`)
		if m := nameRe.FindStringSubmatch(content); len(m) > 1 {
			return m[1]
		}
	}

	return ""
}

// detectGradleMultiProject checks if a Gradle project is multi-module.
func detectGradleMultiProject(rootDir string) bool {
	subprojects := parseSettingsGradle(rootDir)
	return len(subprojects) > 0
}
