package tenancy

import (
	"context"
)

type contextKey string

const tenantContextKey contextKey = "governance.tenancy.tenant_id"

// TenantID is an opaque tenant identifier for multi-tenant isolation.
type TenantID string

// WithTenant returns a context that carries the tenant ID.
func WithTenant(ctx context.Context, tenant TenantID) context.Context {
	return context.WithValue(ctx, tenantContextKey, tenant)
}

// FromContext returns the tenant ID from the context, or empty if not set.
func FromContext(ctx context.Context) TenantID {
	v, _ := ctx.Value(tenantContextKey).(TenantID)
	return v
}

// HasTenant returns true if the context has a non-empty tenant.
func HasTenant(ctx context.Context) bool {
	return FromContext(ctx) != ""
}
