// Package extract contains Extractor implementations for the AI Architect module.
package extract

import (
	"bufio"
	"context"
	"encoding/json"
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
//   - Dockerfile: base images (FROM), exposed ports (EXPOSE), ENTRYPOINT/CMD, ENV
//   - docker-compose.yml: services, networks, volumes, environment variables, depends_on
//   - Kubernetes YAML: Deployments, StatefulSets, Services, ConfigMaps, Ingress
//   - Terraform .tf: resource blocks, module blocks, variables
//   - GitHub Actions workflows: services, jobs, triggers
//   - GitLab CI .gitlab-ci.yml: stages, services, images
//   - Jenkinsfile: stages, agents
//   - Module boundaries: Maven pom.xml, Gradle settings.gradle, npm workspaces, Go cmd/
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
			if perr := parseDockerfile(path, rel, info); perr != nil {
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

		case isGitLabCI(rel, name):
			if perr := parseGitLabCI(path, info); perr != nil {
				walkErr = perr
			}
			artifacts = append(artifacts, rel)

		case isJenkinsfile(name):
			if perr := parseJenkinsfile(path, info); perr != nil {
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

	// Module boundary detection
	detectModuleBoundaries(root, info)

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

func isGitLabCI(rel, name string) bool {
	return name == ".gitlab-ci.yml" || name == ".gitlab-ci.yaml"
}

func isJenkinsfile(name string) bool {
	return name == "Jenkinsfile" || strings.HasPrefix(name, "Jenkinsfile.")
}

// ---------------------------------------------------------------------------
// Dockerfile parsing
// ---------------------------------------------------------------------------

// reFrom matches FROM lines: FROM <image>[:<tag>][ AS <stage>]
var reFrom = regexp.MustCompile(`(?i)^\s*FROM\s+(\S+)`)

// reExpose matches EXPOSE lines: EXPOSE <port>[/<proto>] ...
var reExpose = regexp.MustCompile(`(?i)^\s*EXPOSE\s+(.+)`)

// reEntrypoint matches ENTRYPOINT in exec or shell form.
var reEntrypoint = regexp.MustCompile(`(?i)^\s*ENTRYPOINT\s+(.+)`)

// reCmd matches CMD in exec or shell form.
var reCmd = regexp.MustCompile(`(?i)^\s*CMD\s+(.+)`)

// reEnv matches ENV lines: ENV KEY=VALUE or ENV KEY VALUE
var reEnv = regexp.MustCompile(`(?i)^\s*ENV\s+(.+)`)

func parseDockerfile(path, relPath string, info *architect.InfraInfo) error {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open dockerfile %s: %w", path, err)
	}
	defer f.Close()

	// Derive container name from directory path
	dir := filepath.Dir(path)
	baseName := filepath.Base(dir)
	if baseName == "." || strings.ToLower(baseName) == "docker" || strings.ToLower(baseName) == "dockerfiles" {
		parentDir := filepath.Base(filepath.Dir(dir))
		if parentDir != "." {
			baseName = parentDir
		} else {
			baseName = "default"
		}
	}

	var (
		foundFrom    bool
		lastImage    string
		entrypoint   string
		cmd          string
		envRefs      []string
	)

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()

		if m := reFrom.FindStringSubmatch(line); m != nil {
			img := m[1]
			// Skip ARG variable references like $BASE
			if !strings.HasPrefix(img, "$") {
				info.BaseImages = appendUnique(info.BaseImages, img)
				lastImage = img
				foundFrom = true
			}
		}

		if m := reExpose.FindStringSubmatch(line); m != nil {
			for _, port := range strings.Fields(m[1]) {
				info.ExposedPorts = appendUnique(info.ExposedPorts, port)
			}
		}

		if m := reEntrypoint.FindStringSubmatch(line); m != nil {
			entrypoint = cleanInstruction(m[1])
		}

		if m := reCmd.FindStringSubmatch(line); m != nil {
			cmd = cleanInstruction(m[1])
		}

		if m := reEnv.FindStringSubmatch(line); m != nil {
			envName := extractEnvName(m[1])
			if envName != "" {
				envRefs = appendUnique(envRefs, envName)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	// Create a ContainerInfo entry if we found a FROM instruction
	if foundFrom {
		ci := architect.ContainerInfo{
			Name:       baseName,
			Image:      lastImage,
			Type:       "service",
			Source:     relPath,
			Entrypoint: entrypoint,
			Cmd:        cmd,
			EnvRefs:    envRefs,
		}
		info.Containers = append(info.Containers, ci)
	}

	return nil
}

// cleanInstruction strips surrounding quotes and brackets from Dockerfile instructions.
func cleanInstruction(s string) string {
	s = strings.TrimSpace(s)
	// Handle JSON array form: ["exec", "arg"]
	if strings.HasPrefix(s, "[") {
		var parts []string
		if err := json.Unmarshal([]byte(s), &parts); err == nil {
			return strings.Join(parts, " ")
		}
	}
	// Strip surrounding quotes
	s = strings.Trim(s, `"`)
	return s
}

// extractEnvName extracts the variable name from an ENV instruction value.
// Handles both "KEY=VALUE" and "KEY VALUE" forms.
func extractEnvName(s string) string {
	s = strings.TrimSpace(s)
	if idx := strings.Index(s, "="); idx >= 0 {
		return s[:idx]
	}
	// Space-separated form: "KEY VALUE"
	if idx := strings.Index(s, " "); idx >= 0 {
		return s[:idx]
	}
	return s
}

// ---------------------------------------------------------------------------
// docker-compose.yml parsing
// ---------------------------------------------------------------------------

// composeFile is a minimal representation of a docker-compose file.
type composeFile struct {
	Services map[string]composeService `yaml:"services"`
	Networks interface{}               `yaml:"networks"` // map or nil
	Volumes  interface{}               `yaml:"volumes"`  // map or nil
}

type composeService struct {
	Image     string            `yaml:"image"`
	Build     interface{}       `yaml:"build"` // string or map
	Ports     []string          `yaml:"ports"`
	DependsOn interface{}       `yaml:"depends_on"` // list or map
	Labels    map[string]string `yaml:"labels"`
	EnvFile   interface{}       `yaml:"env_file"`   // string or list
	Networks  interface{}       `yaml:"networks"`   // list or map
	Volumes   interface{}       `yaml:"volumes"`    // list or map
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

	// Extract top-level network names
	info.Networks = append(info.Networks, extractMapKeys(cf.Networks)...)

	// Extract top-level volume names
	info.Volumes = append(info.Volumes, extractMapKeys(cf.Volumes)...)

	for name, svc := range cf.Services {
		ci := architect.ContainerInfo{
			Name:      name,
			Image:     svc.Image,
			Type:      "service",
			Source:    rel,
			Networks:  extractStringList(svc.Networks),
			DependsOn: parseDependsOn(svc.DependsOn),
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

// extractMapKeys returns the keys from a YAML map (interface{}).
func extractMapKeys(v interface{}) []string {
	if v == nil {
		return nil
	}
	if m, ok := v.(map[string]interface{}); ok {
		keys := make([]string, 0, len(m))
		for k := range m {
			keys = append(keys, k)
		}
		return keys
	}
	return nil
}

// extractStringList returns string keys from a YAML list or map.
func extractStringList(v interface{}) []string {
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
	APIVersion string       `yaml:"apiVersion"`
	Kind       string       `yaml:"kind"`
	Metadata   k8sMeta      `yaml:"metadata"`
	Spec       k8sSpecRaw   `yaml:"spec"`
}

type k8sMeta struct {
	Name      string `yaml:"name"`
	Namespace string `yaml:"namespace"`
}

// k8sSpecRaw uses a generic map to handle different resource types.
type k8sSpecRaw struct {
	Template *k8sPodTemplate `yaml:"template"`
	Ports    []k8sServicePort `yaml:"ports"`
	Type     string          `yaml:"type"`
	Selector map[string]string `yaml:"selector"`
	Rules    []k8sIngressRule `yaml:"rules"`
	Data     map[string]interface{} `yaml:"data"`
}

type k8sPodTemplate struct {
	Spec k8sPodSpec `yaml:"spec"`
}

type k8sPodSpec struct {
	Containers []k8sContainer `yaml:"containers"`
}

type k8sContainer struct {
	Name  string    `yaml:"name"`
	Image string    `yaml:"image"`
	Ports []k8sPort `yaml:"ports"`
	Env   []k8sEnv  `yaml:"env"`
}

type k8sPort struct {
	ContainerPort int    `yaml:"containerPort"`
	Protocol      string `yaml:"protocol"`
}

type k8sEnv struct {
	Name  string `yaml:"name"`
	Value string `yaml:"value"`
}

type k8sServicePort struct {
	Name       string `yaml:"name"`
	Port       int    `yaml:"port"`
	TargetPort int    `yaml:"targetPort"`
	Protocol   string `yaml:"protocol"`
}

type k8sIngressRule struct {
	Host string           `yaml:"host"`
	HTTP *k8sHTTPIngress  `yaml:"http"`
}

type k8sHTTPIngress struct {
	Paths []k8sHTTPIngressPath `yaml:"paths"`
}

type k8sHTTPIngressPath struct {
	Path string `yaml:"path"`
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
		switch kind {
		case "deployment", "statefulset", "daemonset", "job", "cronjob":
			parseK8sWorkload(kd, rel, kind, info)
		case "service":
			parseK8sService(kd, rel, info)
		case "ingress":
			parseK8sIngress(kd, rel, info)
		case "configmap":
			parseK8sConfigMap(kd, rel, info)
		case "namespace":
			// Namespaces are informational; no structured extraction needed
		}
	}
	return nil
}

func parseK8sWorkload(kd k8sDoc, rel, kind string, info *architect.InfraInfo) {
	if kd.Spec.Template == nil {
		return
	}
	for _, c := range kd.Spec.Template.Spec.Containers {
		ci := architect.ContainerInfo{
			Name:   c.Name,
			Image:  c.Image,
			Type:   kind,
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
		// Extract env var names (values are sanitized -- only names recorded)
		for _, env := range c.Env {
			ci.EnvRefs = appendUnique(ci.EnvRefs, env.Name)
		}
		if c.Image != "" {
			info.BaseImages = appendUnique(info.BaseImages, c.Image)
		}
		info.Containers = append(info.Containers, ci)
	}
}

func parseK8sService(kd k8sDoc, rel string, info *architect.InfraInfo) {
	svc := architect.K8sServiceInfo{
		Name:      kd.Metadata.Name,
		Namespace: kd.Metadata.Namespace,
		Type:      kd.Spec.Type,
		Selector:  kd.Spec.Selector,
		Source:    rel,
	}
	for _, p := range kd.Spec.Ports {
		portStr := fmt.Sprintf("%d", p.Port)
		if p.TargetPort != 0 {
			portStr += fmt.Sprintf(":%d", p.TargetPort)
		}
		svc.Ports = append(svc.Ports, portStr)
	}
	info.K8sServices = append(info.K8sServices, svc)
}

func parseK8sIngress(kd k8sDoc, rel string, info *architect.InfraInfo) {
	ing := architect.IngressInfo{
		Name:      kd.Metadata.Name,
		Namespace: kd.Metadata.Namespace,
		Source:    rel,
	}
	for _, rule := range kd.Spec.Rules {
		if rule.Host != "" {
			ing.Hosts = appendUnique(ing.Hosts, rule.Host)
		}
		if rule.HTTP != nil {
			for _, p := range rule.HTTP.Paths {
				ing.Paths = appendUnique(ing.Paths, p.Path)
			}
		}
	}
	info.Ingresses = append(info.Ingresses, ing)
}

func parseK8sConfigMap(kd k8sDoc, rel string, info *architect.InfraInfo) {
	cm := architect.ConfigMapInfo{
		Name:      kd.Metadata.Name,
		Namespace: kd.Metadata.Namespace,
		Source:    rel,
	}
	for k := range kd.Spec.Data {
		cm.Keys = append(cm.Keys, k)
	}
	info.ConfigMaps = append(info.ConfigMaps, cm)
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

// reModule matches: module "name" {
var reModule = regexp.MustCompile(`(?m)^\s*module\s+"([^"]+)"`)

// reVar matches: variable "name" {
var reVar = regexp.MustCompile(`(?m)^\s*variable\s+"([^"]+)"`)

// reVarDefault matches: default = ...
var reVarDefault = regexp.MustCompile(`(?m)^\s*default\s*=\s*(.+)`)

// reVarType matches: type = ...
var reVarType = regexp.MustCompile(`(?m)^\s*type\s*=\s*(.+)`)

// reData matches: data "type" "name" {
var reData = regexp.MustCompile(`(?m)^\s*data\s+"([^"]+)"\s+"([^"]+)"`)

func parseTerraformFile(path string, info *architect.InfraInfo) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read terraform %s: %w", path, err)
	}

	content := string(data)
	rel := filepath.Base(path)

	// Parse resources
	for _, m := range reResource.FindAllStringSubmatch(content, -1) {
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

	// Parse data sources (recorded as resources with "data:" prefix)
	for _, m := range reData.FindAllStringSubmatch(content, -1) {
		resType := "data:" + m[1]
		resName := m[2]
		provider := providerFromType(m[1])
		info.Resources = append(info.Resources, architect.ResourceInfo{
			Type:     resType,
			Name:     resName,
			Provider: provider,
			Source:   rel,
		})
	}

	// Parse module blocks (recorded as resources with type "module")
	for _, m := range reModule.FindAllStringSubmatch(content, -1) {
		info.Resources = append(info.Resources, architect.ResourceInfo{
			Type:     "module",
			Name:     m[1],
			Provider: "terraform",
			Source:   rel,
		})
	}

	// Parse variable blocks
	vars := parseTerraformVars(content, rel)
	info.TerraformVars = append(info.TerraformVars, vars...)

	return nil
}

// parseTerraformVars extracts variable definitions from Terraform content.
// Uses block-based parsing to associate default/type with each variable.
func parseTerraformVars(content, rel string) []architect.TerraformVarInfo {
	var result []architect.TerraformVarInfo

	// Find all variable block positions
	varMatches := reVar.FindAllStringSubmatchIndex(content, -1)
	for _, loc := range varMatches {
		name := content[loc[2]:loc[3]]

		// Extract the block body (from opening { to closing })
		blockStart := loc[1]
		blockBody := extractBlockBody(content, blockStart)

		vi := architect.TerraformVarInfo{
			Name:   name,
			Source: rel,
		}

		if m := reVarDefault.FindStringSubmatch(blockBody); m != nil {
			vi.Default = strings.TrimSpace(m[1])
		}
		if m := reVarType.FindStringSubmatch(blockBody); m != nil {
			vi.Type = strings.TrimSpace(m[1])
		}

		result = append(result, vi)
	}
	return result
}

// extractBlockBody extracts content between balanced braces starting from pos.
func extractBlockBody(content string, pos int) string {
	depth := 0
	start := -1
	for i := pos; i < len(content); i++ {
		if content[i] == '{' {
			if depth == 0 {
				start = i + 1
			}
			depth++
		} else if content[i] == '}' {
			depth--
			if depth == 0 && start >= 0 {
				return content[start:i]
			}
		}
	}
	if start >= 0 {
		return content[start:]
	}
	return ""
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
	On      interface{}       `yaml:"on"` // string, list, or map
	Jobs    map[string]ghJob  `yaml:"jobs"`
}

type ghJob struct {
	Services map[string]ghService `yaml:"services"`
	Steps    []ghStep             `yaml:"steps"`
}

type ghStep struct {
	Uses string `yaml:"uses"`
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

	// Extract triggers
	triggers := parseGHTriggers(wf.On)

	// Extract CI job info
	for jobName, job := range wf.Jobs {
		ciJob := architect.CIJobInfo{
			Name:     jobName,
			Source:   rel,
			Triggers: triggers,
		}

		// Detect deploy targets from step uses
		for _, step := range job.Steps {
			if strings.Contains(step.Uses, "deploy") || strings.Contains(step.Uses, "release") {
				ciJob.DeployTargets = appendUnique(ciJob.DeployTargets, step.Uses)
			}
		}

		info.CIJobs = append(info.CIJobs, ciJob)

		// Extract service containers
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

// parseGHTriggers extracts trigger names from the "on" field.
func parseGHTriggers(v interface{}) []string {
	if v == nil {
		return nil
	}
	switch val := v.(type) {
	case string:
		return []string{val}
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
// GitLab CI parsing
// ---------------------------------------------------------------------------

func parseGitLabCI(path string, info *architect.InfraInfo) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read gitlab-ci %s: %w", path, err)
	}

	// Parse as generic map to handle the inline job structure
	var raw map[string]interface{}
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("parse gitlab-ci %s: %w", path, err)
	}

	rel := filepath.Base(path)

	// Known non-job keys in .gitlab-ci.yml
	nonJobKeys := map[string]bool{
		"stages": true, "variables": true, "default": true,
		"include": true, "image": true, "services": true,
		"before_script": true, "after_script": true,
		"cache": true, "artifacts": true,
		"workflow": true,
	}

	for key, val := range raw {
		if nonJobKeys[key] {
			continue
		}
		// Try to parse as a job
		jobMap, ok := val.(map[string]interface{})
		if !ok {
			continue
		}

		ciJob := architect.CIJobInfo{
			Name:   key,
			Source: rel,
		}

		if s, ok := jobMap["stage"].(string); ok {
			ciJob.Stage = s
		}
		if s, ok := jobMap["image"].(string); ok {
			ciJob.Image = s
		}
		if triggers := extractStringList(jobMap["only"]); triggers != nil {
			ciJob.Triggers = triggers
		}

		// Extract service containers
		if services, ok := jobMap["services"].([]interface{}); ok {
			for _, svc := range services {
				switch sv := svc.(type) {
				case string:
					ci := architect.ContainerInfo{
						Name:   sv,
						Image:  sv,
						Type:   "service",
						Source: rel,
					}
					info.Containers = append(info.Containers, ci)
				case map[string]interface{}:
					name := ""
					if n, ok := sv["name"].(string); ok {
						name = n
					}
					img := name
					if i, ok := sv["image"].(string); ok {
						img = i
					}
					ci := architect.ContainerInfo{
						Name:   name,
						Image:  img,
						Type:   "service",
						Source: rel,
					}
					info.Containers = append(info.Containers, ci)
				}
			}
		}

		info.CIJobs = append(info.CIJobs, ciJob)
	}

	return nil
}

// ---------------------------------------------------------------------------
// Jenkinsfile parsing
// ---------------------------------------------------------------------------

// reStage matches: stage('name') or stage("name")
var reStage = regexp.MustCompile(`(?m)stage\s*[\('"]+([\w\s\-./]+)[\)'"']`)

// reAgent matches: agent { ... } or agent none/any/label '...'
var reAgent = regexp.MustCompile(`(?im)agent\s*\{[^}]*image\s*['"]?([^'"}\s]+)`)

// reWhen matches: when { branch 'name' }
var reWhen = regexp.MustCompile(`(?im)when\s*\{[^}]*branch\s*['"]([^'"]+)['"]`)

// reDeployStep matches deploy-related step keywords
var reDeployStep = regexp.MustCompile(`(?im)(deploy|publish|release|push\s+image)`)

func parseJenkinsfile(path string, info *architect.InfraInfo) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read jenkinsfile %s: %w", path, err)
	}

	content := string(data)
	rel := filepath.Base(path)

	stages := reStage.FindAllStringSubmatch(content, -1)
	for _, m := range stages {
		stageName := strings.TrimSpace(m[1])
		ciJob := architect.CIJobInfo{
			Name:   stageName,
			Stage:  stageName,
			Source: rel,
		}

		// Check if stage name suggests a deployment target
		lowerName := strings.ToLower(stageName)
		if strings.Contains(lowerName, "deploy") || strings.Contains(lowerName, "release") ||
			strings.Contains(lowerName, "publish") || strings.Contains(lowerName, "production") ||
			strings.Contains(lowerName, "staging") {
			ciJob.DeployTargets = append(ciJob.DeployTargets, stageName)
		}

		info.CIJobs = append(info.CIJobs, ciJob)
	}

	// Extract agent images
	for _, m := range reAgent.FindAllStringSubmatch(content, -1) {
		info.BaseImages = appendUnique(info.BaseImages, m[1])
	}

	// Extract branch triggers
	for _, m := range reWhen.FindAllStringSubmatch(content, -1) {
		for i := range info.CIJobs {
			if info.CIJobs[i].Source == rel {
				info.CIJobs[i].Triggers = appendUnique(info.CIJobs[i].Triggers, "branch:"+m[1])
			}
		}
	}

	// Check for deploy steps
	if reDeployStep.MatchString(content) {
		for i := range info.CIJobs {
			if info.CIJobs[i].Source == rel && len(info.CIJobs[i].DeployTargets) == 0 {
				info.CIJobs[i].DeployTargets = append(info.CIJobs[i].DeployTargets, "pipeline")
			}
		}
	}

	return nil
}

// ---------------------------------------------------------------------------
// Module boundary detection
// ---------------------------------------------------------------------------

// rePomModule matches <module>name</module> in Maven pom.xml
var rePomModule = regexp.MustCompile(`(?s)<module>\s*(.*?)\s*</module>`)

// reGradleInclude matches include 'module' in settings.gradle
var reGradleInclude = regexp.MustCompile(`(?m)include\s+['"]([^'"]+)['"]`)

// reNpmWorkspace matches "workspaces" array in package.json
var reNpmWorkspace = regexp.MustCompile(`(?s)"workspaces"\s*:\s*\[([^\]]*)\]`)

// reNpmWorkspacePkg matches individual package names in workspaces array
var reNpmWorkspacePkg = regexp.MustCompile(`"([^"]+)"`)

func detectModuleBoundaries(root string, info *architect.InfraInfo) {
	// Maven: pom.xml with <modules>
	detectMavenModules(root, info)

	// Gradle: settings.gradle with include
	detectGradleModules(root, info)

	// npm workspaces: package.json with workspaces
	detectNpmWorkspaces(root, info)

	// Go: cmd/ directories
	detectGoCmdModules(root, info)
}

func detectMavenModules(root string, info *architect.InfraInfo) {
	// Walk and find pom.xml files with <modules>
	err := filepath.Walk(root, func(path string, fi os.FileInfo, err error) error {
		if err != nil || fi.IsDir() {
			return nil
		}
		if fi.Name() != "pom.xml" {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		matches := rePomModule.FindAllStringSubmatch(string(data), -1)
		if len(matches) == 0 {
			return nil
		}

		rel, _ := filepath.Rel(root, path)
		var children []string
		for _, m := range matches {
			children = append(children, m[1])
		}
		info.ModuleBoundaries = append(info.ModuleBoundaries, architect.ModuleBoundaryInfo{
			Name:        filepath.Dir(rel),
			BuildSystem: "maven",
			Path:        rel,
			Children:    children,
		})
		return nil
	})
	_ = err
}

func detectGradleModules(root string, info *architect.InfraInfo) {
	settingsPath := filepath.Join(root, "settings.gradle")
	if _, err := os.Stat(settingsPath); err != nil {
		// Also check settings.gradle.kts
		settingsPath = filepath.Join(root, "settings.gradle.kts")
		if _, err := os.Stat(settingsPath); err != nil {
			return
		}
	}
	data, err := os.ReadFile(settingsPath)
	if err != nil {
		return
	}

	matches := reGradleInclude.FindAllStringSubmatch(string(data), -1)
	if len(matches) == 0 {
		return
	}

	rel, _ := filepath.Rel(root, settingsPath)
	var children []string
	for _, m := range matches {
		children = append(children, m[1])
	}
	info.ModuleBoundaries = append(info.ModuleBoundaries, architect.ModuleBoundaryInfo{
		Name:        "gradle-root",
		BuildSystem: "gradle",
		Path:        rel,
		Children:    children,
	})
}

func detectNpmWorkspaces(root string, info *architect.InfraInfo) {
	pkgPath := filepath.Join(root, "package.json")
	data, err := os.ReadFile(pkgPath)
	if err != nil {
		return
	}

	// Extract workspaces array from package.json
	wsMatch := reNpmWorkspace.FindStringSubmatch(string(data))
	if wsMatch == nil {
		return
	}

	pkgs := reNpmWorkspacePkg.FindAllStringSubmatch(wsMatch[1], -1)
	if len(pkgs) == 0 {
		return
	}

	var children []string
	for _, p := range pkgs {
		children = append(children, p[1])
	}
	info.ModuleBoundaries = append(info.ModuleBoundaries, architect.ModuleBoundaryInfo{
		Name:        "npm-workspaces",
		BuildSystem: "npm",
		Path:        "package.json",
		Children:    children,
	})
}

func detectGoCmdModules(root string, info *architect.InfraInfo) {
	// Look for cmd/ directories containing main.go files
	cmdDir := filepath.Join(root, "cmd")
	entries, err := os.ReadDir(cmdDir)
	if err != nil {
		return
	}

	var children []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		// Check if directory contains a .go file (likely main.go)
		subPath := filepath.Join(cmdDir, e.Name())
		subEntries, err := os.ReadDir(subPath)
		if err != nil {
			continue
		}
		for _, se := range subEntries {
			if strings.HasSuffix(se.Name(), ".go") {
				children = append(children, "cmd/"+e.Name())
				break
			}
		}
	}

	if len(children) > 0 {
		info.ModuleBoundaries = append(info.ModuleBoundaries, architect.ModuleBoundaryInfo{
			Name:        "go-cmd",
			BuildSystem: "go",
			Path:        "cmd/",
			Children:    children,
		})
	}
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
