package dns

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/kavix/why/internal/adapters"
	"github.com/kavix/why/internal/model"
)

type DNSAdapter struct{}

func New() *DNSAdapter {
	return &DNSAdapter{}
}

func (a *DNSAdapter) Name() string     { return "dns" }
func (a *DNSAdapter) Protocol() string { return "dns" }

func (a *DNSAdapter) CanHandle(target string) bool {
	clean := strings.TrimSpace(target)
	if strings.HasPrefix(clean, "dns ") {
		return true
	}
	// If it doesn't contain :// or @ or slashes, and has dots
	if !strings.Contains(clean, "://") && !strings.Contains(clean, "@") && !strings.Contains(clean, "/") && strings.Contains(clean, ".") {
		return true
	}
	return false
}

func (a *DNSAdapter) Diagnose(ctx context.Context, target string, opts adapters.DiagnosticOptions) (*model.Diagnostic, error) {
	domain := cleanDomain(target)
	diag := &model.Diagnostic{
		Target:    domain,
		Protocol:  "dns",
		Timestamp: time.Now(),
		Status:    "passed",
		Metadata: map[string]interface{}{
			"deep_mode": opts.Deep,
		},
	}

	startTotal := time.Now()

	// 1. Syntax / Domain name validity check
	c1Start := time.Now()
	if len(domain) == 0 || strings.Contains(domain, " ") {
		diag.Status = "failed"
		diag.FailedAt = "domain_syntax"
		diag.Checks = append(diag.Checks, model.Check{
			Name:        "Domain syntax",
			Stage:       "syntax",
			Status:      model.StatusFailed,
			Duration:    time.Since(c1Start),
			DurationStr: time.Since(c1Start).String(),
			Error:       "Invalid domain name syntax",
			Evidence:    map[string]interface{}{"domain": domain},
		})
		diag.Failure = &model.Failure{
			Stage:  "syntax",
			Check:  "Domain syntax",
			Reason: "Domain name contains illegal characters or is empty",
		}
		diag.TotalElapsed = time.Since(startTotal).String()
		return diag, nil
	}

	diag.Checks = append(diag.Checks, model.Check{
		Name:        "Domain syntax",
		Stage:       "syntax",
		Status:      model.StatusPassed,
		Duration:    time.Since(c1Start),
		DurationStr: time.Since(c1Start).String(),
		Evidence:    map[string]interface{}{"domain": domain},
	})

	// 2. Primary DNS Resolution (A & AAAA records)
	resolver := net.DefaultResolver
	c2Start := time.Now()
	timeoutCtx, cancel := context.WithTimeout(ctx, 4*time.Second)
	defer cancel()

	ips, err := resolver.LookupIPAddr(timeoutCtx, domain)
	c2Duration := time.Since(c2Start)

	if err != nil {
		diag.Status = "failed"
		diag.FailedAt = "dns_resolution"

		errStr := err.Error()
		evidence := map[string]interface{}{
			"domain": domain,
			"error":  errStr,
		}

		reason := "Could not resolve domain name"
		if strings.Contains(errStr, "no such host") || strings.Contains(errStr, "NXDOMAIN") {
			reason = "Domain does not exist (NXDOMAIN)"
			evidence["dns_rcode"] = "NXDOMAIN"
		} else if strings.Contains(errStr, "i/o timeout") || strings.Contains(errStr, "deadline exceeded") {
			reason = "DNS query timed out (Resolver unresponsive or port 53 UDP blocked)"
			evidence["dns_rcode"] = "TIMEOUT"
		} else if strings.Contains(errStr, "server failure") || strings.Contains(errStr, "SERVFAIL") {
			reason = "Nameserver failed to resolve query (SERVFAIL - possible DNSSEC failure)"
			evidence["dns_rcode"] = "SERVFAIL"
		}

		diag.Checks = append(diag.Checks, model.Check{
			Name:        "A/AAAA Record Lookup",
			Stage:       "resolution",
			Status:      model.StatusFailed,
			Duration:    c2Duration,
			DurationStr: c2Duration.String(),
			Error:       errStr,
			Evidence:    evidence,
		})

		diag.Failure = &model.Failure{
			Stage:    "dns_resolution",
			Check:    "A/AAAA Record Lookup",
			Reason:   reason,
			Evidence: evidence,
		}

		// If deep mode, check alternate public DNS resolvers to isolate local vs public DNS
		if opts.Deep {
			probeAlternateResolvers(ctx, domain, diag)
		}

		diag.TotalElapsed = time.Since(startTotal).String()
		return diag, nil
	}

	ipStrings := make([]string, 0, len(ips))
	ipv4s := []string{}
	ipv6s := []string{}
	for _, ip := range ips {
		ipStrings = append(ipStrings, ip.IP.String())
		if ip.IP.To4() != nil {
			ipv4s = append(ipv4s, ip.IP.String())
		} else {
			ipv6s = append(ipv6s, ip.IP.String())
		}
	}

	diag.Checks = append(diag.Checks, model.Check{
		Name:        "A/AAAA Record Lookup",
		Stage:       "resolution",
		Status:      model.StatusPassed,
		Duration:    c2Duration,
		DurationStr: c2Duration.String(),
		Summary:     fmt.Sprintf("Resolved %d IP(s)", len(ipStrings)),
		Evidence: map[string]interface{}{
			"records": ipStrings,
			"ipv4":    ipv4s,
			"ipv6":    ipv6s,
			"count":   len(ipStrings),
		},
	})

	// 3. CNAME lookup
	c3Start := time.Now()
	cname, cnameErr := resolver.LookupCNAME(ctx, domain)
	c3Duration := time.Since(c3Start)
	if cnameErr == nil && cname != "" && cname != domain+"." && cname != domain {
		diag.Checks = append(diag.Checks, model.Check{
			Name:        "CNAME Aliasing",
			Stage:       "cname",
			Status:      model.StatusPassed,
			Duration:    c3Duration,
			DurationStr: c3Duration.String(),
			Summary:     fmt.Sprintf("Points to %s", cname),
			Evidence: map[string]interface{}{
				"cname": cname,
			},
		})
	}

	// 4. In deep mode, check NS records and latency against 8.8.8.8 & 1.1.1.1
	if opts.Deep {
		probeAlternateResolvers(ctx, domain, diag)
	}

	diag.TotalElapsed = time.Since(startTotal).String()
	return diag, nil
}

