package bot

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/MetroReviews/backend-v2/helpers"
	"github.com/MetroReviews/backend-v2/perms"
	"github.com/MetroReviews/backend-v2/roles"
	"github.com/MetroReviews/backend-v2/state"
	"github.com/bwmarrin/discordgo"
	"go.uber.org/zap"
)

const (
	emoteID     = "<:idemote:912034927443320862>"
	emoteBot    = "<:bot:970349895829561420>"
	emoteCrown  = "<:owner:912356178833596497>"
	emoteInvite = "<:plus:912363980490702918>"
	emoteNote   = "<:activity:912031377422172160>"
)

// AnnounceBotQueue posts the "new bot added to queue" embed, called by the
// POST /bots handler right after a bot is inserted.
func AnnounceBotQueue(botID int64, username, ownerID, invite, reviewNote string) error {
	if state.Discord == nil {
		return nil
	}

	if invite == "" {
		invite = helpers.InviteURL(strconv.FormatInt(botID, 10))
	}
	if reviewNote == "" {
		reviewNote = "No review notes for this bot"
	}

	desc := fmt.Sprintf(
		"%s %d\n%s %s\n%s %s (<@%s>)\n%s [Invite](%s)\n%s %s",
		emoteID, botID,
		emoteBot, username,
		emoteCrown, ownerID, ownerID,
		emoteInvite, invite,
		emoteNote, reviewNote,
	)

	embed := &discordgo.MessageEmbed{
		URL:         fmt.Sprintf("https://metrobots.xyz/bots/%d", botID),
		Title:       "Bot Added To Queue",
		Description: desc,
		Color:       0x2ecc71,
	}

	channelID := strconv.FormatUint(state.Config.QueueChannelID(), 10)
	_, err := state.Discord.ChannelMessageSendComplex(channelID, &discordgo.MessageSend{
		Content: reviewerMentions(),
		Embed:   embed,
	})
	return err
}

// reviewerMentions builds a space-separated "<@&id>" mention for every
// Discord role currently linked to the queue.review permission (see the
// roles/perms packages) — there's no single static "reviewer role" or
// test-ping override in config anymore, so this pings whichever role(s)
// are actually wired up to review the queue, or nobody if none are.
func reviewerMentions() string {
	roleIDs, err := roles.DiscordRoleIDsWithPermission(state.Context, perms.QueueReview)
	if err != nil {
		state.Logger.Error("[bot] failed to load reviewer role IDs for announcement", zap.Error(err))
		return ""
	}

	mentions := make([]string, len(roleIDs))
	for i, id := range roleIDs {
		mentions[i] = fmt.Sprintf("<@&%d>", id)
	}
	return strings.Join(mentions, " ")
}
