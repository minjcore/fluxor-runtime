package github

import "context"

// Client is the main GitHub client interface
type Client interface {
	Repositories() RepositoriesClient
	Issues() IssuesClient
	PullRequests() PullRequestsClient
	Users() UsersClient
}

// RepositoriesClient provides operations for GitHub repositories
type RepositoriesClient interface {
	// List returns repositories for the authenticated user or organization
	List(ctx context.Context, owner string, params ListReposParams) ([]Repository, error)

	// Get returns a specific repository
	Get(ctx context.Context, owner, repo string) (*Repository, error)

	// Create creates a new repository
	Create(ctx context.Context, repo *CreateRepoRequest) (*Repository, error)

	// Delete deletes a repository
	Delete(ctx context.Context, owner, repo string) error

	// ListBranches lists branches for a repository
	ListBranches(ctx context.Context, owner, repo string) ([]Branch, error)

	// GetBranch gets a specific branch
	GetBranch(ctx context.Context, owner, repo, branch string) (*Branch, error)
}

// IssuesClient provides operations for GitHub issues
type IssuesClient interface {
	// List returns issues for a repository
	List(ctx context.Context, owner, repo string, params ListIssuesParams) ([]Issue, error)

	// Get returns a specific issue
	Get(ctx context.Context, owner, repo string, number int) (*Issue, error)

	// Create creates a new issue
	Create(ctx context.Context, owner, repo string, issue *CreateIssueRequest) (*Issue, error)

	// Update updates an existing issue
	Update(ctx context.Context, owner, repo string, number int, issue *UpdateIssueRequest) (*Issue, error)

	// Close closes an issue
	Close(ctx context.Context, owner, repo string, number int) (*Issue, error)

	// CreateComment creates a comment on an issue
	CreateComment(ctx context.Context, owner, repo string, number int, body string) (*Comment, error)

	// ListComments lists comments on an issue
	ListComments(ctx context.Context, owner, repo string, number int) ([]Comment, error)
}

// PullRequestsClient provides operations for GitHub pull requests
type PullRequestsClient interface {
	// List returns pull requests for a repository
	List(ctx context.Context, owner, repo string, params ListPRParams) ([]PullRequest, error)

	// Get returns a specific pull request
	Get(ctx context.Context, owner, repo string, number int) (*PullRequest, error)

	// Create creates a new pull request
	Create(ctx context.Context, owner, repo string, pr *CreatePRRequest) (*PullRequest, error)

	// Merge merges a pull request
	Merge(ctx context.Context, owner, repo string, number int, opts *MergeOptions) (*MergeResult, error)

	// ListReviews lists reviews for a pull request
	ListReviews(ctx context.Context, owner, repo string, number int) ([]Review, error)

	// CreateReview creates a review for a pull request
	CreateReview(ctx context.Context, owner, repo string, number int, review *CreateReviewRequest) (*Review, error)
}

// UsersClient provides operations for GitHub users
type UsersClient interface {
	// Get returns the authenticated user
	Get(ctx context.Context) (*User, error)

	// GetByUsername returns a user by username
	GetByUsername(ctx context.Context, username string) (*User, error)

	// ListOrgs lists organizations for the authenticated user
	ListOrgs(ctx context.Context) ([]Organization, error)
}

// Repository represents a GitHub repository
type Repository struct {
	ID              int64    `json:"id"`
	NodeID          string   `json:"node_id"`
	Name            string   `json:"name"`
	FullName        string   `json:"full_name"`
	Description     string   `json:"description"`
	Private         bool     `json:"private"`
	Fork            bool     `json:"fork"`
	URL             string   `json:"url"`
	HTMLURL         string   `json:"html_url"`
	CloneURL        string   `json:"clone_url"`
	SSHURL          string   `json:"ssh_url"`
	DefaultBranch   string   `json:"default_branch"`
	Language        string   `json:"language"`
	StargazersCount int      `json:"stargazers_count"`
	WatchersCount   int      `json:"watchers_count"`
	ForksCount      int      `json:"forks_count"`
	OpenIssuesCount int      `json:"open_issues_count"`
	CreatedAt       string   `json:"created_at"`
	UpdatedAt       string   `json:"updated_at"`
	PushedAt        string   `json:"pushed_at"`
	Owner           *User    `json:"owner"`
	Topics          []string `json:"topics"`
	Archived        bool     `json:"archived"`
	Disabled        bool     `json:"disabled"`
}

// Branch represents a Git branch
type Branch struct {
	Name      string  `json:"name"`
	Commit    *Commit `json:"commit"`
	Protected bool    `json:"protected"`
}

// Commit represents a Git commit
type Commit struct {
	SHA    string      `json:"sha"`
	URL    string      `json:"url"`
	Author *CommitUser `json:"author"`
}

// CommitUser represents a commit author
type CommitUser struct {
	Name  string `json:"name"`
	Email string `json:"email"`
	Date  string `json:"date"`
}

