package commands

import (
	"errors"

	"github.com/MetroReviews/backend-v2/state"
	"github.com/bwmarrin/discordgo"
	"github.com/jackc/pgx/v5/pgconn"
	"go.uber.org/zap"
)

func ptrInt(v int) *int { return &v }

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

func followupError(s *discordgo.Session, i *discordgo.InteractionCreate, msg string) {
	if _, err := s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
		Content: msg,
		Flags:   discordgo.MessageFlagsEphemeral,
	}); err != nil {
		state.Logger.Error("[bot] failed to send role error followup", zap.Error(err))
	}
}
