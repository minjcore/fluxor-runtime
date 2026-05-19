package rbac

// Permission is action + resource (e.g. "read", "users:*").
type Permission struct {
	Action   string
	Resource string
}

// Checker determines if a subject is allowed to perform an action on a resource.
type Checker interface {
	Allow(subjectID, action, resource string) (bool, error)
}
