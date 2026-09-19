package explain

import (
	"testing"

	"github.com/kavix/why/internal/model"
)

func TestExplainSSHAuthenticationFailure(t *testing.T) {
	engine := New()

	diag := &model.Diagnostic{
		Target:   "ssh://deploy@prod.internal:22",
		Protocol: "ssh",
		Status:   "failed",
		Metadata: map[string]interface{}{
			"user": "deploy",
			"host": "prod.internal",
			"port": 22,
		},
		Failure: &model.Failure{
			Stage:  "authentication",
			Check:  "User Authentication",
			Reason: "Server rejected client authentication credentials",
			Evidence: map[string]interface{}{
				"username":         "deploy",
				"methods_offered":  []string{"publickey"},
				"methods_accepted": []string{"publickey", "keyboard-interactive"},
				"keys_attempted":   []string{"/home/user/.ssh/id_ed25519"},
			},
		},
	}

	engine.Explain(diag)

	if len(diag.Causes) == 0 {
		t.Fatalf("expected causal deductions, got 0")
	}

	cause := diag.Causes[0]
	if cause.Confidence != "high" {
		t.Errorf("expected high confidence, got %s", cause.Confidence)
	}

	if len(cause.Remediation) == 0 {
		t.Errorf("expected remediation steps, got 0")
	}
}

func TestExplainHTTP403(t *testing.T) {
	engine := New()

	diag := &model.Diagnostic{
		Target:   "https://api.example.com/admin",
		Protocol: "http",
		Status:   "failed",
		Metadata: map[string]interface{}{
			"url": "https://api.example.com/admin",
		},
		Failure: &model.Failure{
			Stage:  "http_response",
			Check:  "HTTP GET /admin",
			Reason: "Server returned error status 403 (Forbidden)",
			Evidence: map[string]interface{}{
				"status_code": 403,
			},
		},
	}

	engine.Explain(diag)

	if len(diag.Causes) == 0 {
		t.Fatalf("expected cause for HTTP 403, got 0")
	}

	if cause := diag.Causes[0]; cause.Confidence != "high" {
		t.Errorf("expected high confidence for 403, got %s", cause.Confidence)
	}
}

func TestExplainDNSNXDOMAIN(t *testing.T) {
	engine := New()

	diag := &model.Diagnostic{
		Target:   "nonexistent.test",
		Protocol: "dns",
		Status:   "failed",
		Failure: &model.Failure{
			Stage:  "dns_resolution",
			Check:  "A/AAAA Record Lookup",
			Reason: "Domain does not exist (NXDOMAIN)",
			Evidence: map[string]interface{}{
				"domain":    "nonexistent.test",
				"dns_rcode": "NXDOMAIN",
			},
		},
	}

	engine.Explain(diag)

	if len(diag.Causes) == 0 {
		t.Fatalf("expected cause for NXDOMAIN, got 0")
	}
}

func TestExplainTCPConnectionRefused(t *testing.T) {
	engine := New()

	diag := &model.Diagnostic{
		Target:   "127.0.0.1:9999",
		Protocol: "tcp",
		Status:   "failed",
		Failure: &model.Failure{
			Stage:  "tcp",
			Check:  "TCP Handshake :9999",
			Reason: "Connection refused",
			Evidence: map[string]interface{}{
				"tcp_state": "ECONNREFUSED",
			},
		},
	}

	engine.Explain(diag)

	if len(diag.Causes) == 0 {
		t.Fatalf("expected cause for TCP ECONNREFUSED, got 0")
	}
}
