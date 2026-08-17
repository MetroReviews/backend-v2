package bot

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/MetroReviews/backend-v2/silverpelt"
	"github.com/MetroReviews/backend-v2/state"
	"github.com/MetroReviews/backend-v2/types"
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

var guildCommands = []*discordgo.ApplicationCommand{
	{
		Name:        "invite",
		Description: "Get a bot's invite link",
		Options: []*discordgo.ApplicationCommandOption{
			{Type: discordgo.ApplicationCommandOptionString, Name: "bot_id", Description: "The bot's ID", Required: true},
		},
	},
	{
		Name:        "queue",
		Description: "Show the bot queue",
		Options: []*discordgo.ApplicationCommandOption{
			{Type: discordgo.ApplicationCommandOptionBoolean, Name: "show_all", Description: "Show all states", Required: false},
		},
	},
	{Name: "sync", Description: "Syncs all commands"},
	{Name: "support", Description: "Show the link to support"},
	{Name: "claim", Description: "Claim a bot"},
	{Name: "unclaim", Description: "Unclaim a bot"},
	{Name: "approve", Description: "Approve a bot"},
	{Name: "deny", Description: "Deny a bot"},
}

func Setup() error {
	s, err := discordgo.New("Bot " + state.Config.Token)
	if err != nil {
		return err
	}

	s.Identify.Intents = discordgo.IntentsAll

	s.AddHandler(onReady)
	s.AddHandler(onInteraction)
	s.AddHandler(onMessage)

	state.Discord = s
	return nil
}

func Open() error {
	return state.Discord.Open()
}

func Close() error {
	if state.Discord == nil {
		return nil
	}
	return state.Discord.Close()
}

func onReady(s *discordgo.Session, _ *discordgo.Ready) {
	state.Logger.Info("[bot] client is now ready and up", zap.String("user", s.State.User.String()))

	gid := strconv.FormatUint(state.Config.GuildID(), 10)
	if _, err := s.ApplicationCommandBulkOverwrite(s.State.User.ID, gid, guildCommands); err != nil {
		state.Logger.Error("[bot] failed to register guild commands", zap.Error(err))
	}
}

func onInteraction(s *discordgo.Session, i *discordgo.InteractionCreate) {
	switch i.Type {
	case discordgo.InteractionApplicationCommand:
		handleCommand(s, i)
	case discordgo.InteractionModalSubmit:
		handleModal(s, i)
	}
}

func handleCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	data := i.ApplicationCommandData()
	switch data.Name {
	case "invite":
		cmdInvite(s, i, data)
	case "queue":
		cmdQueue(s, i, data)
	case "sync":
		cmdSync(s, i)
	case "support":
		respondText(s, i, "https://github.com/MetroReviews/support", false)
	case "claim":
		openReviewModal(s, i, types.ActionClaim)
	case "unclaim":
		openReviewModal(s, i, types.ActionUnclaim)
	case "approve":
		openReviewModal(s, i, types.ActionApprove)
	case "deny":
		openReviewModal(s, i, types.ActionDeny)
	}
}

func cmdInvite(s *discordgo.Session, i *discordgo.InteractionCreate, data discordgo.ApplicationCommandInteractionData) {
	botID := data.Options[0].StringValue()
	if _, err := strconv.ParseInt(botID, 10, 64); err != nil {
		respondText(s, i, "Invalid bot id", false)
		return
	}
	respondText(s, i, inviteURL(botID), false)
}

func cmdSync(s *discordgo.Session, i *discordgo.InteractionCreate) {
	gid := strconv.FormatUint(state.Config.GuildID(), 10)
	if _, err := s.ApplicationCommandBulkOverwrite(s.State.User.ID, gid, guildCommands); err != nil {
		respondText(s, i, "Failed to sync: "+err.Error(), true)
		return
	}
	respondText(s, i, "Done syncing", false)
}

