package perms

const Wildcard = "*"

const (
	PanelAccess = "panel.access"

	QueueReview = "queue.review"

	ReviewsModerate = "reviews.moderate"

	ClaimsResolve = "claims.resolve"

	RolesManage = "roles.manage"

	BusinessesFeature = "businesses.feature"
)

type Permission struct {
	Slug        string
	Description string
}

var Catalog = []Permission{
	{PanelAccess, "Log into the staff panel"},
	{QueueReview, "Claim, unclaim, approve and deny businesses and projects"},
	{ReviewsModerate, "Flag/remove reviews and resolve reports"},
	{ClaimsResolve, "Approve or deny business ownership claims"},
	{RolesManage, "Manage roles and permissions, including the Discord role sync"},
	{BusinessesFeature, "Toggle a business's sponsored/featured placement"},
}

func Valid(slug string) bool {
	if slug == Wildcard {
		return true
	}
	for _, p := range Catalog {
		if p.Slug == slug {
			return true
		}
	}
	return false
}

func Has(granted []string, want string) bool {
	for _, p := range granted {
		if p == Wildcard || p == want {
			return true
		}
	}
	return false
}

func Union(sets ...[]string) []string {
	for _, set := range sets {
		for _, p := range set {
			if p == Wildcard {
				return []string{Wildcard}
			}
		}
	}

	seen := map[string]struct{}{}
	out := []string{}
	for _, set := range sets {
		for _, p := range set {
			if _, ok := seen[p]; ok {
				continue
			}
			seen[p] = struct{}{}
			out = append(out, p)
		}
	}
	return out
}
