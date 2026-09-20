# Implementation Plan & Engineering Specifications

This document defines the technical design specifications, engineering standards, and testing strategy for implementing and extending `why`.

---

## 1. Architectural Principles

### 1.1 The Causal Invariance Principle
Diagnostics must never jump straight to human explanations. The diagnostic adapter must faithfully record observable protocol state transitions. The explanation engine operates on the complete observable graph.

$$\text{Adapter}(\text{Target}) \longrightarrow \text{Evidence Graph} \xrightarrow{\text{Causal Engine}} \text{Causes \& Remediations}$$

### 1.2 Isolation & Zero Side-Effects
Probes initiated by `why` must be strictly read-only and non-destructive:
- HTTP adapter issues `GET` or `HEAD` unless explicitly configured with `-X`.
- SSH adapter never attempts state alteration or command execution on remote hosts; it halts immediately after authentication negotiation or banner verification.
- Probes respect timeouts (default 10s) with active cancellation contexts.

### 1.3 Offline-First & Privacy First
- `why` must never transmit user data, hostname queries, or environment variables to external servers unless the user explicitly specifies the `--ai` flag.
- When `--ai` is invoked with cloud providers, private keys, authorization tokens, and credentials in headers are strictly redacted before dispatch.

---

## 2. Adapter Specification Checklist

When implementing any adapter:

| Requirement | Description | Verification Method |
|---|---|---|
| **Interface Adherence** | Implements `adapters.Adapter` interface | Compile-time interface assertion: `var _ adapters.Adapter = (*Adapter)(nil)` |
| **Monotonic Timers** | Uses `time.Since` for probe durations | `Check.Duration` populated in nanoseconds |
| **Evidence Map** | All numeric metrics and protocol flags placed in `Evidence` | Checked in unit tests |
| **Context Respect** | All network dials accept `context.Context` | Cancels gracefully on `SIGINT` |
| **Error Handling** | Does not panic on nil pointers or closed sockets | Fuzz testing & unit tests |

---

## 3. Phase Implementation Matrix

### Phase 2: Git Adapter Architecture
- **Target Syntax**: `why git push [remote] [branch]`, `why git fetch`, `why git clone <url>`
- **Probe Stages**:
  1. Local Git State: check `.git` existence, current branch, upstream tracking configuration (`git rev-parse --abbrev-ref --symbolic-full-name @{u}`).
  2. Remote Transport: resolve remote URL (SSH vs HTTPS).
  3. Credential Probe: check git credential helper or SSH agent.
  4. Remote Ref Comparison: compare local HEAD SHA vs remote ref SHA.
  5. Hook Verification: inspect `.git/hooks/pre-push` execution permissions and exit codes.

### Phase 3: Docker Adapter Architecture
- **Target Syntax**: `why docker compose up`, `why docker run <image>`
- **Probe Stages**:
  1. Daemon Socket: probe `/var/run/docker.sock` or `DOCKER_HOST` TCP socket.
  2. Compose File Syntax: validate `docker-compose.yml` parsing and schema version.
  3. Port Conflict Probe: for every declared port mapping `host_port:container_port`, probe local TCP bind to check if already occupied.
  4. Volume Path Check: check if local directory mounts exist and verify read/write permissions for container UID.
  5. Container State Inspection: read exit codes and termination reason (`OOMKilled: true`).

### Phase 4: Kubernetes Adapter Architecture
- **Target Syntax**: `why k8s pod/<name>`, `why k8s deployment/<name>`
- **Probe Stages**:
  1. Kubeconfig & API Server: probe cluster endpoint reachability and token expiration.
  2. Resource Status: fetch Pod `status.phase`, `status.conditions`, and `status.containerStatuses`.
  3. Event Correlation: inspect recent `kubectl get events` associated with the target resource.
  4. Root Cause Deduction:
     - `BackOff`: check `lastTerminationState.exitCode` and fetch last 50 lines of logs.
     - `ErrImagePull`: check image repository DNS and secret configuration.
     - `Pending`: inspect node affinity, resource requests vs node capacity.

---

## 4. Test Strategy & Quality Gates

### 4.1 Unit Testing
- Every adapter must have mock server tests covering:
  - Happy path (clean pass)
  - Connection refused
  - Host unreachable
  - DNS resolution failure
  - Protocol-specific failure (403, cert expired, SSH key rejected)

### 4.2 Race Detection & Linting
- All PRs must pass:
  ```bash
  go test -v -race ./...
  go vet ./...
  ```

### 4.3 Determinism Testing
- Given the same mocked evidence graph, `CauseEngine.Explain` must produce the exact same `Causes` and `Remediation` across runs.
