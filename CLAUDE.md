# Kubeflow Coach

Educational simplified clone of [Kubeflow Trainer v2](https://github.com/kubeflow/trainer) for learning Kubernetes operator development.

## Project Purpose

"Coach" is a play on "Trainer". This project replicates core Kubeflow Trainer v2 concepts in a minimal way:
- **TrainJob** — user-facing CRD to run training jobs (references a runtime)
- **ClusterTrainingRuntime** — cluster-scoped blueprint defining infrastructure (image, resources, parallelism)
- **Controller** — reconcile loop that merges TrainJob + Runtime → creates batch/v1 Jobs

Upstream trainer uses JobSet; we use plain k8s Jobs for simplicity.

## Architecture Decisions

- **Scaffolded with kubebuilder v4** — standard k8s operator layout
- **API group**: `coach.kubeflow.io/v1alpha1`
- **Workload**: plain `batch/v1 Job` (not JobSet)
- **Scope**: ultra-minimal MVP — no initializers, no RuntimePatches, no webhooks

## Project Structure

```
kubeflow-coach/
├── cmd/                    # Entrypoint — boots the operator Manager, registers controllers
│   └── main.go
├── api/                    # CRD type definitions (the YAML contract)
│   └── v1alpha1/
│       ├── trainjob_types.go                  # TrainJob spec/status structs
│       ├── clustertrainingruntime_types.go     # Runtime spec structs
│       ├── groupversion_info.go               # Scheme registration (maps Go types ↔ k8s resources)
│       └── zz_generated.deepcopy.go           # Auto-generated, never edit (make generate)
├── internal/               # Business logic (internal/ = not importable by other modules)
│   └── controller/
│       ├── trainjob_controller.go             # Reconcile loop — the main logic
│       └── trainjob_controller_test.go        # Integration tests with envtest
├── config/                 # Generated k8s manifests for deployment
│   ├── crd/                # CRD YAML (auto-generated from Go types by make manifests)
│   ├── rbac/               # RBAC rules (auto-generated from marker comments in controller)
│   ├── manager/            # Deployment YAML for the operator pod
│   ├── default/            # Kustomize overlay combining all config/ into one deployable unit
│   └── samples/            # Example TrainJob and Runtime YAML for users
├── hack/                   # Developer scripts (code-gen, linting, CI helpers)
├── Taskfile.yml            # Build system (go-task): task generate, task manifests, task build, task test
├── Makefile                # Kubebuilder-generated Makefile (kept for reference, use Taskfile.yml instead)
├── Dockerfile              # Container image for the operator
├── PROJECT                 # Kubebuilder metadata (tracks scaffolded APIs/controllers)
├── go.mod
└── go.sum
```

### How the pieces connect

```
User writes YAML ──→ api/ types define the schema
                          │
kubectl apply ────→ Kubernetes stores it (using CRDs from config/crd/)
                          │
                     controller-runtime detects the change
                          │
                     internal/controller/ Reconcile() runs
                          │
                     Creates/updates batch/v1 Jobs
                          │
                     Updates TrainJob status (back to api/ types)
```

### Why directories are named this way

- **cmd/** — Go convention for entrypoints (same as kubectl, kubelet)
- **api/** — kubebuilder v4 convention for CRD types (older projects use pkg/apis/)
- **internal/** — Go's package visibility rule: code here cannot be imported externally. Controller logic is an implementation detail, not a public API
- **config/** — k8s manifests convention
- **hack/** — Kubernetes ecosystem convention meaning "development utilities"

## Key Concepts Mapping (Coach → Upstream Trainer)

| Coach (simplified) | Upstream Trainer | Notes |
|---|---|---|
| `TrainJob` | `TrainJob` | Same concept, simplified spec |
| `ClusterTrainingRuntime` | `ClusterTrainingRuntime` + `TrainingRuntime` | We skip namespace-scoped runtime |
| `batch/v1 Job` | `JobSet` | We use plain Jobs instead of JobSet |
| `TrainJobSpec.Image/Command/Args/Env` | `Trainer` + `RuntimePatches` | We inline overrides instead of patch API |
| `TrainJobStatus.Conditions` | `TrainJobStatus.Conditions` | Same pattern |

## Development Workflow

This project uses [Taskfile](https://taskfile.dev/) instead of Make. Install with `brew install go-task`.

```bash
# Prerequisites: Go 1.26+, kubebuilder CLI, kubectl + test cluster (kind/minikube), go-task

# Code generation (run after modifying api/ types)
task generate          # Regenerate deepcopy methods (zz_generated.deepcopy.go)
task manifests         # Regenerate CRD YAML + RBAC from marker comments

# Build & test
task build             # Compile the operator binary → bin/manager
task test              # Run unit + integration tests (uses envtest)
task lint              # Run golangci-lint
task lint-fix          # Run golangci-lint with auto-fix

# Local development
task install           # Apply CRDs to the current cluster
task run               # Run controller locally (outside cluster)
kubectl apply -f config/samples/   # Create sample resources

# Cluster deployment
task deploy IMG=kubeflow-coach:dev    # Deploy operator to cluster
task undeploy                         # Remove operator from cluster
task docker-build IMG=kubeflow-coach:dev  # Build container image
```

## Upstream References

- [Kubeflow Trainer repo](https://github.com/kubeflow/trainer)
- [Kubeflow Trainer docs](https://www.kubeflow.org/docs/components/trainer/overview/)
- [Kubebuilder book](https://book.kubebuilder.io/)
- [controller-runtime docs](https://pkg.go.dev/sigs.k8s.io/controller-runtime)