# Architecture & Philosophy of `why`

> **"Don't just tell me that it failed. Tell me why."**

`why` is designed as a Unix-native troubleshooting framework. Unlike traditional diagnostic utilities that dump raw logs or run isolated checks, `why` constructs a **machine-readable failure graph** and performs **causal reasoning** to explain the exact stage and root cause of a failure.

---

## 1. Core Architecture

The foundational design principle of `why` is the **strict separation of diagnostics from explanation**:

1. **Diagnostic Adapters** run network and protocol probes and collect structured evidence (facts). Adapters **never** format human text.
2. **The Cause Engine** inspects structured facts across the probe graph, correlates dependencies, and generates causal deductions and remediation steps.
3. **The Output Formatter** renders human-friendly ANSI terminal trees or machine-readable JSON.
4. **Optional AI Layer** synthesizes complicated failure graphs into enriched contextual playbooks when requested.

```mermaid
flowchart TD
    CLI["why CLI Command Parser"] --> Engine["Diagnostic Engine"]
    
    subgraph Probes ["Pluggable Diagnostic Adapters"]
        DNS["DNS Probe Adapter"]
        TCP["TCP Probe Adapter"]
        TLS["TLS Probe Adapter"]
        SSH["SSH Adapter"]
        HTTP["HTTP / curl Adapter"]
        Ext["Future Adapters (Git, Docker, K8s)"]
    end

    Engine --> Probes
    Probes --> Facts["Structured Evidence & Failure Graph"]

    subgraph Reasoning ["Causal Reasoning Layer"]
        Facts --> CauseEngine["Deterministic Cause Engine"]
        Facts -.->|"Flag: --ai"| AIEngine["AI Causal Analysis (Ollama/Gemini/OpenAI/Claude)"]
    end

    CauseEngine --> Formatter["Unified Output Formatter"]
    AIEngine --> Formatter
    Formatter --> Terminal["Human Terminal Output (✓, ✗, Cause Tree)"]
    Formatter --> JSON["Machine-Readable JSON Output (--json)"]
```

---

## 2. Structured Evidence Model

Every probe executed by an adapter emits structured evidence conforming to a unified data contract:

```go
type Diagnostic struct {
    Target       string                 `json:"target"`
    Protocol     string                 `json:"protocol"`
    Timestamp    time.Time              `json:"timestamp"`
    Status       string                 `json:"status"` // "passed", "failed"
    FailedAt     string                 `json:"failed_at,omitempty"`
    TotalElapsed string                 `json:"total_elapsed"`
    Checks       []Check                `json:"checks"`
    Failure      *Failure               `json:"failure,omitempty"`
    Causes       []Cause                `json:"causes,omitempty"`
    AIAnalysis   string                 `json:"ai_analysis,omitempty"`
}
```

### Check
Represents an individual step or probe in the diagnostic pipeline:
- `Name`: Human-readable probe title (e.g. `TCP Connect :22`)
- `Stage`: Canonical stage identifier (`dns`, `tcp`, `tls`, `ssh_handshake`, `authentication`)
- `Status`: `passed`, `failed`, `warning`, `skipped`
- `Evidence`: Key-value map of raw facts (`rtt`, `ip`, `cipher_suite`, `methods_accepted`)
- `Duration`: Monotonic probe execution time

### Failure & Causes
When a check fails:
- The adapter populates `Failure` with the failure point and raw evidence.
- The `CauseEngine` iterates over the evidence and generates `Cause` hypotheses:
  - `Explanation`: Direct, concise explanation of the root cause.
  - `Confidence`: `high`, `medium`, `low`.
  - `Remediation`: Copy-pasteable terminal commands to resolve the issue.

---

## 3. Protocol Pipelines

### SSH Pipeline
```mermaid
sequenceDiagram
    participant User as why CLI
    participant DNS as DNS Resolver
    participant TCP as Remote Host (Port 22)
    participant KEX as SSH Key Exchange
    participant Auth as SSH Auth Engine

    User->>DNS: 1. Resolve Hostname
    DNS-->>User: IP Address (A/AAAA)
    User->>TCP: 2. SYN Packet
    TCP-->>User: SYN-ACK (RTT measured)
    User->>TCP: 3. Banner Exchange (SSH-2.0-...)
    TCP-->>User: Server Identification Banner
    User->>KEX: 4. KEXInit & Host Key Verification
    KEX-->>User: Host Key Fingerprint (SHA256)
    User->>Auth: 5. Userauth Request (publickey)
    Auth-->>User: Rejection / Accepted Methods
```

If authentication fails, `why` extracts the offered keys vs accepted methods and explains:
- Whether the remote server rejected the key
- If `~/.ssh/authorized_keys` permissions are wrong
- If sshd configuration restricts the user

### HTTP / curl Pipeline
```mermaid
sequenceDiagram
    participant User as why CLI
    participant DNS as DNS Resolver
    participant TCP as TCP Handshake (80/443)
    participant TLS as TLS Handshake
    participant HTTP as HTTP Request

    User->>DNS: Resolve Hostname
    DNS-->>User: IP Address
    User->>TCP: TCP Connect
    TCP-->>User: Connected
    User->>TLS: ClientHello (SNI)
    TLS-->>User: ServerHello + Certificate Chain
    User->>HTTP: GET /path (Headers)
    HTTP-->>User: Status Code (401, 403, 404, 500, 502)
```

---

## 4. Deterministic Core + AI Enhancement

`why` is strictly **offline-first and deterministic**:
- 100% of standard causal deductions work without any internet access, API keys, or LLMs.
- When `--ai` is enabled, the structured evidence graph is sent to local Ollama (`llama3.2`) or cloud providers (Gemini, OpenAI, Claude) to provide extended contextual diagnosis.