// Issue represents a GitHub issue
type Issue struct {
	ID          int64      `json:"id"`
	NodeID      string     `json:"node_id"`
	Number      int        `json:"number"`
	Title       string     `json:"title"`
	Body        string     `json:"body"`
	State       string     `json:"state"` // "open" or "closed"
	HTMLURL     string     `json:"html_url"`
	User        *User      `json:"user"`
	Labels      []Label    `json:"labels"`
	Assignees   []User     `json:"assignees"`
	Milestone   *Milestone `json:"milestone"`
	Comments    int        `json:"comments"`
	CreatedAt   string     `json:"created_at"`
	UpdatedAt   string     `json:"updated_at"`
	ClosedAt    string     `json:"closed_at,omitempty"`
	PullRequest *PRRef     `json:"pull_request,omitempty"`
}

// PRRef is a reference to a pull request in an issue
type PRRef struct {
	URL      string `json:"url"`
	HTMLURL  string `json:"html_url"`
	DiffURL  string `json:"diff_url"`
	PatchURL string `json:"patch_url"`
}

// Label represents a GitHub label
type Label struct {
	ID          int64  `json:"id"`
	NodeID      string `json:"node_id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Color       string `json:"color"`
	Default     bool   `json:"default"`
}

// Milestone represents a GitHub milestone
type Milestone struct {
	ID           int64  `json:"id"`
	Number       int    `json:"number"`
	Title        string `json:"title"`
	Description  string `json:"description"`
	State        string `json:"state"`
	OpenIssues   int    `json:"open_issues"`
	ClosedIssues int    `json:"closed_issues"`
	DueOn        string `json:"due_on,omitempty"`
}

// Comment represents a comment on an issue or PR
type Comment struct {
	ID        int64  `json:"id"`
	NodeID    string `json:"node_id"`
	Body      string `json:"body"`
	User      *User  `json:"user"`
	HTMLURL   string `json:"html_url"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

// PullRequest represents a GitHub pull request
type PullRequest struct {
	ID                int64      `json:"id"`
	NodeID            string     `json:"node_id"`
	Number            int        `json:"number"`
	Title             string     `json:"title"`
	Body              string     `json:"body"`
	State             string     `json:"state"` // "open", "closed"
	HTMLURL           string     `json:"html_url"`
	DiffURL           string     `json:"diff_url"`
	PatchURL          string     `json:"patch_url"`
	User              *User      `json:"user"`
	Head              *PRBranch  `json:"head"`
	Base              *PRBranch  `json:"base"`
	Draft             bool       `json:"draft"`
	Merged            bool       `json:"merged"`
	Mergeable         *bool      `json:"mergeable"`
	MergeableState    string     `json:"mergeable_state"`
	MergedBy          *User      `json:"merged_by,omitempty"`
	MergeCommitSHA    string     `json:"merge_commit_sha,omitempty"`
	Comments          int        `json:"comments"`
	ReviewComments    int        `json:"review_comments"`
	Commits           int        `json:"commits"`
	Additions         int        `json:"additions"`
	Deletions         int        `json:"deletions"`
	ChangedFiles      int        `json:"changed_files"`
	Labels            []Label    `json:"labels"`
	Assignees         []User     `json:"assignees"`
	RequestedReviewers []User    `json:"requested_reviewers"`
	Milestone         *Milestone `json:"milestone"`
	CreatedAt         string     `json:"created_at"`
	UpdatedAt         string     `json:"updated_at"`
	ClosedAt          string     `json:"closed_at,omitempty"`
	MergedAt          string     `json:"merged_at,omitempty"`
}

// PRBranch represents a branch in a pull request
type PRBranch struct {
	Label string      `json:"label"`
	Ref   string      `json:"ref"`
	SHA   string      `json:"sha"`
	User  *User       `json:"user"`
	Repo  *Repository `json:"repo"`
}

// Review represents a pull request review
type Review struct {
	ID          int64  `json:"id"`
	NodeID      string `json:"node_id"`
	User        *User  `json:"user"`
	Body        string `json:"body"`
	State       string `json:"state"` // APPROVED, CHANGES_REQUESTED, COMMENTED, DISMISSED, PENDING
	HTMLURL     string `json:"html_url"`
	SubmittedAt string `json:"submitted_at"`
	CommitID    string `json:"commit_id"`
}

// User represents a GitHub user
type User struct {
	ID        int64  `json:"id"`
	NodeID    string `json:"node_id"`
	Login     string `json:"login"`
	Name      string `json:"name,omitempty"`
	Email     string `json:"email,omitempty"`
	AvatarURL string `json:"avatar_url"`
	HTMLURL   string `json:"html_url"`
	Type      string `json:"type"` // "User" or "Organization"
	SiteAdmin bool   `json:"site_admin"`
	Bio       string `json:"bio,omitempty"`
	Company   string `json:"company,omitempty"`
	Location  string `json:"location,omitempty"`
	Blog      string `json:"blog,omitempty"`
}

// Organization represents a GitHub organization
type Organization struct {
	ID          int64  `json:"id"`
	NodeID      string `json:"node_id"`
	Login       string `json:"login"`
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
	AvatarURL   string `json:"avatar_url"`
	HTMLURL     string `json:"html_url"`
}

// ListReposParams contains parameters for listing repositories
type ListReposParams struct {
	Type      string `json:"type,omitempty"`      // all, owner, public, private, member
	Sort      string `json:"sort,omitempty"`      // created, updated, pushed, full_name
	Direction string `json:"direction,omitempty"` // asc, desc
	PerPage   int    `json:"per_page,omitempty"`
	Page      int    `json:"page,omitempty"`
}

// ListIssuesParams contains parameters for listing issues
type ListIssuesParams struct {
	State     string   `json:"state,omitempty"`     // open, closed, all
	Labels    []string `json:"labels,omitempty"`
	Sort      string   `json:"sort,omitempty"`      // created, updated, comments
	Direction string   `json:"direction,omitempty"` // asc, desc
	Since     string   `json:"since,omitempty"`
	PerPage   int      `json:"per_page,omitempty"`
	Page      int      `json:"page,omitempty"`
}

// ListPRParams contains parameters for listing pull requests
type ListPRParams struct {
	State     string `json:"state,omitempty"`     // open, closed, all
	Head      string `json:"head,omitempty"`
	Base      string `json:"base,omitempty"`
	Sort      string `json:"sort,omitempty"`      // created, updated, popularity, long-running
	Direction string `json:"direction,omitempty"` // asc, desc
	PerPage   int    `json:"per_page,omitempty"`
	Page      int    `json:"page,omitempty"`
}

// CreateRepoRequest contains parameters for creating a repository
type CreateRepoRequest struct {
	Name          string `json:"name"`
	Description   string `json:"description,omitempty"`
	Private       bool   `json:"private,omitempty"`
	AutoInit      bool   `json:"auto_init,omitempty"`
	GitIgnore     string `json:"gitignore_template,omitempty"`
	License       string `json:"license_template,omitempty"`
	AllowSquash   bool   `json:"allow_squash_merge,omitempty"`
	AllowMerge    bool   `json:"allow_merge_commit,omitempty"`
	AllowRebase   bool   `json:"allow_rebase_merge,omitempty"`
}

// CreateIssueRequest contains parameters for creating an issue
type CreateIssueRequest struct {
	Title     string   `json:"title"`
	Body      string   `json:"body,omitempty"`
	Assignees []string `json:"assignees,omitempty"`
	Labels    []string `json:"labels,omitempty"`
	Milestone *int     `json:"milestone,omitempty"`
}

// UpdateIssueRequest contains parameters for updating an issue
type UpdateIssueRequest struct {
	Title     string   `json:"title,omitempty"`
	Body      string   `json:"body,omitempty"`
	State     string   `json:"state,omitempty"` // open, closed
	Assignees []string `json:"assignees,omitempty"`
	Labels    []string `json:"labels,omitempty"`
	Milestone *int     `json:"milestone,omitempty"`
}

// CreatePRRequest contains parameters for creating a pull request
type CreatePRRequest struct {
	Title string `json:"title"`
	Body  string `json:"body,omitempty"`
	Head  string `json:"head"` // branch to merge from
	Base  string `json:"base"` // branch to merge into
	Draft bool   `json:"draft,omitempty"`
}

// MergeOptions contains options for merging a pull request
type MergeOptions struct {
	CommitTitle   string `json:"commit_title,omitempty"`
	CommitMessage string `json:"commit_message,omitempty"`
	SHA           string `json:"sha,omitempty"`
	MergeMethod   string `json:"merge_method,omitempty"` // merge, squash, rebase
}

// MergeResult represents the result of merging a pull request
type MergeResult struct {
	SHA     string `json:"sha"`
	Merged  bool   `json:"merged"`
	Message string `json:"message"`
}

// CreateReviewRequest contains parameters for creating a review
type CreateReviewRequest struct {
	Body     string          `json:"body,omitempty"`
	Event    string          `json:"event"` // APPROVE, REQUEST_CHANGES, COMMENT
	Comments []ReviewComment `json:"comments,omitempty"`
}

// ReviewComment represents a comment in a review
type ReviewComment struct {
	Path     string `json:"path"`
	Position int    `json:"position,omitempty"`
	Line     int    `json:"line,omitempty"`
	Side     string `json:"side,omitempty"` // LEFT, RIGHT
	Body     string `json:"body"`
}

// APIResponse represents a generic API response
type APIResponse struct {
	Message          string `json:"message,omitempty"`
	DocumentationURL string `json:"documentation_url,omitempty"`
}
