package explain

import (
	"fmt"
	"strings"

	"github.com/kavix/why/internal/model"
)

// Engine performs causal reasoning on structured evidence from diagnostic checks.
type Engine struct{}

func New() *Engine {
	return &Engine{}
}

// Explain inspects a Diagnostic and populates its Causes list with causal deductions.
func (e *Engine) Explain(diag *model.Diagnostic) {
	if diag.Status != "failed" || diag.Failure == nil {
		return
	}

	switch diag.Failure.Stage {
	case "dns", "dns_resolution":
		e.explainDNS(diag)
	case "tcp":
		e.explainTCP(diag)
	case "tls", "tls_handshake", "certificate":
		e.explainTLS(diag)
	case "ssh_handshake", "ssh_kex", "authentication", "local_env":
		e.explainSSH(diag)
	case "url_parse", "http_request", "http_response":
		e.explainHTTP(diag)
	default:
		e.explainGeneric(diag)
	}
}

func (e *Engine) explainSSH(diag *model.Diagnostic) {
	ev := diag.Failure.Evidence
	if ev == nil {
		ev = make(map[string]interface{})
	}

	user, _ := diag.Metadata["user"].(string)
	host, _ := diag.Metadata["host"].(string)
	port, _ := diag.Metadata["port"].(int)
	if port == 0 {
		port = 22
	}

	switch diag.Failure.Stage {
	case "authentication":
		attemptedKeys, _ := ev["keys_attempted"].([]string)
		keysSummary := "none"
		if len(attemptedKeys) > 0 {
			keysSummary = strings.Join(attemptedKeys, ", ")
		}

		acceptedList, _ := ev["methods_accepted"].([]string)
		acceptedSummary := strings.Join(acceptedList, ", ")
		if acceptedSummary == "" {
			acceptedSummary = "publickey, password, keyboard-interactive"
		}

		c := model.Cause{
			Explanation: fmt.Sprintf("The SSH server rejected your credentials for user %q.", user),
			Confidence:  "high",
			Likely:      true,
			Evidence: map[string]interface{}{
				"client_offered": keysSummary,
				"server_accepts": acceptedSummary,
			},
			Remediation: []string{
				fmt.Sprintf("Install your public key on the server: ssh-copy-id -p %d %s@%s", port, user, host),
				"Verify your key permissions locally: chmod 600 ~/.ssh/id_* && chmod 700 ~/.ssh",
				"Ensure remote ~/.ssh/authorized_keys exists and has 0600 permissions",
				fmt.Sprintf("If connecting with a specific key, run: why ssh -i ~/.ssh/your_key %s@%s", user, host),
			},
		}
		diag.Causes = append(diag.Causes, c)

	case "ssh_handshake":
		c := model.Cause{
			Explanation: fmt.Sprintf("Port %d is open on %s, but the service did not respond with an SSH protocol banner.", port, host),
			Confidence:  "high",
			Likely:      true,
			Remediation: []string{
				fmt.Sprintf("Verify that an SSH daemon (sshd) is listening on port %d, not an HTTP or other proxy service.", port),
				fmt.Sprintf("Test raw connection: nc -zv %s %d", host, port),
			},
		}
		diag.Causes = append(diag.Causes, c)

	case "ssh_kex":
		c := model.Cause{
			Explanation: "SSH key exchange or cryptographic cipher algorithm negotiation failed.",
			Confidence:  "high",
			Likely:      true,
			Remediation: []string{
				"The remote server may require older or legacy ciphers/KEX algorithms disabled by default in modern OpenSSH.",
				"Check remote sshd logs: journalctl -u ssh or /var/log/auth.log",
			},
		}
		diag.Causes = append(diag.Causes, c)
	}
}

