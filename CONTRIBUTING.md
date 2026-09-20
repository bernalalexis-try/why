# Contributing to `why`

Thank you for your interest in contributing to **`why`**!

Our philosophy is simple:
> **"Don't just tell me that it failed. Tell me why."**

We strive to keep `why` fast, Unix-native, deterministic, and modular. This document will guide you through setting up your development environment, finding an issue, and submitting your contributions.

---

## Where to Start: `good first issue`

If you are new to the project or looking for an easy place to get started, check out our curated list of beginner-friendly tasks:

👉 **[Browse `good first issue` on GitHub](https://github.com/kavix/why/issues?q=is%3Aissue+is%3Aopen+label%3A%22good+first+issue%22)**

Before starting work:
1. Comment on the issue to let others know you are working on it so efforts aren't duplicated.
2. If you have questions about the design, ask in the issue thread!

---

## Roadmap & Implementation Plan

- **[ROADMAP.md](ROADMAP.md)**: Outlines the phase-by-phase deliverables from Alpha (v0.1.0) to General Availability (v1.0.0).
- **[IMPLEMENTATION_PLAN.md](IMPLEMENTATION_PLAN.md)**: Detailed engineering specifications, protocol contracts, test strategy, and privacy rules.
- **[docs/architecture.md](docs/architecture.md)**: Full explanation of the evidence graph, sequence diagrams, and causal reasoning layer.
- **[docs/plugins.md](docs/plugins.md)**: Step-by-step tutorial on implementing a new diagnostic adapter.

---

## Development Setup

### Prerequisites
- [Go 1.22+](https://go.dev/dl/)
- `make`
- `git`

### Quick Start
```bash
# 1. Fork and clone the repository
git clone https://github.com/<your-username>/why.git
cd why

# 2. Run unit tests
make test

# 3. Build the binary
make build

# 4. Run your local build
./bin/why dns google.com
./bin/why ssh nonexistentuser@github.com
```

---

## Code Contribution Workflow

### 1. Create a Branch
```bash
git checkout -b feat/your-feature-name
# or
git checkout -b fix/issue-description
```

### 2. Follow Coding Conventions
- **The Causal Invariance Principle**: Diagnostic adapters **never** format human presentation text. They emit structured facts in `model.Check.Evidence`. The `explain.Engine` converts facts into human causes and playbooks.
- **Error Handling**: Do not panic. Propagate errors gracefully and record failure reasons into `model.Failure`.
- **Concurrency & Races**: Always verify your code passes the Go race detector:
  ```bash
  go test -v -race ./...
  ```
- **Linting & Formatting**:
  ```bash
  go vet ./...
  go fmt ./...
  ```

### 3. Commit Guidelines (Conventional Commits)
We follow the [Conventional Commits](https://www.conventionalcommits.org/) format:

- `feat:` A new feature or diagnostic adapter (e.g. `feat(adapter): add git push diagnosis`)
- `fix:` A bug fix (e.g. `fix(tls): handle self-signed SAN wildcards correctly`)
- `docs:` Documentation improvements (e.g. `docs: update ROADMAP.md with phase 3 details`)
- `test:` Adding or refactoring tests (e.g. `test(ssh): add mock server auth failure fixtures`)
- `refactor:` Code restructuring without functional change

### 4. Submit a Pull Request
- Push your branch to your fork on GitHub.
- Open a Pull Request targeting the `main` branch.
- Fill out the [Pull Request Template](.github/PULL_REQUEST_TEMPLATE.md) and link the issue it resolves (e.g. `Fixes #11`).
- Ensure CI workflows pass. A maintainer will review your PR promptly!

---

## Community & Code of Conduct

All contributors are expected to uphold our [Code of Conduct](CODE_OF_CONDUCT.md). Please be kind, constructive, and respectful.