func probeAlternateResolvers(ctx context.Context, domain string, diag *model.Diagnostic) {
	publicResolvers := []struct {
		name string
		addr string
	}{
		{"Cloudflare (1.1.1.1)", "1.1.1.1:53"},
		{"Google (8.8.8.8)", "8.8.8.8:53"},
	}

	for _, r := range publicResolvers {
		start := time.Now()
		customResolver := &net.Resolver{
			PreferGo: true,
			Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
				d := net.Dialer{Timeout: 2 * time.Second}
				return d.DialContext(ctx, "udp", r.addr)
			},
		}

		ips, err := customResolver.LookupIPAddr(ctx, domain)
		dur := time.Since(start)
		if err != nil {
			diag.Checks = append(diag.Checks, model.Check{
				Name:        fmt.Sprintf("Probe %s", r.name),
				Stage:       "public_dns_probe",
				Status:      model.StatusFailed,
				Duration:    dur,
				DurationStr: dur.String(),
				Error:       err.Error(),
				Evidence: map[string]interface{}{
					"resolver": r.name,
					"error":    err.Error(),
				},
			})
		} else {
			ipList := []string{}
			for _, ip := range ips {
				ipList = append(ipList, ip.IP.String())
			}
			diag.Checks = append(diag.Checks, model.Check{
				Name:        fmt.Sprintf("Probe %s", r.name),
				Stage:       "public_dns_probe",
				Status:      model.StatusPassed,
				Duration:    dur,
				DurationStr: dur.String(),
				Summary:     fmt.Sprintf("Reachable, resolved %d IP(s)", len(ipList)),
				Evidence: map[string]interface{}{
					"resolver": r.name,
					"ips":      ipList,
				},
			})
		}
	}
}

func cleanDomain(target string) string {
	target = strings.TrimSpace(target)
	target = strings.TrimPrefix(target, "dns ")
	target = strings.TrimPrefix(target, "https://")
	target = strings.TrimPrefix(target, "http://")
	if idx := strings.Index(target, "/"); idx != -1 {
		target = target[:idx]
	}
	if idx := strings.Index(target, ":"); idx != -1 {
		target = target[:idx]
	}
	return target
}
