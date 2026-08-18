package types

import "github.com/google/uuid"

// User is a Metro account: its own identity, not a Discord account itself.
// DiscordID is whichever Discord account is linked to it (see the identity
// package) — nullable in principle, though every account today is created
// via Discord OAuth so it's always populated in practice.
type User struct {
	ID        uuid.UUID `db:"id" json:"id" description:"The user's Metro ID"`
	DiscordID *int64    `db:"discord_id" json:"discord_id" description:"The user's linked Discord ID, if any"`
	Username  *string   `db:"username" json:"username" description:"The user's display username"`
	Avatar    *string   `db:"avatar" json:"avatar" description:"The user's avatar URL"`
	Bio       *string   `db:"bio" json:"bio" description:"The user's profile bio"`
	IsStaff   bool      `db:"is_staff" json:"is_staff" description:"Whether the user can moderate businesses/bots/reviews"`
	Banned    bool      `db:"banned" json:"banned" description:"Whether the user is banned from writing reviews/businesses"`
}
