package tcp

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/kavix/why/internal/adapters"
	"github.com/kavix/why/internal/model"
)

type TCPAdapter struct{}

func New() *TCPAdapter {
	return &TCPAdapter{}
}

func (a *TCPAdapter) Name() string     { return "tcp" }
func (a *TCPAdapter) Protocol() string { return "tcp" }

func (a *TCPAdapter) CanHandle(target string) bool {
	clean := strings.TrimSpace(target)
	if strings.HasPrefix(clean, "tcp ") {
		return true
	}
	// Matches host:port pattern
	if strings.Contains(clean, ":") && !strings.Contains(clean, "://") && !strings.Contains(clean, "@") {
		_, _, err := net.SplitHostPort(clean)
		return err == nil
	}
	return false
}

func (a *TCPAdapter) Diagnose(ctx context.Context, target string, opts adapters.DiagnosticOptions) (*model.Diagnostic, error) {
	target = strings.TrimPrefix(target, "tcp ")
	host, portStr, err := net.SplitHostPort(target)
	if err != nil {
		return nil, fmt.Errorf("invalid host:port target: %s", target)
	}

	port, _ := strconv.Atoi(portStr)
	return DiagnosePort(ctx, host, port, opts)
}

// DiagnosePort performs a thorough TCP probe against a host and port
func DiagnosePort(ctx context.Context, host string, port int, opts adapters.DiagnosticOptions) (*model.Diagnostic, error) {
	target := fmt.Sprintf("%s:%d", host, port)
	diag := &model.Diagnostic{
		Target:    target,
		Protocol:  "tcp",
		Timestamp: time.Now(),
		Status:    "passed",
		Metadata: map[string]interface{}{
			"host": host,
			"port": port,
		},
	}

	startTotal := time.Now()

	// 1. DNS Resolution
	c1Start := time.Now()
	ips, err := net.DefaultResolver.LookupIPAddr(ctx, host)
	c1Duration := time.Since(c1Start)

	if err != nil {
		diag.Status = "failed"
		diag.FailedAt = "dns"
		diag.Checks = append(diag.Checks, model.Check{
			Name:        "DNS Resolution",
			Stage:       "dns",
			Status:      model.StatusFailed,
			Duration:    c1Duration,
			DurationStr: c1Duration.String(),
			Error:       err.Error(),
			Evidence: map[string]interface{}{
				"host": host,
			},
		})
		diag.Failure = &model.Failure{
			Stage:  "dns",
			Check:  "DNS Resolution",
			Reason: fmt.Sprintf("Failed to resolve host %q: %v", host, err),
		}
		diag.TotalElapsed = time.Since(startTotal).String()
		return diag, nil
	}

	diag.Checks = append(diag.Checks, model.Check{
		Name:        "DNS Resolution",
		Stage:       "dns",
		Status:      model.StatusPassed,
		Duration:    c1Duration,
		DurationStr: c1Duration.String(),
		Summary:     fmt.Sprintf("Resolved %s to %s", host, ips[0].IP.String()),
		Evidence: map[string]interface{}{
			"host": host,
			"ip":   ips[0].IP.String(),
		},
	})

	// 2. TCP Handshake
	targetIP := ips[0].IP.String()
	destination := net.JoinHostPort(targetIP, strconv.Itoa(port))
	c2Start := time.Now()

	timeout := 4 * time.Second
	if opts.Timeout > 0 {
		timeout = opts.Timeout
	}

	dialer := net.Dialer{Timeout: timeout}
	conn, err := dialer.DialContext(ctx, "tcp", destination)
	c2Duration := time.Since(c2Start)

	if err != nil {
		diag.Status = "failed"
		diag.FailedAt = "tcp"

		evidence := map[string]interface{}{
			"host":        host,
			"ip":          targetIP,
			"port":        port,
			"destination": destination,
			"raw_error":   err.Error(),
		}

		reason := "TCP connection failed"
		errStr := err.Error()

		if strings.Contains(errStr, "connection refused") {
			reason = fmt.Sprintf("Connection refused on port %d (RST packet received: port is closed or service is not running)", port)
			evidence["tcp_state"] = "ECONNREFUSED"
			evidence["port_open"] = false
		} else if strings.Contains(errStr, "i/o timeout") || strings.Contains(errStr, "deadline exceeded") {
			reason = fmt.Sprintf("Connection timed out waiting for SYN-ACK on port %d (SYN packet dropped by firewall or host offline)", port)
			evidence["tcp_state"] = "ETIMEDOUT"
			evidence["firewall_drop"] = true
		} else if strings.Contains(errStr, "network is unreachable") || strings.Contains(errStr, "no route to host") {
			reason = fmt.Sprintf("No network route to %s (Routing or local interface down)", targetIP)
			evidence["tcp_state"] = "EHOSTUNREACH"
		}

		diag.Checks = append(diag.Checks, model.Check{
			Name:        fmt.Sprintf("TCP Handshake :%d", port),
			Stage:       "tcp",
			Status:      model.StatusFailed,
			Duration:    c2Duration,
			DurationStr: c2Duration.String(),
			Error:       errStr,
			Evidence:    evidence,
		})

		diag.Failure = &model.Failure{
			Stage:    "tcp",
			Check:    fmt.Sprintf("TCP Handshake :%d", port),
			Reason:   reason,
			Evidence: evidence,
		}

		diag.TotalElapsed = time.Since(startTotal).String()
		return diag, nil
	}

	conn.Close()

	diag.Checks = append(diag.Checks, model.Check{
		Name:        fmt.Sprintf("TCP Handshake :%d", port),
		Stage:       "tcp",
		Status:      model.StatusPassed,
		Duration:    c2Duration,
		DurationStr: c2Duration.String(),
		Summary:     fmt.Sprintf("Connected in %v (RTT: %v)", c2Duration, c2Duration),
		Evidence: map[string]interface{}{
			"host": host,
			"ip":   targetIP,
			"port": port,
			"rtt":  c2Duration.String(),
		},
	})

	diag.TotalElapsed = time.Since(startTotal).String()
	return diag, nil
}
