package helpers

import "strings"

// Truncate shortens s to at most max characters, replacing the last
// character with an ellipsis when it had to cut anything. Used to keep
// user-supplied text within Discord's field/option length limits.
func Truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	if max <= 1 {
		return s[:max]
	}
	return s[:max-1] + "…"
}

// HTTPSify normalizes v to an https:// URL: it's returned as-is if already
// https, upgraded if it's http, and rejected (ok=false) otherwise.
func HTTPSify(v string) (string, bool) {
	if strings.HasPrefix(v, "https://") {
		return v, true
	}
	v = strings.Replace(v, "http://", "https://", 1)
	if strings.HasPrefix(v, "https://") {
		return v, true
	}
	return "", false
}

// InviteURL builds the Discord OAuth2 invite link for a bot ID. Shared by
// the bot's /invite command, its queue/detail views, and the "bot added to
// queue" announcement.
func InviteURL(botID string) string {
	return "https://discord.com/oauth2/authorize?client_id=" + botID + "&scope=bot%20applications.commands&permissions=0"
}
