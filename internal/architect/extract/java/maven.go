package java

import (
	"os"
	"regexp"
	"strings"
)

// Maven pom.xml parsing patterns
var (
	pomDepBlockRe     = regexp.MustCompile(`(?s)<dependency>(.*?)</dependency>`)
	pomGroupRe        = regexp.MustCompile(`<groupId>\s*(.*?)\s*</groupId>`)
	pomArtifactRe     = regexp.MustCompile(`<artifactId>\s*(.*?)\s*</artifactId>`)
	pomScopeRe        = regexp.MustCompile(`<scope>\s*(.*?)\s*</scope>`)
	pomVersionRe      = regexp.MustCompile(`<version>\s*(.*?)\s*</version>`)
	pomModuleRe       = regexp.MustCompile(`<module>\s*(.*?)\s*</module>`)
	pomModulesRe      = regexp.MustCompile(`(?s)<modules>(.*?)</modules>`)
	pomParentRe       = regexp.MustCompile(`(?s)<parent>(.*?)</parent>`)
	pomPropertyTagRe  = regexp.MustCompile(`<([a-zA-Z][a-zA-Z0-9._-]*)>([^<]+)</[a-zA-Z0-9._-]+>`)
	pomTopArtifactIDRe = regexp.MustCompile(`(?s)<artifactId>\s*(.*?)\s*</artifactId>`)
)

// parsePomXML reads a pom.xml file and returns parsed dependencies.
func parsePomXML(path string) (*JavaBuildSystem, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	props := extractPomProperties(string(data))
	return parsePomXMLContent(string(data), props), nil
}

// parsePomXMLWithMeta reads a pom.xml file and returns both the parsed
// dependencies and the submodule's own artifactId.
func parsePomXMLWithMeta(path string) (*JavaBuildSystem, string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, "", err
	}
	content := string(data)
	props := extractPomProperties(content)
	bs := parsePomXMLContent(content, props)
	artifactID := extractPomArtifactID(content)
	return bs, artifactID, nil
}

// parsePomXMLContent parses Maven dependencies from pom.xml content string.
func parsePomXMLContent(content string, props map[string]string) *JavaBuildSystem {
	blocks := pomDepBlockRe.FindAllStringSubmatch(content, -1)
	bs := &JavaBuildSystem{Type: "maven"}

	for _, block := range blocks {
		dep := JavaDependency{}
		if m := pomGroupRe.FindStringSubmatch(block[1]); m != nil {
			dep.Group = resolvePomProperties(m[1], props)
		}
		if m := pomArtifactRe.FindStringSubmatch(block[1]); m != nil {
			dep.Artifact = resolvePomProperties(m[1], props)
		}
		if m := pomScopeRe.FindStringSubmatch(block[1]); m != nil {
			dep.Scope = m[1]
		}
		if dep.Group != "" || dep.Artifact != "" {
			bs.Dependencies = append(bs.Dependencies, dep)
		}
	}

	return bs
}

// extractPomProperties extracts property definitions from the <properties> section.
func extractPomProperties(content string) map[string]string {
	props := make(map[string]string)

	// Find <properties> block
	startIdx := strings.Index(content, "<properties>")
	if startIdx < 0 {
		return props
	}
	endIdx := strings.Index(content[startIdx:], "</properties>")
	if endIdx < 0 {
		return props
	}

	propsBlock := content[startIdx : startIdx+endIdx]
	for _, m := range pomPropertyTagRe.FindAllStringSubmatch(propsBlock, -1) {
		if len(m) >= 3 {
			props[m[1]] = m[2]
		}
	}

	return props
}

// resolvePomProperties replaces ${property.name} placeholders using the provided map.
func resolvePomProperties(s string, props map[string]string) string {
	if len(props) == 0 {
		return s
	}
	for name, value := range props {
		if value != "" {
			s = strings.ReplaceAll(s, "${"+name+"}", value)
		}
	}
	return s
}

// extractPomArtifactID returns the first top-level <artifactId> from pom.xml content.
func extractPomArtifactID(content string) string {
	// Search zone: everything before <dependencies>
	searchZone := content
	if idx := strings.Index(content, "<dependencies>"); idx > 0 {
		searchZone = content[:idx]
	}

	// Strip <parent>...</parent> to avoid matching parent's artifactId
	searchZone = pomParentRe.ReplaceAllString(searchZone, "")

	m := pomTopArtifactIDRe.FindStringSubmatch(searchZone)
	if len(m) >= 2 {
		return m[1]
	}
	return ""
}

// parseModules extracts module names from a pom.xml <modules> section.
func parseModules(path string) []string {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	content := string(data)

	modulesBlock := pomModulesRe.FindStringSubmatch(content)
	if modulesBlock == nil {
		return nil
	}

	matches := pomModuleRe.FindAllStringSubmatch(modulesBlock[1], -1)
	var modules []string
	for _, m := range matches {
		modules = append(modules, m[1])
	}
	return modules
}