func (e *Engine) explainHTTP(diag *model.Diagnostic) {
	ev := diag.Failure.Evidence
	if ev == nil {
		ev = make(map[string]interface{})
	}

	statusCode, _ := ev["status_code"].(int)
	urlStr, _ := diag.Metadata["url"].(string)

	switch statusCode {
	case 401:
		diag.Causes = append(diag.Causes, model.Cause{
			Explanation: "The server requires authentication credentials (missing or invalid Authorization header).",
			Confidence:  "high",
			Likely:      true,
			Remediation: []string{
				"Provide valid credentials via Authorization: Bearer <token> or Basic auth header",
				"Check if your API token has expired or been revoked",
			},
		})

	case 403:
		diag.Causes = append(diag.Causes, model.Cause{
			Explanation: fmt.Sprintf("The server understood the request but refuses to authorize it (Forbidden). Server returned 403 for %s.", urlStr),
			Confidence:  "high",
			Likely:      true,
			Remediation: []string{
				"Your IP address may be blocked by a Web Application Firewall (Cloudflare, AWS WAF, or iptables)",
				"Your user/role lacks permission or scope to access this specific resource path",
				"The server may require a specific User-Agent, Origin, or Referer header",
			},
		})

	case 404:
		diag.Causes = append(diag.Causes, model.Cause{
			Explanation: "The requested endpoint or file path does not exist on the server (404 Not Found).",
			Confidence:  "high",
			Likely:      true,
			Remediation: []string{
				"Check for typos in the URL path or trailing slash requirements",
				"Verify routing rules in your reverse proxy (nginx, ingress, caddy)",
				"Confirm that the API version prefix (e.g. /v1, /api/v2) is correct",
			},
		})

	case 500:
		diag.Causes = append(diag.Causes, model.Cause{
			Explanation: "The server encountered an unhandled internal exception while processing the request (500 Internal Server Error).",
			Confidence:  "high",
			Likely:      true,
			Remediation: []string{
				"Check application backend logs / stack traces (e.g. journalctl, docker logs, Datadog/Sentry)",
				"Verify database connection pools, unhandled nil pointers, or fatal panics",
			},
		})

	case 502:
		diag.Causes = append(diag.Causes, model.Cause{
			Explanation: "Bad Gateway: The reverse proxy received an invalid response or connection refusal from the upstream application server.",
			Confidence:  "high",
			Likely:      true,
			Remediation: []string{
				"The backend process (e.g. Node, Python, Go, Java) is crashed, not running, or failed to start",
				"Verify that the reverse proxy is proxy_passing to the correct internal port / socket",
			},
		})

	case 503:
		diag.Causes = append(diag.Causes, model.Cause{
			Explanation: "Service Unavailable: The server is temporarily overloaded or down for scheduled maintenance.",
			Confidence:  "high",
			Likely:      true,
			Remediation: []string{
				"Check if server resources (CPU, RAM, max open file descriptors) are exhausted",
				"Inspect load balancer health check status to confirm healthy targets exist",
			},
		})

	case 504:
		diag.Causes = append(diag.Causes, model.Cause{
			Explanation: "Gateway Timeout: The upstream server failed to complete the request within the gateway proxy timeout window.",
			Confidence:  "high",
			Likely:      true,
			Remediation: []string{
				"A slow database query or downstream microservice deadlock is blocking the worker",
				"Increase proxy timeout (e.g. proxy_read_timeout in Nginx) if long-running queries are expected",
			},
		})

	default:
		diag.Causes = append(diag.Causes, model.Cause{
			Explanation: fmt.Sprintf("HTTP request returned failure status (%v).", diag.Failure.Reason),
			Confidence:  "medium",
			Likely:      true,
			Remediation: []string{
				"Inspect server response headers and error payload",
				"Re-run with --deep for complete protocol trace",
			},
		})
	}
}

func (e *Engine) explainTCP(diag *model.Diagnostic) {
	ev := diag.Failure.Evidence
	if ev == nil {
		ev = make(map[string]interface{})
	}

	state, _ := ev["tcp_state"].(string)
	if state == "" {
		state, _ = ev["state"].(string)
	}

	switch state {
	case "ECONNREFUSED":
		diag.Causes = append(diag.Causes, model.Cause{
			Explanation: "Connection was actively refused (TCP RST packet received). The target machine is online, but no process is listening on this port.",
			Confidence:  "high",
			Likely:      true,
			Remediation: []string{
				"Check if the service daemon is running: systemctl status <service>",
				"Verify the daemon is bound to 0.0.0.0 (all interfaces) rather than 127.0.0.1 (localhost only)",
				"Confirm the port number is correct",
			},
		})

	case "ETIMEDOUT":
		diag.Causes = append(diag.Causes, model.Cause{
			Explanation: "Connection timed out with no response (SYN packets dropped silently).",
			Confidence:  "high",
			Likely:      true,
			Remediation: []string{
				"A firewall or cloud security group (AWS SG, GCP Firewall, ufw) is dropping inbound traffic on this port",
				"The target host is offline, powered off, or experiencing network routing loss",
				"Check ISP or intermediate network ACLs",
			},
		})

	case "EHOSTUNREACH":
		diag.Causes = append(diag.Causes, model.Cause{
			Explanation: "Host or network is unreachable. No routing table entry exists to route packets to the destination.",
			Confidence:  "high",
			Likely:      true,
			Remediation: []string{
				"Check local network interfaces and default gateway: ip route or netstat -rn",
				"Verify VPN connection if targeting private corporate IP ranges",
			},
		})

	default:
		diag.Causes = append(diag.Causes, model.Cause{
			Explanation: "TCP socket connection could not be established.",
			Confidence:  "medium",
			Likely:      true,
			Remediation: []string{
				"Check host connectivity and routing",
			},
		})
	}
}

