# Contributing to `why`

Thank you for your interest in contributing to `why`!

Our philosophy is: **"Don't just tell me that it failed. Tell me why."**
We strive to keep `why` fast, Unix-native, deterministic, and modular.

---

## Code of Conduct

Please be respectful, collaborative, and constructive in all discussions, issues, and pull requests.

---

## Development Setup

### Prerequisites
- [Go 1.22+](https://go.dev/dl/)
- `make`
- `git`

### Clone and Build
```bash
git clone https://github.com/kavix/why.git
cd why
make test
make build
```

The compiled binary will be placed at `./bin/why`.

---

## Adding a New Adapter

To add a new adapter (e.g. `git`, `docker`, `kubernetes`):
1. Read [docs/architecture.md](docs/architecture.md) and [docs/plugins.md](docs/plugins.md).
2. Create a new package under `internal/adapters/<name>`.
3. Implement `adapters.Adapter` (ensure you emit structured facts in `model.Check.Evidence`).
4. Update `internal/explain/explainer.go` with causal heuristics for that domain.
5. Register the new adapter in `internal/engine/engine.go`.
6. Add unit tests in `internal/explain/explainer_test.go` and within your adapter package.

---

## Submitting Pull Requests

1. Fork the repository and create your branch from `main`.
2. Ensure `go test -v -race ./...` passes without errors or race warnings.
3. Keep pull requests focused on a single change or feature.
4. Reference any relevant GitHub issues in your PR description.
