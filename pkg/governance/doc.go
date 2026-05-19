// Package governance provides application-level governance: audit logging,
// billing/usage, policies, quotas, RBAC, and tenancy. Subpackages are
// independent and can be used separately.
//
// Subpackages:
//   - audit: immutable audit events and loggers (compliance, SOX, HIPAA, GDPR)
//   - billing: usage events and recording for metering/billing
//   - policies: policy evaluation (allow/deny) for actions and resources
//   - quotas: quota limits and consumption (rate/usage limiting)
//   - rbac: role-based access control (roles, permissions, checkers)
//   - tenancy: tenant context and isolation (multi-tenant scoping)
package governance
