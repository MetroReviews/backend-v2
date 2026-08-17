package commands

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/MetroReviews/backend-v2/state"
	"github.com/bwmarrin/discordgo"
	"go.uber.org/zap"
)

// HandleMessage handles legacy `%`-prefixed text commands (currently just
// %delbot, kept for owners as an emergency escape hatch).
func HandleMessage(s *discordgo.Session, m *discordgo.MessageCreate) {
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
		cmdDelBot(s, m, authorID, fields)
	}
}

func cmdDelBot(s *discordgo.Session, m *discordgo.MessageCreate, authorID int64, fields []string) {
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
		state.Logger.Error("[bot] delbot failed", zap.Int64("bot_id", botID), zap.Error(err))
		s.ChannelMessageSend(m.ChannelID, "Failed to delete bot.")
		return
	}
	s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Success, deleted %d row(s)", tag.RowsAffected()))
}
