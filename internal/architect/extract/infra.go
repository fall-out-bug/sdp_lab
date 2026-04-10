// Package extract contains Extractor implementations for the AI Architect module.
package extract

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"

	"sdp_dev/internal/architect"
)

// InfraExtractor scans a project tree for infrastructure artifacts and returns
// a ProfileFragment with InfraInfo and InfraArtifacts populated.
//
// It handles:
//   - Dockerfile: base images (FROM), exposed ports (EXPOSE)
//   - docker-compose.yml: services, depends_on, ports
//   - Kubernetes YAML: Deployment/StatefulSet container specs
//   - Terraform .tf: resource blocks
//   - GitHub Actions workflows: services sections
type InfraExtractor struct{}

var _ architect.Extractor = (*InfraExtractor)(nil)

// Name returns the extractor identifier.
func (e *InfraExtractor) Name() string { return "infra" }

// Extract walks root and populates InfraInfo.
func (e *InfraExtractor) Extract(ctx context.Context, root string) (*architect.ProfileFragment, error) {
	info := &architect.InfraInfo{}
	var artifacts []string

	var walkErr error
	err := filepath.Walk(root, func(path string, fi os.FileInfo, err error) error {
		if err != nil {
			return nil // skip unreadable entries
		}
		if fi.IsDir() {
			base := fi.Name()
			// skip common vendored / hidden dirs
			if base == "vendor" || base == "node_modules" || base == ".git" {
				return filepath.SkipDir
			}
			return nil
		}

		rel, _ := filepath.Rel(root, path)
		name := fi.Name()

		switch {
		case isDockerfile(name):
			if perr := parseDockerfile(path, info); perr != nil {
				walkErr = perr
			}
			artifacts = append(artifacts, rel)

		case isComposeFile(name):
			if perr := parseComposeFile(path, info); perr != nil {
				walkErr = perr
			}
			artifacts = append(artifacts, rel)

		case isKubernetesYAML(rel, name):
			if perr := parseKubernetesYAML(path, info); perr != nil {
				walkErr = perr
			}
			artifacts = append(artifacts, rel)

		case isTerraformFile(name):
			if perr := parseTerraformFile(path, info); perr != nil {
				walkErr = perr
			}
			artifacts = append(artifacts, rel)

		case isGitHubWorkflow(rel):
			if perr := parseGitHubWorkflow(path, info); perr != nil {
				walkErr = perr
			}
			artifacts = append(artifacts, rel)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walk %s: %w", root, err)
	}
	if walkErr != nil {
		return nil, walkErr
	}

	info.DeploymentType = detectDeploymentType(artifacts)

	frag := &architect.ProfileFragment{
		Infra:          info,
		InfraArtifacts: artifacts,
	}
	return frag, nil
}

// ---------------------------------------------------------------------------
// File matchers
// ---------------------------------------------------------------------------

func isDockerfile(name string) bool {
	lower := strings.ToLower(name)
	return lower == "dockerfile" || strings.HasPrefix(lower, "dockerfile.")
}

func isComposeFile(name string) bool {
	lower := strings.ToLower(name)
	return lower == "docker-compose.yml" || lower == "docker-compose.yaml" ||
		lower == "compose.yml" || lower == "compose.yaml"
}

func isKubernetesYAML(rel, name string) bool {
	lower := strings.ToLower(name)
	if !(strings.HasSuffix(lower, ".yml") || strings.HasSuffix(lower, ".yaml")) {
		return false
	}
	// Heuristic: files under k8s/, kubernetes/, deploy/, manifests/ directories
	relLower := strings.ToLower(rel)
	for _, prefix := range []string{"k8s/", "k8s" + string(os.PathSeparator),
		"kubernetes/", "kubernetes" + string(os.PathSeparator),
		"deploy/", "deploy" + string(os.PathSeparator),
		"manifests/", "manifests" + string(os.PathSeparator)} {
		if strings.HasPrefix(relLower, prefix) {
			return true
		}
	}
	return false
}

func isTerraformFile(name string) bool {
	return strings.HasSuffix(strings.ToLower(name), ".tf")
}

func isGitHubWorkflow(rel string) bool {
	normalized := filepath.ToSlash(rel)
	return strings.HasPrefix(normalized, ".github/workflows/") &&
		(strings.HasSuffix(normalized, ".yml") || strings.HasSuffix(normalized, ".yaml"))
}

// ---------------------------------------------------------------------------
// Dockerfile parsing
// ---------------------------------------------------------------------------

// reFrom matches FROM lines: FROM <image>[:<tag>][ AS <stage>]
var reFrom = regexp.MustCompile(`(?i)^\s*FROM\s+(\S+)`)

// reExpose matches EXPOSE lines: EXPOSE <port>[/<proto>] ...
var reExpose = regexp.MustCompile(`(?i)^\s*EXPOSE\s+(.+)`)

func parseDockerfile(path string, info *architect.InfraInfo) error {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open dockerfile %s: %w", path, err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()

		if m := reFrom.FindStringSubmatch(line); m != nil {
			img := m[1]
			// Skip ARG variable references like $BASE
			if !strings.HasPrefix(img, "$") {
				info.BaseImages = appendUnique(info.BaseImages, img)
			}
		}

		if m := reExpose.FindStringSubmatch(line); m != nil {
			for _, port := range strings.Fields(m[1]) {
				info.ExposedPorts = appendUnique(info.ExposedPorts, port)
			}
		}
	}
	return scanner.Err()
}

// ---------------------------------------------------------------------------
// docker-compose.yml parsing
// ---------------------------------------------------------------------------

// composeFile is a minimal representation of a docker-compose file.
type composeFile struct {
	Services map[string]composeService `yaml:"services"`
}

type composeService struct {
	Image     string            `yaml:"image"`
	Build     interface{}       `yaml:"build"` // string or map
	Ports     []string          `yaml:"ports"`
	DependsOn interface{}       `yaml:"depends_on"` // list or map
	Labels    map[string]string `yaml:"labels"`
}

func parseComposeFile(path string, info *architect.InfraInfo) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read compose %s: %w", path, err)
	}

	var cf composeFile
	if err := yaml.Unmarshal(data, &cf); err != nil {
		return fmt.Errorf("parse compose %s: %w", path, err)
	}

	rel := filepath.Base(path)

	for name, svc := range cf.Services {
		ci := architect.ContainerInfo{
			Name:   name,
			Image:  svc.Image,
			Type:   "service",
			Source: rel,
		}
		for _, p := range svc.Ports {
			ci.Ports = append(ci.Ports, p)
			info.ExposedPorts = appendUnique(info.ExposedPorts, p)
		}
		info.Containers = append(info.Containers, ci)

		// depends_on can be a list or a map
		deps := parseDependsOn(svc.DependsOn)
		for _, dep := range deps {
			info.Services = append(info.Services, architect.ServiceDep{From: name, To: dep})
		}
	}
	return nil
}

