package main

import "github.com/google/uuid"

// Fixed IDs so reseeding updates the same rows instead of duplicating them.
// user*ID are Metro accounts; user*Discord are the Discord accounts linked
// to them (see the identity package) — two different ID spaces on purpose,
// same as in the real schema.
var (
	categoryRestaurants = uuid.MustParse("00000000-0000-0000-0000-0000000000c1")
	categorySoftware    = uuid.MustParse("00000000-0000-0000-0000-0000000000c2")

	businessAlpha = uuid.MustParse("00000000-0000-0000-0000-0000000000d1")
	businessBeta  = uuid.MustParse("00000000-0000-0000-0000-0000000000d2")

	// projectKitchen is Approved and has reviews (see seedReviews);
	// projectPatio is Pending, to demo /queue with a project in it.
	projectKitchen = uuid.MustParse("00000000-0000-0000-0000-0000000000a1")
	projectPatio   = uuid.MustParse("00000000-0000-0000-0000-0000000000a2")

	userAlphaID    = uuid.MustParse("00000000-0000-0000-0000-0000000000f1")
	userBetaID     = uuid.MustParse("00000000-0000-0000-0000-0000000000f2")
	userExtraID    = uuid.MustParse("00000000-0000-0000-0000-0000000000f3")
	userReviewerID = uuid.MustParse("00000000-0000-0000-0000-0000000000f4")

	userAlphaDiscord    = int64(2100000000000000001)
	userBetaDiscord     = int64(2100000000000000002)
	userExtraDiscord    = int64(2100000000000000003)
	userReviewerDiscord = int64(2200000000000000001)
)

func strPtr(s string) *string { return &s }
func int64Ptr(i int64) *int64 { return &i }
