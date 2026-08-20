package bot

import (
	"strconv"

	"github.com/MetroReviews/backend-v2/bot/commands"
	"github.com/MetroReviews/backend-v2/helpers"
	"github.com/MetroReviews/backend-v2/identity"
	"github.com/MetroReviews/backend-v2/roles"
	"github.com/MetroReviews/backend-v2/state"
	"github.com/bwmarrin/discordgo"
	"go.uber.org/zap"
)

func Setup() error {
	s, err := discordgo.New("Bot " + state.Config.Discord.Token)
	if err != nil {
		return err
	}

	s.Identify.Intents = discordgo.IntentsAll

	s.AddHandler(onReady)
	s.AddHandler(onInteraction)
	s.AddHandler(onGuildMemberAdd)
	s.AddHandler(onGuildMemberUpdate)

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

	if err := commands.Sync(s, state.Config.GuildID()); err != nil {
		state.Logger.Error("[bot] failed to register guild commands", zap.Error(err))
	}

	go func() {
		if err := roles.SyncGuild(state.Context, s, state.Config.GuildID()); err != nil {
			state.Logger.Error("[bot] initial role sync failed", zap.Error(err))
			return
		}
		state.Logger.Info("[bot] synced roles from the guild")
	}()
}

func onInteraction(s *discordgo.Session, i *discordgo.InteractionCreate) {
	switch i.Type {
	case discordgo.InteractionApplicationCommand:
		commands.HandleCommand(s, i)
	case discordgo.InteractionModalSubmit:
		commands.HandleModal(s, i)
	case discordgo.InteractionMessageComponent:
		commands.HandleComponent(s, i)
	}
}

func onGuildMemberAdd(_ *discordgo.Session, m *discordgo.GuildMemberAdd) {
	if m.GuildID != strconv.FormatUint(state.Config.GuildID(), 10) {
		return
	}
	go syncMemberRoles(m.Member)
}

func onGuildMemberUpdate(_ *discordgo.Session, m *discordgo.GuildMemberUpdate) {
	if m.GuildID != strconv.FormatUint(state.Config.GuildID(), 10) {
		return
	}
	go syncMemberRoles(m.Member)
}

func syncMemberRoles(m *discordgo.Member) {
	if m == nil || m.User == nil {
		return
	}
	discordID, err := strconv.ParseInt(m.User.ID, 10, 64)
	if err != nil {
		return
	}

	linked, err := roles.LinkedDiscordRoleIDs(state.Context)
	if err != nil {
		state.Logger.Error("[bot] failed to load linked roles", zap.Error(err))
		return
	}
	holdsLinked := false
	for _, roleID := range m.Roles {
		if helpers.Contains(linked, roleID) {
			holdsLinked = true
			break
		}
	}

	userID, existed, err := identity.Lookup(state.Context, discordID)
	if err != nil {
		state.Logger.Error("[bot] failed to look up user for role sync", zap.Error(err))
		return
	}
	if !holdsLinked && !existed {
		return
	}
	if !existed {
		userID, err = identity.EnsureDiscordUser(state.Context, discordID, m.User.Username)
		if err != nil {
			state.Logger.Error("[bot] failed to provision user for role sync", zap.Error(err))
			return
		}
	}

	if err := roles.SyncMember(state.Context, userID, discordID, m.Roles); err != nil {
		state.Logger.Error("[bot] failed to sync member roles", zap.Error(err))
	}
}