func parseDependsOn(v interface{}) []string {
	if v == nil {
		return nil
	}
	switch val := v.(type) {
	case []interface{}:
		out := make([]string, 0, len(val))
		for _, item := range val {
			if s, ok := item.(string); ok {
				out = append(out, s)
			}
		}
		return out
	case map[string]interface{}:
		out := make([]string, 0, len(val))
		for k := range val {
			out = append(out, k)
		}
		return out
	}
	return nil
}

// ---------------------------------------------------------------------------
// Kubernetes YAML parsing
// ---------------------------------------------------------------------------

// k8sDoc is a minimal representation of a k8s resource.
type k8sDoc struct {
	APIVersion string   `yaml:"apiVersion"`
	Kind       string   `yaml:"kind"`
	Metadata   k8sMeta  `yaml:"metadata"`
	Spec       k8sSpec  `yaml:"spec"`
}

type k8sMeta struct {
	Name string `yaml:"name"`
}

type k8sSpec struct {
	Template k8sPodTemplate `yaml:"template"`
}

type k8sPodTemplate struct {
	Spec k8sPodSpec `yaml:"spec"`
}

type k8sPodSpec struct {
	Containers []k8sContainer `yaml:"containers"`
}

type k8sContainer struct {
	Name  string      `yaml:"name"`
	Image string      `yaml:"image"`
	Ports []k8sPort   `yaml:"ports"`
}

type k8sPort struct {
	ContainerPort int    `yaml:"containerPort"`
	Protocol      string `yaml:"protocol"`
}

func parseKubernetesYAML(path string, info *architect.InfraInfo) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read k8s %s: %w", path, err)
	}

	rel := filepath.Base(path)

	// A single YAML file can contain multiple documents separated by "---"
	docs := splitYAMLDocs(data)
	for _, doc := range docs {
		if len(strings.TrimSpace(string(doc))) == 0 {
			continue
		}
		var kd k8sDoc
		if err := yaml.Unmarshal(doc, &kd); err != nil {
			continue // skip unparseable docs
		}

		kind := strings.ToLower(kd.Kind)
		if kind != "deployment" && kind != "statefulset" {
			continue
		}

		containerType := kind // "deployment" or "statefulset"

		for _, c := range kd.Spec.Template.Spec.Containers {
			ci := architect.ContainerInfo{
				Name:   c.Name,
				Image:  c.Image,
				Type:   containerType,
				Source: rel,
			}
			for _, p := range c.Ports {
				portStr := fmt.Sprintf("%d", p.ContainerPort)
				if p.Protocol != "" && strings.ToUpper(p.Protocol) != "TCP" {
					portStr += "/" + strings.ToLower(p.Protocol)
				}
				ci.Ports = append(ci.Ports, portStr)
				info.ExposedPorts = appendUnique(info.ExposedPorts, portStr)
			}
			if c.Image != "" {
				info.BaseImages = appendUnique(info.BaseImages, c.Image)
			}
			info.Containers = append(info.Containers, ci)
		}
	}
	return nil
}

