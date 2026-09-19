package engine

import (
	"context"
	"fmt"
	"strings"

	"github.com/kavix/why/internal/adapters"
	"github.com/kavix/why/internal/adapters/dns"
	"github.com/kavix/why/internal/adapters/http"
	"github.com/kavix/why/internal/adapters/ssh"
	"github.com/kavix/why/internal/adapters/tcp"
	"github.com/kavix/why/internal/adapters/tls"
	"github.com/kavix/why/internal/ai"
	"github.com/kavix/why/internal/explain"
	"github.com/kavix/why/internal/model"
)

// Options configuration for engine run
type RunOptions struct {
	adapters.DiagnosticOptions
	Explain bool
	AI      bool
	JSON    bool
}

// Engine orchestrates adapters, explanation, and AI layers.
type Engine struct {
	adapters []adapters.Adapter
	explain  *explain.Engine
	ai       *ai.Engine
}

func New() *Engine {
	e := &Engine{
		adapters: []adapters.Adapter{
			ssh.New(),
			http.New(),
			dns.New(),
			tls.New(),
			tcp.New(),
		},
		explain: explain.New(),
		ai:      ai.New(),
	}
	return e
}

// RegisterAdapter adds a custom diagnostic adapter plugin
func (e *Engine) RegisterAdapter(a adapters.Adapter) {
	e.adapters = append(e.adapters, a)
}

// Diagnose evaluates a target using the most appropriate registered adapter.
func (e *Engine) Diagnose(ctx context.Context, target string, opts RunOptions) (*model.Diagnostic, error) {
	adapter := e.findAdapter(target)
	if adapter == nil {
		return nil, fmt.Errorf("no diagnostic adapter found for target: %q\nTry specifying protocol, e.g. 'why ssh %s' or 'why curl %s'", target, target, target)
	}

	// 1. Run multi-stage diagnostic probes
	diag, err := adapter.Diagnose(ctx, target, opts.DiagnosticOptions)
	if err != nil {
		return nil, err
	}

	// 2. Perform causal reasoning via explanation engine
	e.explain.Explain(diag)

	// 3. Optional AI analysis if requested
	if opts.AI {
		analysis, aiErr := e.ai.Analyze(ctx, diag)
		if aiErr != nil {
			diag.AIAnalysis = fmt.Sprintf("AI Explanation Unavailable:\n%v", aiErr)
		} else {
			diag.AIAnalysis = analysis
		}
	}

	return diag, nil
}

func (e *Engine) findAdapter(target string) adapters.Adapter {
	clean := strings.TrimSpace(target)

	// Check explicit prefixes first
	switch {
	case strings.HasPrefix(clean, "ssh ") || strings.HasPrefix(clean, "ssh://"):
		for _, a := range e.adapters {
			if a.Name() == "ssh" {
				return a
			}
		}
	case strings.HasPrefix(clean, "curl ") || strings.HasPrefix(clean, "http ") || strings.HasPrefix(clean, "https ") ||
		strings.HasPrefix(clean, "http://") || strings.HasPrefix(clean, "https://"):
		for _, a := range e.adapters {
			if a.Name() == "http" {
				return a
			}
		}
	case strings.HasPrefix(clean, "tls ") || strings.HasPrefix(clean, "ssl "):
		for _, a := range e.adapters {
			if a.Name() == "tls" {
				return a
			}
		}
	case strings.HasPrefix(clean, "dns "):
		for _, a := range e.adapters {
			if a.Name() == "dns" {
				return a
			}
		}
	case strings.HasPrefix(clean, "tcp "):
		for _, a := range e.adapters {
			if a.Name() == "tcp" {
				return a
			}
		}
	}

	// Fallback to CanHandle matchers
	for _, a := range e.adapters {
		if a.CanHandle(clean) {
			return a
		}
	}

	return nil
}
