package policies

import "pipeline-cli/core/policy"

// All returns the full registry of available policies keyed by their Name().
func All() map[string]policy.Policy {
	return map[string]policy.Policy{
		"no-hardcoded-secrets": &NoHardcodedSecrets{},
		"health-endpoint":      &HealthEndpoint{},
		"dependency-audit":     &DependencyAudit{},
		"feature-flags":        &FeatureFlags{},
		"api-versioning":       &ApiVersioning{},
		"logging-standard":     &LoggingStandard{},
	}
}
