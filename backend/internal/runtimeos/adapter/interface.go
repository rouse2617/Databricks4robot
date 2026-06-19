package adapter

import (
	"context"
	"io"
	"time"
)

// RunRef is the product execution identity passed into runtime adapters.
// Runtime adapters may use it for labels, annotations, and event correlation,
// but must not treat runtime job names as product IDs.
type RunRef struct {
	ID   string
	Name string
}

// RuntimeRef points at an external runtime object. For Argo this is a
// Workflow name, namespace, and optional UID.
type RuntimeRef struct {
	RuntimeType string
	Name        string
	Namespace   string
	UID         string
}

// RuntimeSpec is the runtime-specific submission payload. Phase 2 keeps the
// manifest opaque so the Run Kernel can form the boundary before schema moves.
type RuntimeSpec struct {
	RuntimeType string
	Namespace   string
	Manifest    any
}

// RuntimeJob is the adapter-neutral representation of a submitted runtime job.
type RuntimeJob struct {
	Ref RuntimeRef
	Raw any
}

// RuntimeJobStatus is the adapter-neutral status snapshot for a runtime job.
type RuntimeJobStatus struct {
	Ref        RuntimeRef
	Status     string
	Message    string
	StartedAt  *time.Time
	FinishedAt *time.Time
	Raw        any
}

type RetryOptions struct{}

type ResubmitOptions struct{}

type LogOptions struct {
	Container    string
	TailLines    *int64
	LimitBytes   *int64
	SinceSeconds *int64
	SinceTime    string
	Previous     bool
	Timestamps   bool
	Follow       bool
}

type LogResult struct {
	Logs       string
	LineCount  int
	Truncated  bool
	LimitBytes int64
}

// RuntimeAdapter isolates Run Kernel lifecycle operations from a concrete
// runtime such as Argo Workflows.
type RuntimeAdapter interface {
	Submit(ctx context.Context, run RunRef, spec RuntimeSpec) (*RuntimeJob, error)
	Get(ctx context.Context, ref RuntimeRef) (*RuntimeJobStatus, error)
	Stop(ctx context.Context, ref RuntimeRef) error
	Suspend(ctx context.Context, ref RuntimeRef) error
	Resume(ctx context.Context, ref RuntimeRef) error
	Terminate(ctx context.Context, ref RuntimeRef) error
	Retry(ctx context.Context, ref RuntimeRef, opts RetryOptions) (*RuntimeJob, error)
	Resubmit(ctx context.Context, ref RuntimeRef, opts ResubmitOptions) (*RuntimeJob, error)
	Logs(ctx context.Context, ref RuntimeRef, nodeID string, opts LogOptions) (*LogResult, error)
	LogStream(ctx context.Context, ref RuntimeRef, nodeID string, opts LogOptions) (io.ReadCloser, error)
}
