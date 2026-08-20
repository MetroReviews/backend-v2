package commands

import (
	"fmt"
	"sort"
	"strings"

	"github.com/MetroReviews/backend-v2/perms"
	"github.com/MetroReviews/backend-v2/types"
)

func discordRoleValue(role types.Role) string {
	if role.DiscordRoleID != nil {
		return fmt.Sprintf("<@&%d>", *role.DiscordRoleID)
	}
	return "_Panel-only — assigned by hand, not synced from Discord_"
}

func permissionsValue(permissions []string) string {
	if len(permissions) == 0 {
		return "_None_"
	}
	if perms.Has(permissions, perms.Wildcard) {
		return "**Everything** (`*`)"
	}
	sorted := sortedPermissions(permissions)
	for i, p := range sorted {
		sorted[i] = "`" + p + "`"
	}
	return strings.Join(sorted, ", ")
}

func permissionsSummary(permissions []string) string {
	if len(permissions) == 0 {
		return "No permissions"
	}
	if perms.Has(permissions, perms.Wildcard) {
		return "Everything (Sudo)"
	}
	return strings.Join(sortedPermissions(permissions), ", ")
}

func sortedPermissions(permissions []string) []string {
	sorted := append([]string(nil), permissions...)
	sort.Strings(sorted)
	return sorted
}
