package roles

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/MetroReviews/backend-v2/identity"
	"github.com/MetroReviews/backend-v2/perms"
	"github.com/MetroReviews/backend-v2/state"
	"github.com/MetroReviews/backend-v2/types"
	"github.com/bwmarrin/discordgo"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// SyncMember reconciles userID's assignment to every Discord-linked role
// against heldDiscordRoleIDs (a guild member's current Roles), then
// refreshes users.is_staff from the resulting permission set. Panel-only
// role assignments (no linked Discord role) are untouched.
//
// discordID is that same user's Discord ID: a config.yaml owner bypasses
// the permissions system entirely, so is_staff is forced true for them
// regardless of what roles (if any) they hold or are even configured yet.
// It's also used to report the change (if any) to the configured logs
// channel — see postRoleSyncLog. Most calls are no-ops (a login or profile
// update for someone whose linked roles didn't actually change), which
// aren't logged.
func SyncMember(ctx context.Context, userID uuid.UUID, discordID int64, heldDiscordRoleIDs []string) error {
	linked, err := linkedRoles(ctx)
	if err != nil {
		return err
	}

	held := map[uuid.UUID]bool{}
	for _, discordRoleID := range heldDiscordRoleIDs {
		if role, ok := linked[discordRoleID]; ok {
			held[role.ID] = true
		}
	}

	before, err := assignedLinkedRoleIDs(ctx, userID)
	if err != nil {
		return err
	}

	tx, err := state.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck // no-op once committed

	for _, role := range linked {
		if held[role.ID] {
			if _, err := tx.Exec(ctx, `
				INSERT INTO user_roles (user_id, role_id) VALUES ($1, $2)
				ON CONFLICT DO NOTHING`, userID, role.ID); err != nil {
				return err
			}
		} else if _, err := tx.Exec(ctx,
			"DELETE FROM user_roles WHERE user_id = $1 AND role_id = $2", userID, role.ID,
		); err != nil {
			return err
		}
	}

	rows, err := tx.Query(ctx, `
		SELECT r.name, r.permissions FROM roles r JOIN user_roles ur ON ur.role_id = r.id WHERE ur.user_id = $1`, userID)
	if err != nil {
		return err
	}
	var currentRoleNames []string
	var sets [][]string
	for rows.Next() {
		var name string
		var p []string
		if err := rows.Scan(&name, &p); err != nil {
			rows.Close()
			return err
		}
		currentRoleNames = append(currentRoleNames, name)
		sets = append(sets, p)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}

	currentPerms := perms.Union(sets...)
	isStaff := perms.Has(currentPerms, perms.PanelAccess) || state.Config.IsOwner(discordID)
	if _, err := tx.Exec(ctx, "UPDATE users SET is_staff = $1 WHERE id = $2", isStaff, userID); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return err
	}

	if added, removed, changed := diffRoleNames(before, held, linked); changed {
		postRoleSyncLog(discordID, added, removed, currentRoleNames, currentPerms)
	}

	return nil
}

// assignedLinkedRoleIDs returns which of userID's currently-assigned roles
// are Discord-linked — SyncMember's "before" snapshot, compared against
// the "after" set once it's done reconciling.
func assignedLinkedRoleIDs(ctx context.Context, userID uuid.UUID) (map[uuid.UUID]bool, error) {
	rows, err := state.Pool.Query(ctx, `
		SELECT ur.role_id FROM user_roles ur
		JOIN roles r ON r.id = ur.role_id
		WHERE ur.user_id = $1 AND r.discord_role_id IS NOT NULL`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := map[uuid.UUID]bool{}
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out[id] = true
	}
	return out, rows.Err()
}

// diffRoleNames compares a linked-role assignment snapshot from before a
// sync to the set applied after, returning the role names added/removed.
// changed is false (added/removed both nil) when the sets are identical —
// the common case, and the caller's cue not to log anything.
func diffRoleNames(before, after map[uuid.UUID]bool, linked map[string]types.Role) (added, removed []string, changed bool) {
	names := make(map[uuid.UUID]string, len(linked))
	for _, role := range linked {
		names[role.ID] = role.Name
	}

	for id := range after {
		if !before[id] {
			added = append(added, names[id])
			changed = true
		}
	}
	for id := range before {
		if !after[id] {
			removed = append(removed, names[id])
			changed = true
		}
	}
	return added, removed, changed
}

