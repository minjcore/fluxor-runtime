package tenancy

import (
	"context"
	"testing"
)

func TestWithTenant_FromContext(t *testing.T) {
	ctx := context.Background()
	if HasTenant(ctx) {
		t.Error("expected no tenant on background context")
	}
	ctx = WithTenant(ctx, TenantID("tenant-1"))
	if !HasTenant(ctx) {
		t.Error("expected tenant after WithTenant")
	}
	id := FromContext(ctx)
	if id != TenantID("tenant-1") {
		t.Errorf("FromContext = %q", id)
	}
}

func TestFromContext_empty(t *testing.T) {
	ctx := context.Background()
	id := FromContext(ctx)
	if id != "" {
		t.Errorf("FromContext(background) = %q", id)
	}
}
