package policy

// Policy defines the interface every policy check must implement.
type Policy interface {
	Name()        string
	DisplayName() string
	Description() string
	Category()    string
	Severity()    string // "error" or "warning"
	Run(projectPath string, config map[string]map[string]interface{}) PolicyResult
}

// PolicyResult holds the outcome of a single policy check.
type PolicyResult struct {
	PolicyName string
	Category   string
	Passed     bool
	Severity   string
	Message    string
	Findings   []Finding
}

// Finding represents a specific code location that violated a policy.
type Finding struct {
	File   string
	Line   int
	Detail string
}