// postRoleSyncLog reports one member's role change to the configured logs
// channel — the role-sync equivalent of bot/commands/result.go's
// logAction for review actions. A no-op if the bot isn't connected or no
// logs channel is configured.
func postRoleSyncLog(discordID int64, added, removed, currentRoles, currentPerms []string) {
	if state.Discord == nil || state.Config.LogsChannelID() == 0 {
		return
	}

	fields := []*discordgo.MessageEmbedField{
		{Name: "Member", Value: fmt.Sprintf("<@%d>", discordID), Inline: true},
	}
	if len(added) > 0 {
		fields = append(fields, &discordgo.MessageEmbedField{Name: "➕ Roles Added", Value: strings.Join(added, ", ")})
	}
	if len(removed) > 0 {
		fields = append(fields, &discordgo.MessageEmbedField{Name: "➖ Roles Removed", Value: strings.Join(removed, ", ")})
	}
	fields = append(fields,
		&discordgo.MessageEmbedField{Name: "Current Roles", Value: joinOrNone(currentRoles)},
		&discordgo.MessageEmbedField{Name: "Current Permissions", Value: joinOrNone(currentPerms)},
	)

	embed := &discordgo.MessageEmbed{
		Title:     "🔄 Roles Synced",
		Color:     0x5865f2,
		Fields:    fields,
		Timestamp: time.Now().Format(time.RFC3339),
	}

	channelID := strconv.FormatUint(state.Config.LogsChannelID(), 10)
	if _, err := state.Discord.ChannelMessageSendEmbed(channelID, embed); err != nil {
		state.Logger.Error("[roles] failed to send role sync log", zap.Error(err))
	}
}

func joinOrNone(items []string) string {
	if len(items) == 0 {
		return "_None_"
	}
	return strings.Join(items, ", ")
}

// SyncGuild reconciles every user's Discord-linked role assignments
// against the guild's actual current membership: a full pass, in contrast
// to SyncMember's single-user incremental update. It's a no-op if no role
// is Discord-linked yet.
func SyncGuild(ctx context.Context, s *discordgo.Session, guildID uint64) error {
	linked, err := linkedRoles(ctx)
	if err != nil {
		return err
	}
	if len(linked) == 0 {
		return nil
	}

	gid := strconv.FormatUint(guildID, 10)
	held := map[int64][]string{}

	after := ""
	for {
		batch, err := s.GuildMembers(gid, after, 1000)
		if err != nil {
			return err
		}
		for _, m := range batch {
			var mine []string
			for _, roleID := range m.Roles {
				if _, ok := linked[roleID]; ok {
					mine = append(mine, roleID)
				}
			}
			if len(mine) > 0 {
				if discordID, err := strconv.ParseInt(m.User.ID, 10, 64); err == nil {
					held[discordID] = mine
				}
			}
		}
		if len(batch) < 1000 {
			break
		}
		after = batch[len(batch)-1].User.ID
	}

	for discordID, roleIDs := range held {
		userID, err := identity.EnsureDiscordUser(ctx, discordID, "")
		if err != nil {
			return err
		}
		if err := SyncMember(ctx, userID, discordID, roleIDs); err != nil {
			return err
		}
	}

	// Revoke from anyone previously assigned a linked role who wasn't seen
	// holding one above — they lost the Discord role, or left the guild
	// entirely (GuildMembers only returns current members).
	rows, err := state.Pool.Query(ctx, `
		SELECT DISTINCT da.discord_id, ur.user_id
		FROM user_roles ur
		JOIN roles r ON r.id = ur.role_id
		JOIN discord_accounts da ON da.user_id = ur.user_id
		WHERE r.discord_role_id IS NOT NULL`)
	if err != nil {
		return err
	}
	type assignment struct {
		discordID int64
		userID    uuid.UUID
	}
	var previous []assignment
	for rows.Next() {
		var a assignment
		if err := rows.Scan(&a.discordID, &a.userID); err != nil {
			rows.Close()
			return err
		}
		previous = append(previous, a)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}

	for _, a := range previous {
		if _, ok := held[a.discordID]; ok {
			continue // already reconciled with their current role set above
		}
		if err := SyncMember(ctx, a.userID, a.discordID, nil); err != nil {
			return err
		}
	}

	return nil
}
