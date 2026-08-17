package commands

import (
	"github.com/MetroReviews/backend-v2/state"
	"github.com/bwmarrin/discordgo"
	"go.uber.org/zap"
)

// cmdSync handles /sync: re-registers Definitions, useful after a deploy
// that changed the command list.
func cmdSync(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if err := Sync(s, state.Config.GuildID()); err != nil {
		state.Logger.Error("[bot] cmdSync failed", zap.Error(err))
		respondText(s, i, "Failed to sync.", true)
		return
	}
	respondText(s, i, "Done syncing", false)
}