func (e *Engine) explainDNS(diag *model.Diagnostic) {
	ev := diag.Failure.Evidence
	if ev == nil {
		ev = make(map[string]interface{})
	}

	rcode, _ := ev["dns_rcode"].(string)
	domain, _ := ev["domain"].(string)

	switch rcode {
	case "NXDOMAIN":
		diag.Causes = append(diag.Causes, model.Cause{
			Explanation: fmt.Sprintf("Domain name %q does not exist in DNS (Non-Existent Domain / NXDOMAIN).", domain),
			Confidence:  "high",
			Likely:      true,
			Remediation: []string{
				"Check domain spelling for typos",
				"Verify domain registration and WHOIS status (domain may have expired or was never purchased)",
				"Confirm authoritative DNS nameserver records (NS) have propagated",
			},
		})

	case "TIMEOUT":
		diag.Causes = append(diag.Causes, model.Cause{
			Explanation: "DNS resolver did not respond within the timeout window.",
			Confidence:  "high",
			Likely:      true,
			Remediation: []string{
				"Your local DNS server configured in /etc/resolv.conf is unresponsive",
				"UDP port 53 traffic is blocked by local firewall or network operator",
				"Try querying public resolvers directly: dig @1.1.1.1 or why dns <domain> --deep",
			},
		})

	case "SERVFAIL":
		diag.Causes = append(diag.Causes, model.Cause{
			Explanation: "DNS Server Failure (SERVFAIL). The recursive resolver could not retrieve an answer from authoritative nameservers.",
			Confidence:  "high",
			Likely:      true,
			Remediation: []string{
				"DNSSEC validation failure (broken cryptographic signature chain on domain records)",
				"Authoritative nameservers are offline, unreachable, or misconfigured with lame delegations",
			},
		})

	default:
		diag.Causes = append(diag.Causes, model.Cause{
			Explanation: fmt.Sprintf("DNS lookup failed for %s.", domain),
			Confidence:  "medium",
			Likely:      true,
			Remediation: []string{
				"Check internet connection and DNS resolver settings",
			},
		})
	}
}

func (e *Engine) explainTLS(diag *model.Diagnostic) {
	ev := diag.Failure.Evidence
	if ev == nil {
		ev = make(map[string]interface{})
	}

	issue, _ := ev["tls_issue"].(string)
	if issue == "" {
		issue, _ = ev["issue"].(string)
	}

	switch issue {
	case "CERT_EXPIRED":
		diag.Causes = append(diag.Causes, model.Cause{
			Explanation: "The SSL/TLS certificate presented by the server has expired.",
			Confidence:  "high",
			Likely:      true,
			Remediation: []string{
				"Renew the TLS certificate via Let's Encrypt / Certbot / ACME provider",
				"Check if your server's clock/NTP is synchronized correctly",
			},
		})

	case "HOSTNAME_MISMATCH":
		diag.Causes = append(diag.Causes, model.Cause{
			Explanation: "Subject Alternative Name (SAN) mismatch: the domain requested does not match the certificate presented.",
			Confidence:  "high",
			Likely:      true,
			Remediation: []string{
				"Ensure the domain is included in the certificate's Subject Alternative Names (SAN)",
				"If using SNI with multiple virtual hosts, check reverse proxy configuration for default cert fallback",
			},
		})

	case "UNTRUSTED_ROOT_OR_SELF_SIGNED", "UNTRUSTED_AUTHORITY":
		diag.Causes = append(diag.Causes, model.Cause{
			Explanation: "Certificate is signed by an untrusted or private Certificate Authority (CA) or is self-signed.",
			Confidence:  "high",
			Likely:      true,
			Remediation: []string{
				"Install the private/internal root CA certificate into the system trust store",
				"Ensure the server is serving the full intermediate certificate chain (fullchain.pem)",
				"For testing only: bypass verification with curl -k or insecure flags",
			},
		})

	case "CIPHER_MISMATCH":
		diag.Causes = append(diag.Causes, model.Cause{
			Explanation: "No common cipher suite or TLS protocol version could be agreed upon between client and server.",
			Confidence:  "high",
			Likely:      true,
			Remediation: []string{
				"Enable modern TLS 1.2 or TLS 1.3 on the server",
				"Check cipher suite restrictions in web server ssl.conf",
			},
		})

	default:
		diag.Causes = append(diag.Causes, model.Cause{
			Explanation: "TLS negotiation or certificate verification failed.",
			Confidence:  "medium",
			Likely:      true,
			Remediation: []string{
				"Check SSL certificate expiration, chain, and domain matching with why tls <host>:443",
			},
		})
	}
}

func (e *Engine) explainGeneric(diag *model.Diagnostic) {
	diag.Causes = append(diag.Causes, model.Cause{
		Explanation: diag.Failure.Reason,
		Confidence:  "medium",
		Likely:      true,
		Remediation: []string{
			"Run with --deep or --verbose for detailed probe logs",
		},
	})
}
