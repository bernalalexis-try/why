# Writing Custom Diagnostic Adapters

`why` is designed around a pluggable diagnostic adapter architecture. Any contributor can add support for new tools (e.g. `why git`, `why docker`, `why k8s`, `why postgres`) by implementing the `Adapter` interface.

---

## 1. The `Adapter` Interface

Every adapter implements:

```go
package adapters

import (
    "context"
    "github.com/kavix/why/internal/model"
)

type Adapter interface {
    // Name returns the identifier of the adapter (e.g. "docker", "git")
    Name() string

    // Protocol returns the protocol or command scheme handled
    Protocol() string

    // CanHandle returns true if this adapter recognizes the target string
    CanHandle(target string) bool

    // Diagnose executes probe stages and returns structured evidence
    Diagnose(ctx context.Context, target string, opts DiagnosticOptions) (*model.Diagnostic, error)
}
```

---

## 2. Rule: Separate Diagnostics from Explanation

When writing an adapter:
1. **DO NOT** format or print human strings like "You need to fix your password".
2. **DO** record raw facts in `model.Check` and `model.Failure.Evidence`.

### Example: Implementing a Docker Adapter

```go
package docker

import (
    "context"
    "time"
    "github.com/kavix/why/internal/adapters"
    "github.com/kavix/why/internal/model"
)

type DockerAdapter struct{}

func (d *DockerAdapter) Name() string     { return "docker" }
func (d *DockerAdapter) Protocol() string { return "docker" }

func (d *DockerAdapter) CanHandle(target string) bool {
    return strings.HasPrefix(target, "docker ")
}

func (d *DockerAdapter) Diagnose(ctx context.Context, target string, opts adapters.DiagnosticOptions) (*model.Diagnostic, error) {
    diag := &model.Diagnostic{
        Target:   target,
        Protocol: "docker",
        Status:   "passed",
    }

    // 1. Socket check
    if _, err := os.Stat("/var/run/docker.sock"); err != nil {
        diag.Status = "failed"
        diag.FailedAt = "docker_socket"
        diag.Checks = append(diag.Checks, model.Check{
            Name:   "Docker Daemon Socket",
            Stage:  "socket",
            Status: model.StatusFailed,
            Error:  "Cannot connect to the Docker daemon socket",
            Evidence: map[string]interface{}{
                "socket_path": "/var/run/docker.sock",
                "exists":      false,
            },
        })
        diag.Failure = &model.Failure{
            Stage:  "docker_socket",
            Check:  "Docker Daemon Socket",
            Reason: "Docker daemon is not running or current user lacks docker group permission",
            Evidence: map[string]interface{}{
                "socket": "/var/run/docker.sock",
            },
        }
        return diag, nil
    }

    return diag, nil
}
```

---

## 3. Registering the Adapter

Register your new adapter in `internal/engine/engine.go`:

```go
func New() *Engine {
    e := &Engine{
        adapters: []adapters.Adapter{
            ssh.New(),
            http.New(),
            dns.New(),
            tls.New(),
            tcp.New(),
            docker.New(), // <-- registered here
        },
        ...
    }
    return e
}
```
