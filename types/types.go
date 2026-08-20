package types

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
