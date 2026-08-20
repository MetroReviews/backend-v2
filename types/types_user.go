package types

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID        uuid.UUID `db:"id" json:"id" description:"The user's Metro ID"`
	DiscordID *int64    `db:"discord_id" json:"discord_id" description:"The user's linked Discord ID, if any"`
	Username  *string   `db:"username" json:"username" description:"The user's display username"`
	Avatar    *string   `db:"avatar" json:"avatar" description:"The user's avatar URL"`
	Bio       *string   `db:"bio" json:"bio" description:"The user's profile bio"`
	IsStaff   bool      `db:"is_staff" json:"is_staff" description:"Whether the user can moderate businesses/projects/reviews"`
	Banned    bool      `db:"banned" json:"banned" description:"Whether the user is banned from writing reviews/businesses"`
	CreatedAt time.Time `db:"created_at" json:"created_at" description:"When the account was created — used by the fraud package to throttle brand-new accounts"`
}
