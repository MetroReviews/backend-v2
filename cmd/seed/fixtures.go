package main

import "github.com/google/uuid"

var (
	categoryRestaurants = uuid.MustParse("00000000-0000-0000-0000-0000000000c1")
	categorySoftware    = uuid.MustParse("00000000-0000-0000-0000-0000000000c2")

	businessAlpha = uuid.MustParse("00000000-0000-0000-0000-0000000000d1")
	businessBeta  = uuid.MustParse("00000000-0000-0000-0000-0000000000d2")

	businessAlphaInvite        = uuid.MustParse("00000000-0000-0000-0000-0000000000d3")
	businessAlphaInvitePending = uuid.MustParse("00000000-0000-0000-0000-0000000000d4")

	projectKitchen = uuid.MustParse("00000000-0000-0000-0000-0000000000a1")
	projectPatio   = uuid.MustParse("00000000-0000-0000-0000-0000000000a2")

	userAlphaID    = uuid.MustParse("00000000-0000-0000-0000-0000000000f1")
	userBetaID     = uuid.MustParse("00000000-0000-0000-0000-0000000000f2")
	userExtraID    = uuid.MustParse("00000000-0000-0000-0000-0000000000f3")
	userReviewerID = uuid.MustParse("00000000-0000-0000-0000-0000000000f4")

	userLocalID = uuid.MustParse("00000000-0000-0000-0000-0000000000f5")

	userAlphaDiscord    = int64(2100000000000000001)
	userBetaDiscord     = int64(2100000000000000002)
	userExtraDiscord    = int64(2100000000000000003)
	userReviewerDiscord = int64(2200000000000000001)
)
