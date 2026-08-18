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

	// EventTest is never subscribed to (Dispatch never fires it against
	// the events filter) — POST /webhooks/{id}/test sends it straight to
	// one webhook to verify delivery/signing end to end.
	EventTest = "webhook.test"
)

// Catalog is every event a webhook can subscribe to, in the order an
// admin UI should list them. Adding a new one is just adding an entry
// here — nothing else in this package needs to change, and Dispatch's
// callers (review actions, the review queue, and whatever future subject
// type follows the same pattern) decide when to fire it.
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

// ValidEvent reports whether name is a known, subscribable event —
// EventTest deliberately excluded, since it's not something a webhook
// subscribes to.
func ValidEvent(name string) bool {
	for _, e := range Catalog {
		if e.Name == name {
			return true
		}
	}
	return false
}

// queueActionEvents maps a review-queue action (see the types/review
// packages) to the event DispatchQueueAction fires for it.
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