func cmdQueue(s *discordgo.Session, i *discordgo.InteractionCreate, data discordgo.ApplicationCommandInteractionData) {
	// Defer immediately: the queries below can take long enough (cold pool
	// connections, multiple round trips) to blow Discord's 3s ack window,
	// which would otherwise surface as "Unknown interaction" (10062).
	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
	}); err != nil {
		state.Logger.Error("[bot] failed to defer queue response", zap.Error(err))
		return
	}

	showAll := false
	for _, opt := range data.Options {
		if opt.Name == "show_all" {
			showAll = opt.BoolValue()
		}
	}

	statesToShow := []types.State{types.StatePending, types.StateUnderReview, types.StateApproved, types.StateDenied}
	stateNames := map[types.State]string{
		types.StatePending:     "PENDING",
		types.StateUnderReview: "UNDER_REVIEW",
		types.StateApproved:    "APPROVED",
		types.StateDenied:      "DENIED",
	}

	var b strings.Builder
	for _, st := range statesToShow {
		if !showAll && st != types.StatePending && st != types.StateUnderReview {
			continue
		}

		rows, err := state.Pool.Query(state.Context,
			"SELECT bot_id, username FROM bot_queue WHERE state = $1", st)
		if err != nil {
			state.Logger.Error("[bot] cmdQueue query failed", zap.Int("state", int(st)), zap.Error(err))
			continue
		}

		type entry struct {
			id   int64
			name string
		}
		var entries []entry
		for rows.Next() {
			var e entry
			if err := rows.Scan(&e.id, &e.name); err == nil {
				entries = append(entries, e)
			}
		}
		rows.Close()

		if len(entries) == 0 {
			continue
		}

		b.WriteString(fmt.Sprintf("**%s (%d)**\n\n", stateNames[st], len(entries)))
		for _, e := range entries {
			b.WriteString(fmt.Sprintf("%d (%s)\n", e.id, e.name))
		}
		b.WriteString("\n")
	}

	msg := b.String()
	if msg == "" {
		msg = "The queue is empty"
	}

	chunks := chunk(msg, 1900)
	if _, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{Content: &chunks[0]}); err != nil {
		state.Logger.Error("[bot] failed to edit deferred queue response", zap.Error(err))
	}
	for _, c := range chunks[1:] {
		s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{Content: c})
	}
}

func modalID(action types.Action) string {
	switch action {
	case types.ActionClaim:
		return "claim_modal"
	case types.ActionUnclaim:
		return "unclaim_modal"
	case types.ActionApprove:
		return "approve_modal"
	case types.ActionDeny:
		return "deny_modal"
	}
	return "unknown_modal"
}

func actionFromModalID(id string) (types.Action, bool) {
	switch id {
	case "claim_modal":
		return types.ActionClaim, true
	case "unclaim_modal":
		return types.ActionUnclaim, true
	case "approve_modal":
		return types.ActionApprove, true
	case "deny_modal":
		return types.ActionDeny, true
	}
	return 0, false
}

func openReviewModal(s *discordgo.Session, i *discordgo.InteractionCreate, action types.Action) {
	title := map[types.Action]string{
		types.ActionClaim:   "Claim Bot",
		types.ActionUnclaim: "Unclaim Bot",
		types.ActionApprove: "Approve Bot",
		types.ActionDeny:    "Deny Bot",
	}[action]

	components := []discordgo.MessageComponent{
		row(&discordgo.TextInput{
			CustomID: "bot_id", Label: "Bot ID", Style: discordgo.TextInputShort, Required: true,
		}),
	}

	if action != types.ActionClaim {
		components = append(components, row(&discordgo.TextInput{
			CustomID: "reason", Label: "Reason", Style: discordgo.TextInputParagraph,
			Required: true, MinLength: 5, MaxLength: 4000,
		}))
	}

	components = append(components, row(&discordgo.TextInput{
		CustomID: "resend", Label: "Resend to other lists (owner only, T/F)",
		Style: discordgo.TextInputShort, Required: false, Value: "F",
	}))

	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{
			CustomID:   modalID(action),
			Title:      title,
			Components: components,
		},
	})
	if err != nil {
		state.Logger.Error("[bot] failed to open modal", zap.Error(err))
	}
}

