package dto

// PullRequest represents a pull request in the system
type PullRequest struct {
	Author            PullRequestAuthor       `json:"author,omitempty"`
	CheckSummary      PullRequestCheckSummary `json:"check_summary,omitempty"`
	Closed            int64                   `json:"closed,omitempty"`
	Created           int64                   `json:"created,omitempty"`
	Description       string                  `json:"description,omitempty"`
	Edited            int64                   `json:"edited,omitempty"`
	IsDraft           bool                    `json:"is_draft,omitempty"`
	Labels            []PullRequestLabel      `json:"labels,omitempty"`
	MergeBaseSha      string                  `json:"merge_base_sha,omitempty"`
	MergeCheckStatus  string                  `json:"merge_check_status,omitempty"`
	MergeConflicts    []string                `json:"merge_conflicts,omitempty"`
	MergeMethod       string                  `json:"merge_method,omitempty"`
	MergeTargetSha    string                  `json:"merge_target_sha,omitempty"`
	Merged            int64                   `json:"merged,omitempty"`
	Merger            PullRequestAuthor       `json:"merger,omitempty"`
	Number            int                     `json:"number,omitempty"`
	RebaseCheckStatus string                  `json:"rebase_check_status,omitempty"`
	RebaseConflicts   []string                `json:"rebase_conflicts,omitempty"`
	Rules             []PullRequestRule       `json:"rules,omitempty"`
	SourceBranch      string                  `json:"source_branch,omitempty"`
	SourceRepoID      int                     `json:"source_repo_id,omitempty"`
	SourceSha         string                  `json:"source_sha,omitempty"`
	State             string                  `json:"state,omitempty"`
	Stats             PullRequestStats        `json:"stats,omitempty"`
	TargetBranch      string                  `json:"target_branch,omitempty"`
	TargetRepoID      int                     `json:"target_repo_id,omitempty"`
	Title             string                  `json:"title,omitempty"`
	Updated           int64                   `json:"updated,omitempty"`
}

// PullRequestAuthor represents a user in the pull request system
type PullRequestAuthor struct {
	Created     int64  `json:"created,omitempty"`
	DisplayName string `json:"display_name,omitempty"`
	Email       string `json:"email,omitempty"`
	ID          int    `json:"id,omitempty"`
	Type        string `json:"type,omitempty"`
	UID         string `json:"uid,omitempty"`
	Updated     int64  `json:"updated,omitempty"`
}

// PullRequestCheckSummary represents the summary of checks for a pull request
type PullRequestCheckSummary struct {
	Error   int `json:"error,omitempty"`
	Failure int `json:"failure,omitempty"`
	Pending int `json:"pending,omitempty"`
	Running int `json:"running,omitempty"`
	Success int `json:"success,omitempty"`
}

// PullRequestLabel represents a label on a pull request
type PullRequestLabel struct {
	Color      string `json:"color,omitempty"`
	ID         int    `json:"id,omitempty"`
	Key        string `json:"key,omitempty"`
	Scope      int    `json:"scope,omitempty"`
	Value      string `json:"value,omitempty"`
	ValueColor string `json:"value_color,omitempty"`
	ValueCount int    `json:"value_count,omitempty"`
	ValueID    int    `json:"value_id,omitempty"`
}

// PullRequestRule represents a rule for a pull request
type PullRequestRule struct {
	Identifier string `json:"identifier,omitempty"`
	RepoPath   string `json:"repo_path,omitempty"`
	SpacePath  string `json:"space_path,omitempty"`
	State      string `json:"state,omitempty"`
	Type       string `json:"type,omitempty"`
}

// PullRequestStats represents statistics for a pull request
type PullRequestStats struct {
	Additions       int `json:"additions,omitempty"`
	Commits         int `json:"commits,omitempty"`
	Conversations   int `json:"conversations,omitempty"`
	Deletions       int `json:"deletions,omitempty"`
	FilesChanged    int `json:"files_changed,omitempty"`
	UnresolvedCount int `json:"unresolved_count,omitempty"`
}

// CreatePullRequest represents the request body for creating a new pull request
type CreatePullRequest struct {
	Title        string `json:"title"`
	Description  string `json:"description,omitempty"`
	SourceBranch string `json:"source_branch"`
	TargetBranch string `json:"target_branch,omitempty"`
	IsDraft      bool   `json:"is_draft,omitempty"`
}

// PullRequestOptions represents the options for listing pull requests
type PullRequestOptions struct {
	State         []string `json:"state,omitempty"`
	SourceRepoRef string   `json:"source_repo_ref,omitempty"`
	SourceBranch  string   `json:"source_branch,omitempty"`
	TargetBranch  string   `json:"target_branch,omitempty"`
	Query         string   `json:"query,omitempty"`
	CreatedBy     []int    `json:"created_by,omitempty"`
	Order         string   `json:"order,omitempty"`
	Sort          string   `json:"sort,omitempty"`
	CreatedLt     int64    `json:"created_lt,omitempty"`
	CreatedGt     int64    `json:"created_gt,omitempty"`
	UpdatedLt     int64    `json:"updated_lt,omitempty"`
	UpdatedGt     int64    `json:"updated_gt,omitempty"`
	Page          int      `json:"page,omitempty"`
	Limit         int      `json:"limit,omitempty"`
	AuthorID      int      `json:"author_id,omitempty"`
	IncludeChecks bool     `json:"include_checks,omitempty"`
}
