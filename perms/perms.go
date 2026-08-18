// Package perms defines Metro's permission catalog: the fixed set of
// capabilities a role (see the roles package) can grant. Permissions are
// plain strings so they fit in a Postgres TEXT[] and a role's JSON body
// without a lookup table — this package is just what any admin UI lists
// them from, and what the wildcard is short for.
package perms

// Wildcard grants every permission there is, present or future — Metro's
// "Sudo" role uses it instead of enumerating everything.
const Wildcard = "*"

const (
	// PanelAccess gates logging into the staff panel at all. It's kept
	// separate from the other permissions (rather than "any permission
	// implies panel access") so a role can eventually be scoped to a
	// single panel section without also being able to log in generally.
	PanelAccess = "panel.access"

	// QueueReview covers claiming, unclaiming, approving and denying bots,
	// businesses and projects in the shared review queue (/queue and the
	// standalone /claim /unclaim /approve /deny commands and endpoints).
	QueueReview = "queue.review"

	// ReviewsModerate covers flagging/removing reviews and resolving
	// reports filed against a review, business or bot.
	ReviewsModerate = "reviews.moderate"

	// ClaimsResolve covers approving/denying business ownership claims.
	ClaimsResolve = "claims.resolve"

	// RolesManage covers creating, editing and deleting roles, assigning
	// and unassigning them, and triggering a manual Discord role sync.
	RolesManage = "roles.manage"
)

// Permission is one catalog entry, for building an admin UI's role editor.
type Permission struct {
	Slug        string
	Description string
}

// Catalog is every permission a role can be given, in the order an admin
// UI should list them.
var Catalog = []Permission{
	{PanelAccess, "Log into the staff panel"},
	{QueueReview, "Claim, unclaim, approve and deny bots, businesses and projects"},
	{ReviewsModerate, "Flag/remove reviews and resolve reports"},
	{ClaimsResolve, "Approve or deny business ownership claims"},
	{RolesManage, "Manage roles and permissions, including the Discord role sync"},
}

// Valid reports whether slug is a known permission (or the wildcard) —
// used to reject typos/garbage when a role's permission list is set from
// the API rather than chosen from Catalog.
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

// Has reports whether granted (a role's or a user's effective permission
// set) includes want, honoring Wildcard.
func Has(granted []string, want string) bool {
	for _, p := range granted {
		if p == Wildcard || p == want {
			return true
		}
	}
	return false
}

// Union merges any number of permission sets into one deduplicated set,
// short-circuiting to just Wildcard if any set grants it.
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
