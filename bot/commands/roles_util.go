package commands

import (
	"errors"

	"github.com/MetroReviews/backend-v2/state"
	"github.com/bwmarrin/discordgo"
	"github.com/jackc/pgx/v5/pgconn"
	"go.uber.org/zap"
)

func ptrInt(v int) *int { return &v }

// isUniqueViolation reports whether err is a Postgres unique-constraint
// failure — the two ways a role edit can be rejected: a duplicate role
// name, or a Discord role that's already linked to a different local role.
func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

// followupError sends an ephemeral error message alongside an already-
// acknowledged interaction — callers still redraw the screen separately
// afterward, this just explains why nothing changed.
func followupError(s *discordgo.Session, i *discordgo.InteractionCreate, msg string) {
	if _, err := s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
		Content: msg,
		Flags:   discordgo.MessageFlagsEphemeral,
	}); err != nil {
		state.Logger.Error("[bot] failed to send role error followup", zap.Error(err))
	}
}
