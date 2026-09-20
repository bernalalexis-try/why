# Roadmap: Phase-by-Phase Plan until v1.0.0 Release

This document outlines the structured roadmap for `why`, detailing milestones, engineering deliverables, and testing criteria from the initial alpha through the General Availability (v1.0.0) release.

---

## Release Timeline Overview

```
┌──────────────┐     ┌──────────────┐     ┌──────────────┐     ┌──────────────┐     ┌──────────────┐
│   Phase 1    │────>│   Phase 2    │────>│   Phase 3    │────>│   Phase 4    │────>│   Phase 5    │
│    v0.1.0    │     │    v0.2.0    │     │    v0.3.0    │     │    v0.4.0    │     │    v1.0.0    │
│  Alpha Core  │     │ Dev Adapters │     │ Infra Adapts │     │  DAG & TUI   │     │      GA      │
└──────────────┘     └──────────────┘     └──────────────┘     └──────────────┘     └──────────────┘
```

---

## Phase 1: Core Foundation & Flagship Network Adapters (v0.1.0)
**Status:** Completed (Initial Release)  
**Objective:** Deliver an offline-first, deterministic failure diagnosis tool for networking and remote access with optional AI enhancement.

- [x] **Diagnostic Architecture**: Strict separation of structured facts (`model.Check.Evidence`) from human explanation.
- [x] **Core Network Adapters**:
  - **SSH**: Multi-stage probe (DNS, TCP :22, Banner exchange, KEX/ciphers, Host key, Auth rejection deduction).
  - **HTTP / curl**: Lifecycle probe (DNS, TCP, TLS, Request, Status codes 401/403/404/500/502/503/504).
  - **DNS**: Resolution, NXDOMAIN, SERVFAIL, timeout, public resolver comparison.
  - **TLS / SSL**: Certificate expiry, SAN hostname mismatch, self-signed/untrusted root CA.
  - **TCP**: Port reachability, ECONNREFUSED, firewall drop detection.
- [x] **Deterministic Cause Engine**: Rule-based causal inference with confidence ratings and remediation playbooks.
- [x] **JSON Failure Graph**: Machine-readable schema via `--json` for CI/CD pipeline gating.
- [x] **AI Integration Layer**: Multi-provider fallback (`--ai` supporting Ollama local, Gemini, OpenAI, Claude).
- [x] **Initial CI Workflow**: Matrix testing on Linux and macOS.

---

## Phase 2: Developer Tooling & Version Control Adapters (v0.2.0)
**Target:** Q4 2026  
**Objective:** Expand causal failure diagnosis into the daily developer workflow.

- [ ] **Git Adapter (`why git <push|pull|fetch|clone>`)**:
  - Detect non-fast-forward push rejections and remote branch divergence.
  - Diagnose SSH/HTTPS credential authentication failures and expired personal access tokens (PAT).
  - Identify failing pre-push and pre-commit hooks and report the exact hook script and exit code.
  - Check large file limits (GitHub 100MB file limit / Git LFS missing pointers).
- [ ] **SSH Configuration Parser**:
  - Parse `~/.ssh/config` for `Host`, `ProxyJump`, `Port`, and `IdentityFile` directives.
  - Diagnose SSH agent forwarding socket state (`SSH_AUTH_SOCK`).
- [ ] **Community & Contributor Onboarding**:
  - Triage and label `good first issue` tasks.
  - Publish contributor setup screencast and documentation.

---

## Phase 3: Containerization & Cloud-Native Infrastructure (v0.3.0)
**Target:** Q1 2027  
**Objective:** Diagnose local container failures and Kubernetes workload crashes.

- [ ] **Docker Adapter (`why docker <compose up|run|build>`)**:
  - Diagnose Docker daemon connection failures (socket permissions vs stopped daemon).
  - Port collision detection: match `bind: address already in use` to the conflicting process ID (PID) and process name.
  - Compose volume mount permission mismatches (host UID/GID vs container user).
  - Container crash detection (exit code 137 OOMKilled vs exit code 1 syntax/runtime errors).
- [ ] **Kubernetes Adapter (`why k8s <pod|deployment|svc>`)**:
  - Causal diagnosis of `CrashLoopBackOff` (extracting container stderr logs and termination exit codes).
  - Causal diagnosis of `ImagePullBackOff` (401 unauthorized registry vs 404 tag not found vs rate limit 429).
  - Failing liveness and readiness probe inspection (HTTP status vs TCP refusal).
  - Pod scheduling failures (`0/X nodes available: Insufficient memory/cpu/taints`).
- [ ] **Systemd Adapter (`why systemd <service>`)**:
  - Inspect failing systemd units, parse `journalctl -u` exit codes, and detect dependency cycle deadlocks.

---

## Phase 4: DAG Concurrency, Interactive TUI & Standard Protocol (v0.4.0)
**Target:** Q2 2027  
**Objective:** Enterprise-grade performance, interactive exploration, and standardized plugin protocol.

- [ ] **Concurrent DAG Execution Engine**:
  - Execute independent diagnostic probes concurrently via Directed Acyclic Graph (DAG).
  - Reduce total execution latency by 60%+ for multi-resolver, multi-port checks.
- [ ] **Streaming AI Responses**:
  - Stream tokens from local Ollama or LLM APIs directly to stdout in real-time.
- [ ] **Interactive Terminal UI (TUI)**:
  - Built with Bubbletea (`why --interactive` or `why --tui`).
  - Expandable node trees to inspect raw packet headers, TLS cert byte dumps, and timing breakdowns.
- [ ] **Standard Diagnostic Protocol (SDP)**:
  - JSON-RPC / IPC protocol allowing external language plugins (Python, Rust, Bash) to register as `why` adapters.

---

## Phase 5: General Availability & Ecosystem (v1.0.0 GA)
**Target:** Q3 2027  
**Objective:** Stable v1.0 API, distribution across all major package managers, and production enterprise hardening.

- [ ] **Global Package Distribution**:
  - Official Homebrew formula (`brew install why`).
  - Debian/Ubuntu `.deb`, Fedora/RHEL `.rpm`, and Arch Linux AUR packages.
  - Windows `winget` and `scoop` manifests.
- [ ] **Shell Auto-Completion**:
  - Native completion scripts for `bash`, `zsh`, `fish`, and `powershell`.
- [ ] **CI/CD Action & Gating Tool**:
  - Official GitHub Action (`kavix/why-action`) to automatically diagnose test failures and upload failure graphs.
- [ ] **Zero-Vulnerability Security Audit**:
  - Static security analysis (gosec), memory-safety review, and credential redaction verification.
- [ ] **Long-Term Support (LTS) & Semantic Versioning Guarantee**:
  - Freeze core diagnostic schema for `model.Diagnostic` v1.0.
