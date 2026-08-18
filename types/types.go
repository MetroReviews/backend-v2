package types

// Action, State: the shared moderation queue both bots and businesses go
// through — claim it (start reviewing), then approve or deny — surfaced
// together in the Discord /queue command. A submission stays PENDING until
// a staff member claims it; SUSPENDED is for pulling something back down
// after it was already approved (e.g. resolving a report against it).
type Action int

const (
	ActionClaim Action = iota
	ActionUnclaim
	ActionApprove
	ActionDeny
)

type State int

const (
	StatePending State = iota
	StateUnderReview
	StateApproved
	StateDenied
	StateSuspended
)

type ReviewStatus int

const (
	ReviewStatusPublished ReviewStatus = iota
	ReviewStatusFlagged
	ReviewStatusRemoved
)

type ReportStatus int

const (
	ReportStatusOpen ReportStatus = iota
	ReportStatusResolved
	ReportStatusDismissed
)

type ClaimStatus int

const (
	ClaimStatusPending ClaimStatus = iota
	ClaimStatusApproved
	ClaimStatusDenied
)

type ApiError struct {
	Message string            `json:"message" description:"A human readable message describing the result"`
	Error   bool              `json:"error" description:"Whether this response represents an error"`
	Context map[string]string `json:"context,omitempty" description:"Additional context, e.g. per-field validation errors"`
}

type UpdatedResponse struct {
	HasUpdated []string `json:"has_updated" description:"The fields that were updated"`
}
