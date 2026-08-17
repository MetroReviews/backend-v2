// Command seed populates the database with sample data for local development.
//
// It is idempotent: re-running it upserts the same fixed rows rather than
// piling up duplicates, so it's safe to run after every schema change.
//
// Usage:
//
//	go run ./cmd/seed                 # seed sample data (default: config.yaml)
//	go run ./cmd/seed -config foo.yaml
//	go run ./cmd/seed -reset -yes     # wipe bot_list/bot_queue/bot_action/users first, then seed
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/MetroReviews/backend-v2/config"
	"github.com/google/uuid"
	"github.com/infinitybotlist/eureka/crypto"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"gopkg.in/yaml.v3"
)

// Fixed IDs so reseeding updates the same rows instead of duplicating them.
var (
	listSpaceID = uuid.MustParse("00000000-0000-0000-0000-0000000000a1")
	listFatesID = uuid.MustParse("00000000-0000-0000-0000-0000000000a2")

	actionClaimID   = uuid.MustParse("00000000-0000-0000-0000-0000000000b1")
	actionApproveID = uuid.MustParse("00000000-0000-0000-0000-0000000000b2")
	actionDenyID    = uuid.MustParse("00000000-0000-0000-0000-0000000000b3")

	reviewerID = int64(2200000000000000001)
	ownerAlpha = int64(2100000000000000001)
	ownerBeta  = int64(2100000000000000002)
	ownerExtra = int64(2100000000000000003)
)

