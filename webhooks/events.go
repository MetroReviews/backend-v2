package webhooks

import "github.com/MetroReviews/backend-v2/types"

const (
	EventReviewCreated   = "review.created"
	EventReviewUpdated   = "review.updated"
	EventReviewDeleted   = "review.deleted"
	EventReviewVoted     = "review.voted"
	EventReviewResponded = "review.responded"

	EventQueueClaimed   = "queue.claimed"
	EventQueueUnclaimed = "queue.unclaimed"
	EventQueueApproved  = "queue.approved"
	EventQueueDenied    = "queue.denied"

	EventTest = "webhook.test"
)

var Catalog = []types.WebhookEvent{
	{Name: EventReviewCreated, Description: "A new review was posted"},
	{Name: EventReviewUpdated, Description: "A review was edited"},
	{Name: EventReviewDeleted, Description: "A review was deleted"},
	{Name: EventReviewVoted, Description: "A review was marked helpful or unhelpful"},
	{Name: EventReviewResponded, Description: "The owner replied to a review"},
	{Name: EventQueueClaimed, Description: "Claimed for staff review"},
	{Name: EventQueueUnclaimed, Description: "Released back to pending"},
	{Name: EventQueueApproved, Description: "Approved and published"},
	{Name: EventQueueDenied, Description: "Denied"},
}

func ValidEvent(name string) bool {
	for _, e := range Catalog {
		if e.Name == name {
			return true
		}
	}
	return false
}

var queueActionEvents = map[types.Action]string{
	types.ActionClaim:   EventQueueClaimed,
	types.ActionUnclaim: EventQueueUnclaimed,
	types.ActionApprove: EventQueueApproved,
	types.ActionDeny:    EventQueueDenied,
}

var queueActionNames = map[types.Action]string{
	types.ActionClaim:   "Claim",
	types.ActionUnclaim: "Unclaim",
	types.ActionApprove: "Approve",
	types.ActionDeny:    "Deny",
}
