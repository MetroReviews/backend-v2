// Package bot runs the Metro Reviews Discord bot session. It only wires
// the session up and forwards events to the bot/commands package, which
// holds every slash command, the /queue review UI, and the legacy prefix
// commands (one file per command).
package bot

import (
	"github.com/MetroReviews/backend-v2/bot/commands"
	"github.com/MetroReviews/backend-v2/state"
	"github.com/bwmarrin/discordgo"
	"go.uber.org/zap"
)

func Setup() error {
	s, err := discordgo.New("Bot " + state.Config.Token)
	if err != nil {
		return err
	}

	s.Identify.Intents = discordgo.IntentsAll

	s.AddHandler(onReady)
	s.AddHandler(onInteraction)
	s.AddHandler(commands.HandleMessage)

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
