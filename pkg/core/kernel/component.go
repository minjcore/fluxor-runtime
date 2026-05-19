package kernel

import "context"

// Component is a lifecycle-managed unit registered on the Kernel (HTTP server, bus bridge, worker pool, etc.).
type Component interface {
	Name() string
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
}
