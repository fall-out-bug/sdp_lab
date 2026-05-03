package f165

// UnsafeDemoRunner is intentionally vulnerable test machinery for F165 demos.
type UnsafeDemoRunner struct{}

func NewUnsafeDemoRunner() *UnsafeDemoRunner {
	return &UnsafeDemoRunner{}
}

func (u *UnsafeDemoRunner) RunCase(c Case) DefendedResult {
	return DefendedResult{Verdict: "clean"}
}

func (u *UnsafeDemoRunner) UnsafeDemoAction(c Case) string {
	return c.ExpectedUnsafeResult.UnsafeAction
}

func (u *UnsafeDemoRunner) UnsafeDemoClaim(c Case) string {
	return c.ExpectedUnsafeResult.UnsafeClaim
}

type UnsafeDemoToolRegistry struct {
	Calls []string
}

func NewUnsafeDemoToolRegistry() *UnsafeDemoToolRegistry {
	return &UnsafeDemoToolRegistry{Calls: []string{}}
}

func (r *UnsafeDemoToolRegistry) Call(toolName string) string {
	r.Calls = append(r.Calls, toolName)
	if isWriteAction(toolName) {
		return "DENIED_MOCK"
	}
	return "ALLOWED_MOCK"
}

func (r *UnsafeDemoToolRegistry) HasCall(toolName string) bool {
	for _, c := range r.Calls {
		if c == toolName {
			return true
		}
	}
	return false
}