// splitYAMLDocs splits a byte slice into individual YAML documents.
func splitYAMLDocs(data []byte) [][]byte {
	var docs [][]byte
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	var current strings.Builder

	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "---" {
			if current.Len() > 0 {
				docs = append(docs, []byte(current.String()))
				current.Reset()
			}
			continue
		}
		current.WriteString(line)
		current.WriteString("\n")
	}
	if current.Len() > 0 {
		docs = append(docs, []byte(current.String()))
	}
	return docs
}

// ---------------------------------------------------------------------------
// Terraform .tf parsing
// ---------------------------------------------------------------------------

// reResource matches: resource "type" "name" {
var reResource = regexp.MustCompile(`(?m)^\s*resource\s+"([^"]+)"\s+"([^"]+)"`)

func parseTerraformFile(path string, info *architect.InfraInfo) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read terraform %s: %w", path, err)
	}

	rel := filepath.Base(path)
	matches := reResource.FindAllStringSubmatch(string(data), -1)
	for _, m := range matches {
		resType := m[1]
		resName := m[2]
		provider := providerFromType(resType)
		info.Resources = append(info.Resources, architect.ResourceInfo{
			Type:     resType,
			Name:     resName,
			Provider: provider,
			Source:   rel,
		})
	}
	return nil
}

// providerFromType extracts the provider prefix from a Terraform resource type.
// e.g. "aws_s3_bucket" -> "aws", "google_compute_instance" -> "google".
func providerFromType(resType string) string {
	parts := strings.SplitN(resType, "_", 2)
	if len(parts) >= 1 {
		return parts[0]
	}
	return ""
}

// ---------------------------------------------------------------------------
// GitHub Actions workflow parsing
// ---------------------------------------------------------------------------

// ghWorkflow is a minimal representation of a GitHub Actions workflow file.
type ghWorkflow struct {
	Jobs map[string]ghJob `yaml:"jobs"`
}

type ghJob struct {
	Services map[string]ghService `yaml:"services"`
}

type ghService struct {
	Image string   `yaml:"image"`
	Ports []string `yaml:"ports"`
}

func parseGitHubWorkflow(path string, info *architect.InfraInfo) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read workflow %s: %w", path, err)
	}

	var wf ghWorkflow
	if err := yaml.Unmarshal(data, &wf); err != nil {
		return fmt.Errorf("parse workflow %s: %w", path, err)
	}

	rel := filepath.Base(path)

	for _, job := range wf.Jobs {
		for name, svc := range job.Services {
			ci := architect.ContainerInfo{
				Name:   name,
				Image:  svc.Image,
				Type:   "service",
				Source: rel,
			}
			for _, p := range svc.Ports {
				ci.Ports = append(ci.Ports, p)
				info.ExposedPorts = appendUnique(info.ExposedPorts, p)
			}
			info.Containers = append(info.Containers, ci)
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// Deployment type detection
// ---------------------------------------------------------------------------

// detectDeploymentType applies a priority-based heuristic:
//
//	kubernetes > docker-compose > serverless > bare
func detectDeploymentType(artifacts []string) string {
	hasK8s := false
	hasCompose := false

	for _, a := range artifacts {
		normalized := filepath.ToSlash(strings.ToLower(a))
		if strings.HasPrefix(normalized, "k8s/") ||
			strings.HasPrefix(normalized, "kubernetes/") ||
			strings.HasPrefix(normalized, "deploy/") ||
			strings.HasPrefix(normalized, "manifests/") {
			hasK8s = true
		}
		base := filepath.Base(normalized)
		if base == "docker-compose.yml" || base == "docker-compose.yaml" ||
			base == "compose.yml" || base == "compose.yaml" {
			hasCompose = true
		}
	}

	switch {
	case hasK8s:
		return "kubernetes"
	case hasCompose:
		return "docker-compose"
	default:
		return "bare"
	}
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func appendUnique(slice []string, val string) []string {
	for _, s := range slice {
		if s == val {
			return slice
		}
	}
	return append(slice, val)
}
