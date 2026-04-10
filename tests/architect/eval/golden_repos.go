package eval

// StandardGoldenRepos returns a curated list of well-known open-source repos
// with their expected architectural characteristics.
func StandardGoldenRepos() []GoldenRepo {
	return []GoldenRepo{
		// Go microservices
		{
			Name:           "go-microservices-demo",
			ExpectedStyles: []string{"microservices"},
			ExpectedLangs:  []string{"go"},
			Containers:     5,
			HasContracts:   true,
			Complexity:     "medium",
		},
		// Python monolith
		{
			Name:           "django-cms",
			ExpectedStyles: []string{"layered", "modular"},
			ExpectedLangs:  []string{"python"},
			Containers:     1,
			HasContracts:   false,
			Complexity:     "complex",
		},
		// Java microservices
		{
			Name:           "spring-petclinic-microservices",
			ExpectedStyles: []string{"microservices"},
			ExpectedLangs:  []string{"java"},
			Containers:     4,
			HasContracts:   true,
			Complexity:     "medium",
		},
		// TypeScript monorepo
		{
			Name:           "nx-monorepo",
			ExpectedStyles: []string{"monorepo_multi_service"},
			ExpectedLangs:  []string{"typescript"},
			Containers:     3,
			HasContracts:   false,
			Complexity:     "medium",
		},
		// Event-driven Go
		{
			Name:           "go-event-driven",
			ExpectedStyles: []string{"event_driven"},
			ExpectedLangs:  []string{"go"},
			Containers:     3,
			HasContracts:   true,
			Complexity:     "medium",
		},
		// Infra repo
		{
			Name:           "terraform-aws-modules",
			ExpectedStyles: []string{"infra_repo"},
			ExpectedLangs:  []string{"hcl"},
			Containers:     0,
			HasContracts:   false,
			Complexity:     "simple",
		},
		// Library
		{
			Name:           "go-stdlib",
			ExpectedStyles: []string{"library"},
			ExpectedLangs:  []string{"go"},
			Containers:     0,
			HasContracts:   false,
			Complexity:     "simple",
		},
	}
}
