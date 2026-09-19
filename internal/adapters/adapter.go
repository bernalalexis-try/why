package adapters

import (
	"context"
	"time"

	"github.com/kavix/why/internal/model"
)

// DiagnosticOptions encapsulates runtime configuration for probes.
type DiagnosticOptions struct {
	Deep      bool
	Verbose   bool
	Timeout   time.Duration
	Method    string
	Headers   []string
	KeyPath   string
	Port      int
	UserAgent string
}

// Adapter defines the contract for protocol-specific diagnostic plugins.
type Adapter interface {
	// Name returns the identifier of the adapter (e.g. "ssh", "http", "dns", "tls")
	Name() string

	// Protocol returns the canonical protocol scheme handled
	Protocol() string

	// CanHandle returns true if this adapter knows how to diagnose the given target or command
	CanHandle(commandOrTarget string) bool

	// Diagnose executes the multi-stage diagnostic checks and produces structured evidence
	Diagnose(ctx context.Context, target string, opts DiagnosticOptions) (*model.Diagnostic, error)
}