func handleModal(s *discordgo.Session, i *discordgo.InteractionCreate) {
	data := i.ModalSubmitData()
	action, ok := actionFromModalID(data.CustomID)
	if !ok {
		return
	}

	if !isReviewer(i) {
		respondText(s, i, "You are not a reviewer", true)
		return
	}

	values := modalValues(data)

	botID, err := strconv.ParseInt(strings.TrimSpace(values["bot_id"]), 10, 64)
	if err != nil {
		respondText(s, i, "Bot ID invalid", false)
		return
	}

	resend := false
	switch strings.ToLower(strings.TrimSpace(values["resend"])) {
	case "t", "true":
		resend = true
	}

	userID := interactionUserID(i)

	if resend && !state.Config.IsOwner(userID) {
		respondText(s, i, "You are not an owner", false)
		return
	}

	reason := "STUB_REASON"
	if action != types.ActionClaim {
		reason = values["reason"]
	}

	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
	}); err != nil {
		state.Logger.Error("[bot] failed to defer modal response", zap.Error(err))
		return
	}

	res := silverpelt.Handle(state.Context, silverpelt.Request{
		BotID:    botID,
		Reason:   reason,
		Resend:   resend,
		Action:   action,
		Reviewer: userID,
	})

	msg := res.ToMsg()
	if len(msg) > 2000 {
		msg = msg[:2000]
	}
	s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{Content: msg})
}

func onMessage(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author == nil || m.Author.Bot {
		return
	}
	if !strings.HasPrefix(m.Content, "%") {
		return
	}

	fields := strings.Fields(strings.TrimPrefix(m.Content, "%"))
	if len(fields) == 0 {
		return
	}

	authorID, _ := strconv.ParseInt(m.Author.ID, 10, 64)

	switch fields[0] {
	case "delbot":
		if !state.Config.IsOwner(authorID) {
			s.ChannelMessageSend(m.ChannelID, "You aren't a owner of Metro...")
			return
		}
		if len(fields) < 2 {
			s.ChannelMessageSend(m.ChannelID, "Usage: %delbot <bot_id>")
			return
		}
		botID, err := strconv.ParseInt(fields[1], 10, 64)
		if err != nil {
			s.ChannelMessageSend(m.ChannelID, "Invalid bot id")
			return
		}
		tag, err := state.Pool.Exec(state.Context, "DELETE FROM bot_queue WHERE bot_id = $1", botID)
		if err != nil {
			s.ChannelMessageSend(m.ChannelID, "Error: "+err.Error())
			return
		}
		s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Success, deleted %d row(s)", tag.RowsAffected()))
	}
}

func AnnounceBotQueue(botID int64, username, ownerID, invite, reviewNote string) error {
	if state.Discord == nil {
		return nil
	}

	if invite == "" {
		invite = inviteURL(strconv.FormatInt(botID, 10))
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
		Content: "<@&" + state.Config.PingRole() + ">",
		Embed:   embed,
	})
	return err
}

func inviteURL(botID string) string {
	return "https://discord.com/oauth2/authorize?client_id=" + botID + "&scope=bot%20applications.commands&permissions=0"
}

func row(c discordgo.MessageComponent) discordgo.MessageComponent {
	return discordgo.ActionsRow{Components: []discordgo.MessageComponent{c}}
}

func respondText(s *discordgo.Session, i *discordgo.InteractionCreate, content string, ephemeral bool) {
	data := &discordgo.InteractionResponseData{Content: content}
	if ephemeral {
		data.Flags = discordgo.MessageFlagsEphemeral
	}
	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: data,
	}); err != nil {
		state.Logger.Error("[bot] failed to respond to interaction", zap.Error(err))
	}
}

func modalValues(data discordgo.ModalSubmitInteractionData) map[string]string {
	out := map[string]string{}
	for _, comp := range data.Components {
		ar, ok := comp.(*discordgo.ActionsRow)
		if !ok {
			continue
		}
		for _, inner := range ar.Components {
			if ti, ok := inner.(*discordgo.TextInput); ok {
				out[ti.CustomID] = ti.Value
			}
		}
	}
	return out
}

func interactionUserID(i *discordgo.InteractionCreate) int64 {
	var idStr string
	if i.Member != nil && i.Member.User != nil {
		idStr = i.Member.User.ID
	} else if i.User != nil {
		idStr = i.User.ID
	}
	id, _ := strconv.ParseInt(idStr, 10, 64)
	return id
}

func isReviewer(i *discordgo.InteractionCreate) bool {
	if state.Config.IsOwner(interactionUserID(i)) {
		return true
	}
	if i.Member == nil {
		return false
	}
	for _, roleID := range i.Member.Roles {
		if roleID == state.Config.Reviewer {
			return true
		}
	}
	return false
}

func chunk(s string, size int) []string {
	if len(s) <= size {
		return []string{s}
	}
	var out []string
	for len(s) > size {
		out = append(out, s[:size])
		s = s[size:]
	}
	if len(s) > 0 {
		out = append(out, s)
	}
	return out
}