func main() {
	configPath := flag.String("config", "config.yaml", "path to config.yaml")
	reset := flag.Bool("reset", false, "truncate bot_list, bot_queue, bot_action and users before seeding")
	yes := flag.Bool("yes", false, "confirm destructive operations (required together with -reset)")
	flag.Parse()

	cfg, err := loadConfig(*configPath)
	if err != nil {
		log.Fatalf("failed to load %s: %v", *configPath, err)
	}
	if cfg.PostgresURL == "" {
		log.Fatalf("postgres_url is not set in %s", *configPath)
	}

	if *reset && !*yes {
		log.Fatal("-reset will delete existing bot_list/bot_queue/bot_action/users data; re-run with -yes to confirm")
	}

	ctx := context.Background()

	pool, err := pgxpool.New(ctx, cfg.PostgresURL)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		log.Fatalf("failed to ping database: %v", err)
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		log.Fatalf("failed to begin transaction: %v", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck // no-op once committed

	if *reset {
		log.Println("resetting bot_action, bot_queue, bot_list, users ...")
		if _, err := tx.Exec(ctx, `TRUNCATE bot_action, bot_queue, bot_list, users RESTART IDENTITY CASCADE`); err != nil {
			log.Fatalf("failed to truncate tables: %v", err)
		}
	}

	spaceKey, err := seedList(ctx, tx, listSpaceID, "Space (Seed)",
		"A sample bot list used for local development.", "space.example.com")
	if err != nil {
		log.Fatalf("failed to seed list Space: %v", err)
	}

	fatesKey, err := seedList(ctx, tx, listFatesID, "Fates List (Seed)",
		"A second sample bot list used to exercise cross-list flows.", "fates.example.com")
	if err != nil {
		log.Fatalf("failed to seed list Fates: %v", err)
	}

	if err := seedUsers(ctx, tx); err != nil {
		log.Fatalf("failed to seed users: %v", err)
	}

	if err := seedBots(ctx, tx); err != nil {
		log.Fatalf("failed to seed bots: %v", err)
	}

	if err := seedActions(ctx, tx); err != nil {
		log.Fatalf("failed to seed actions: %v", err)
	}

	if err := tx.Commit(ctx); err != nil {
		log.Fatalf("failed to commit: %v", err)
	}

	fmt.Println("seed complete")
	fmt.Println()
	fmt.Println("lists:")
	fmt.Printf("  Space (Seed)       id=%s secret_key=%s\n", listSpaceID, spaceKey)
	fmt.Printf("  Fates List (Seed)  id=%s secret_key=%s\n", listFatesID, fatesKey)
	fmt.Println()
	fmt.Println("bots: 5 seeded in bot_queue (pending/under_review/approved/denied)")
	fmt.Println("actions: 3 seeded in bot_action (claim/approve/deny)")
	fmt.Println("users: 2 seeded")
}

func loadConfig(path string) (*config.Config, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	cfg := &config.Config{}
	if err := yaml.Unmarshal(b, cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

// seedList upserts a bot_list row. secret_key is intentionally left out of
// the UPDATE SET clause so re-running the seed doesn't rotate existing keys;
// RETURNING always yields the key that's actually persisted.
func seedList(ctx context.Context, tx pgx.Tx, id uuid.UUID, name, description, domain string) (string, error) {
	var secretKey string
	err := tx.QueryRow(ctx, `
		INSERT INTO bot_list (
			id, state, name, description, icon,
			claim_bot_api, unclaim_bot_api, approve_bot_api, deny_bot_api,
			domain, secret_key
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (id) DO UPDATE SET
			state = EXCLUDED.state,
			name = EXCLUDED.name,
			description = EXCLUDED.description,
			icon = EXCLUDED.icon,
			claim_bot_api = EXCLUDED.claim_bot_api,
			unclaim_bot_api = EXCLUDED.unclaim_bot_api,
			approve_bot_api = EXCLUDED.approve_bot_api,
			deny_bot_api = EXCLUDED.deny_bot_api,
			domain = EXCLUDED.domain
		RETURNING secret_key`,
		id, 1 /* ListStateSupported */, name, description,
		"https://"+domain+"/icon.png",
		"https://"+domain+"/api/claim",
		"https://"+domain+"/api/unclaim",
		"https://"+domain+"/api/approve",
		"https://"+domain+"/api/deny",
		domain,
		crypto.RandString(43),
	).Scan(&secretKey)
	return secretKey, err
}

func seedUsers(ctx context.Context, tx pgx.Tx) error {
	users := []struct {
		id    int64
		nonce string
	}{
		{ownerAlpha, crypto.RandString(20)},
		{ownerBeta, crypto.RandString(20)},
	}
	for _, u := range users {
		if _, err := tx.Exec(ctx, `
			INSERT INTO users (user_id, nonce) VALUES ($1, $2)
			ON CONFLICT (user_id) DO NOTHING`, u.id, u.nonce); err != nil {
			return err
		}
	}
	return nil
}

func seedBots(ctx context.Context, tx pgx.Tx) error {
	type bot struct {
		botID           int64
		username        string
		description     string
		longDescription string
		website         string
		support         string
		library         string
		nsfw            bool
		prefix          string
		tags            []string
		reviewNote      *string
		invite          string
		state           int
		listSource      uuid.UUID
		owner           int64
		extraOwners     []int64
		reviewer        *int64
		crossAdd        bool
	}

	bots := []bot{
		{
			botID:           1100000000000000001,
			username:        "SeedBot Alpha",
			description:     "A demo moderation bot used as seed data.",
			longDescription: "## SeedBot Alpha\n\nHandles auto-moderation, logging and warnings. This entry is seed data for local development.",
			website:         "https://example.com/alpha",
			support:         "https://discord.gg/example-alpha",
			library:         "discord.js",
			prefix:          "!",
			tags:            []string{"moderation", "logging"},
			invite:          "https://discord.com/oauth2/authorize?client_id=1100000000000000001&scope=bot",
			state:           0, // Pending
			listSource:      listSpaceID,
			owner:           ownerAlpha,
			crossAdd:        true,
		},
		{
			botID:           1100000000000000002,
			username:        "SeedBot Beta",
			description:     "A demo music bot used as seed data.",
			longDescription: "## SeedBot Beta\n\nStreams music from various sources. This entry is seed data for local development.",
			website:         "https://example.com/beta",
			support:         "https://discord.gg/example-beta",
			library:         "discord.py",
			prefix:          "b!",
			tags:            []string{"music"},
			invite:          "https://discord.com/oauth2/authorize?client_id=1100000000000000002&scope=bot",
			state:           1, // UnderReview
			listSource:      listSpaceID,
			owner:           ownerBeta,
			reviewer:        &reviewerID,
			crossAdd:        true,
		},
		{
			botID:           1100000000000000003,
			username:        "SeedBot Gamma",
			description:     "A demo economy and utility bot used as seed data.",
			longDescription: "## SeedBot Gamma\n\nAdds an economy system alongside general utility commands. This entry is seed data for local development.",
			website:         "https://example.com/gamma",
			support:         "https://discord.gg/example-gamma",
			library:         "serenity",
			prefix:          "/",
			tags:            []string{"utility", "economy"},
			invite:          "https://discord.com/oauth2/authorize?client_id=1100000000000000003&scope=bot",
			state:           2, // Approved
			listSource:      listFatesID,
			owner:           ownerAlpha,
			extraOwners:     []int64{ownerExtra},
			reviewer:        &reviewerID,
			crossAdd:        true,
		},
		{
			botID:           1100000000000000004,
			username:        "SeedBot Delta",
			description:     "A demo NSFW bot used as seed data.",
			longDescription: "## SeedBot Delta\n\nDenied during review for missing a privacy policy. This entry is seed data for local development.",
			website:         "https://example.com/delta",
			library:         "discord.js",
			nsfw:            true,
			prefix:          "d!",
			tags:            []string{"nsfw"},
			reviewNote:      strPtr("Missing privacy policy"),
			invite:          "https://discord.com/oauth2/authorize?client_id=1100000000000000004&scope=bot",
			state:           3, // Denied
			listSource:      listFatesID,
			owner:           ownerBeta,
			reviewer:        &reviewerID,
			crossAdd:        false,
		},
		{
			botID:           1100000000000000005,
			username:        "SeedBot Epsilon",
			description:     "A second demo pending bot used as seed data.",
			longDescription: "## SeedBot Epsilon\n\nA freshly submitted bot awaiting review. This entry is seed data for local development.",
			website:         "https://example.com/epsilon",
			support:         "https://discord.gg/example-epsilon",
			library:         "discord.js",
			prefix:          "e.",
			tags:            []string{"leveling", "economy"},
			invite:          "https://discord.com/oauth2/authorize?client_id=1100000000000000005&scope=bot",
			state:           0, // Pending
			listSource:      listSpaceID,
			owner:           ownerExtra,
			crossAdd:        false,
		},
	}

	for _, b := range bots {
		extraOwners := b.extraOwners
		if extraOwners == nil {
			extraOwners = []int64{}
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO bot_queue (
				bot_id, username, banner, description, long_description,
				website, support, donate, library, nsfw, prefix, tags,
				review_note, invite, state, list_source, owner, extra_owners,
				reviewer, invite_link, cross_add
			) VALUES (
				$1, $2, NULL, $3, $4,
				$5, $6, NULL, $7, $8, $9, $10,
				$11, $12, $13, $14, $15, $16,
				$17, NULL, $18
			)
			ON CONFLICT (bot_id) DO UPDATE SET
				username = EXCLUDED.username,
				description = EXCLUDED.description,
				long_description = EXCLUDED.long_description,
				website = EXCLUDED.website,
				support = EXCLUDED.support,
				library = EXCLUDED.library,
				nsfw = EXCLUDED.nsfw,
				prefix = EXCLUDED.prefix,
				tags = EXCLUDED.tags,
				review_note = EXCLUDED.review_note,
				invite = EXCLUDED.invite,
				state = EXCLUDED.state,
				list_source = EXCLUDED.list_source,
				owner = EXCLUDED.owner,
				extra_owners = EXCLUDED.extra_owners,
				reviewer = EXCLUDED.reviewer,
				cross_add = EXCLUDED.cross_add`,
			b.botID, b.username, b.description, b.longDescription,
			b.website, b.support, b.library, b.nsfw, b.prefix, b.tags,
			b.reviewNote, b.invite, b.state, b.listSource, b.owner, extraOwners,
			b.reviewer, b.crossAdd,
		); err != nil {
			return fmt.Errorf("bot %d: %w", b.botID, err)
		}
	}
	return nil
}

func seedActions(ctx context.Context, tx pgx.Tx) error {
	actions := []struct {
		id         uuid.UUID
		botID      int64
		action     int
		reason     string
		listSource uuid.UUID
	}{
		{actionClaimID, 1100000000000000002, 0 /* Claim */, "Claimed for review", listSpaceID},
		{actionApproveID, 1100000000000000003, 2 /* Approve */, "Meets all listing requirements", listFatesID},
		{actionDenyID, 1100000000000000004, 3 /* Deny */, "Missing privacy policy", listFatesID},
	}

	for _, a := range actions {
		if _, err := tx.Exec(ctx, `
			INSERT INTO bot_action (id, bot_id, action, reason, reviewer, list_source)
			VALUES ($1, $2, $3, $4, $5, $6)
			ON CONFLICT (id) DO UPDATE SET
				bot_id = EXCLUDED.bot_id,
				action = EXCLUDED.action,
				reason = EXCLUDED.reason,
				reviewer = EXCLUDED.reviewer,
				list_source = EXCLUDED.list_source`,
			a.id, a.botID, a.action, a.reason, fmt.Sprintf("%d", reviewerID), a.listSource,
		); err != nil {
			return fmt.Errorf("action %s: %w", a.id, err)
		}
	}
	return nil
}

func strPtr(s string) *string { return &s }
