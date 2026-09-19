package model

import (
	"encoding/json"
	"time"
)

// CheckStatus defines the status of an individual diagnostic check.
type CheckStatus string

const (
	StatusPassed  CheckStatus = "passed"
	StatusFailed  CheckStatus = "failed"
	StatusSkipped CheckStatus = "skipped"
	StatusWarning CheckStatus = "warning"
)

// Check represents an individual step or probe in a diagnostic pipeline.
type Check struct {
	Name         string                 `json:"name"`
	Stage        string                 `json:"stage"`
	Status       CheckStatus            `json:"status"`
	Summary      string                 `json:"summary,omitempty"`
	Evidence     map[string]interface{} `json:"evidence,omitempty"`
	Duration     time.Duration          `json:"duration_ns"`
	DurationStr  string                 `json:"duration"`
	Dependencies []string               `json:"dependencies,omitempty"`
	Error        string                 `json:"error,omitempty"`
}

// Failure contains the root point of failure when a pipeline halts.
type Failure struct {
	Stage    string                 `json:"stage"`
	Check    string                 `json:"check"`
	Reason   string                 `json:"reason"`
	Evidence map[string]interface{} `json:"evidence,omitempty"`
}

// Cause represents a causal hypothesis explaining why a failure occurred.
type Cause struct {
	Explanation string                 `json:"explanation"`
	Confidence  string                 `json:"confidence"` // "high", "medium", "low"
	Likely      bool                   `json:"likely"`
	Evidence    map[string]interface{} `json:"evidence,omitempty"`
	Remediation []string               `json:"remediation,omitempty"`
}

// Diagnostic represents the complete causal failure graph and diagnostic results.
type Diagnostic struct {
	Target       string    `json:"target"`
	Protocol     string    `json:"protocol"`
	Timestamp    time.Time `json:"timestamp"`
	Status       string    `json:"status"` // "passed", "failed", "warning"
	FailedAt     string    `json:"failed_at,omitempty"`
	TotalElapsed string    `json:"total_elapsed"`
	Checks       []Check   `json:"checks"`
	Failure      *Failure  `json:"failure,omitempty"`
	Causes       []Cause   `json:"causes,omitempty"`
	AIAnalysis   string    `json:"ai_analysis,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// ToJSON returns a formatted JSON representation of the diagnostic result.
func (d *Diagnostic) ToJSON() (string, error) {
	b, err := json.MarshalIndent(d, "", "  ")
	if err != nil {
		return "", err
	}
	return string(b), nil
}
