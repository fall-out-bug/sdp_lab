package routing

// RoutingExample is a labeled training example.
type RoutingExample struct {
	Title       string // workstream title or task description
	Description string
	Capability  string // e.g. "go-backend", "frontend", "docs", "infra"
}

// DefaultCorpus returns a hardcoded corpus of labeled routing examples (~30 entries).
// Synthetic: does not read files, does not depend on profiles_default.json.
func DefaultCorpus() []RoutingExample {
	return []RoutingExample{
		// go-backend
		{
			Title:       "fix nil pointer in Go handler",
			Description: "HTTP handler panics with nil pointer dereference in request processing",
			Capability:  "go-backend",
		},
		{
			Title:       "add HTTP endpoint for user registration",
			Description: "Implement POST /api/users endpoint with validation and database write",
			Capability:  "go-backend",
		},
		{
			Title:       "refactor database layer",
			Description: "Extract SQL queries into repository pattern, add transaction support",
			Capability:  "go-backend",
		},
		{
			Title:       "implement gRPC service",
			Description: "Add gRPC server with proto definitions and handler implementations in Go",
			Capability:  "go-backend",
		},
		{
			Title:       "add middleware for authentication",
			Description: "JWT validation middleware for Go HTTP router, with token expiry checks",
			Capability:  "go-backend",
		},
		{
			Title:       "optimize SQL query performance",
			Description: "Slow query in Go repository layer causing timeouts, needs index and rewrite",
			Capability:  "go-backend",
		},
		{
			Title:       "write unit tests for Go service layer",
			Description: "Add table-driven tests and mocks for the business logic package",
			Capability:  "go-backend",
		},
		{
			Title:       "migrate database schema with Go migrations",
			Description: "Add Goose migration files and update repository models for new schema",
			Capability:  "go-backend",
		},
		// frontend
		{
			Title:       "add React component for user profile",
			Description: "Create a new ProfileCard React component with TypeScript props",
			Capability:  "frontend",
		},
		{
			Title:       "fix CSS layout on mobile",
			Description: "Responsive layout breaks on small screens, flexbox alignment issue",
			Capability:  "frontend",
		},
		{
			Title:       "implement UX flow for onboarding",
			Description: "Multi-step onboarding wizard with form validation and progress indicator",
			Capability:  "frontend",
		},
		{
			Title:       "add dark mode toggle",
			Description: "Implement theme switcher using CSS variables and React context",
			Capability:  "frontend",
		},
		{
			Title:       "fix broken navigation menu",
			Description: "Dropdown nav menu does not close on outside click, state management bug",
			Capability:  "frontend",
		},
		{
			Title:       "integrate REST API calls in TypeScript",
			Description: "Fetch user data from backend API, handle loading and error states",
			Capability:  "frontend",
		},
		{
			Title:       "add form validation with React Hook Form",
			Description: "Client-side validation for login and registration forms",
			Capability:  "frontend",
		},
		{
			Title:       "optimize bundle size with code splitting",
			Description: "Reduce JavaScript bundle by lazy-loading heavy components",
			Capability:  "frontend",
		},
		// docs
		{
			Title:       "update README with installation steps",
			Description: "Improve getting-started section with clearer setup instructions",
			Capability:  "docs",
		},
		{
			Title:       "fix typo in API documentation",
			Description: "Several spelling mistakes in the public API docs need correction",
			Capability:  "docs",
		},
		{
			Title:       "add API reference for new endpoints",
			Description: "Document POST and GET endpoints with request/response examples",
			Capability:  "docs",
		},
		{
			Title:       "write architecture decision record",
			Description: "Document the reasoning behind choosing PostgreSQL over MongoDB",
			Capability:  "docs",
		},
		{
			Title:       "update changelog for release",
			Description: "Add entries for new features and bug fixes in CHANGELOG.md",
			Capability:  "docs",
		},
		{
			Title:       "add code comments to exported functions",
			Description: "Missing godoc comments on exported types and functions in public package",
			Capability:  "docs",
		},
		// infra
		{
			Title:       "fix CI pipeline failure",
			Description: "GitHub Actions workflow fails on test step, fix environment variables",
			Capability:  "infra",
		},
		{
			Title:       "update Dockerfile for production build",
			Description: "Multi-stage Docker build optimization, reduce final image size",
			Capability:  "infra",
		},
		{
			Title:       "configure nginx reverse proxy",
			Description: "Set up nginx config for load balancing and SSL termination",
			Capability:  "infra",
		},
		{
			Title:       "add Kubernetes deployment manifest",
			Description: "Create k8s Deployment and Service YAML for the Go backend service",
			Capability:  "infra",
		},
		{
			Title:       "set up Terraform for cloud resources",
			Description: "Provision S3 bucket and RDS instance with Terraform modules",
			Capability:  "infra",
		},
		{
			Title:       "fix flaky integration tests in CI",
			Description: "Race condition in test setup causes intermittent CI failures",
			Capability:  "infra",
		},
		{
			Title:       "configure monitoring and alerting",
			Description: "Set up Prometheus metrics and Grafana dashboards for service health",
			Capability:  "infra",
		},
		{
			Title:       "update helm chart for new service",
			Description: "Add helm chart values and templates for the new microservice deployment",
			Capability:  "infra",
		},
	}
}
