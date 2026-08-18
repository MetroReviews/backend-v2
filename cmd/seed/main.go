// Command seed populates the database with sample data for local development.
//
// It is idempotent: re-running it upserts the same fixed rows rather than
// piling up duplicates, so it's safe to run after every schema change.
//
// Usage:
//
//	go run ./cmd/seed                 # seed sample data (default: config.yaml)
//	go run ./cmd/seed -config foo.yaml
//	go run ./cmd/seed -reset -yes     # wipe reviews/businesses/projects/bots/categories/users first, then seed
//
// Split across files by what's being seeded: fixtures.go holds the fixed
// IDs every seed*.go file shares, seed_users.go/seed_catalog.go/seed_bots.go/
// seed_activity.go each seed one slice of the schema. seed_catalog.go seeds
// both businesses and their projects.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/MetroReviews/backend-v2/config"
	"github.com/MetroReviews/backend-v2/migrations"
	"github.com/jackc/pgx/v5/pgxpool"
	"gopkg.in/yaml.v3"
)

func main() {
	configPath := flag.String("config", "config.yaml", "path to config.yaml")
	reset := flag.Bool("reset", false, "truncate reviews, businesses, projects, categories, bots, moderation_actions, claims, reports, discord_accounts and users before seeding")
	yes := flag.Bool("yes", false, "confirm destructive operations (required together with -reset)")
	flag.Parse()

	cfg, err := loadConfig(*configPath)
	if err != nil {
		log.Fatalf("failed to load %s: %v", *configPath, err)
	}
	if cfg.Database.PostgresURL == "" {
		log.Fatalf("postgres_url is not set in %s", *configPath)
	}

	if *reset && !*yes {
		log.Fatal("-reset will delete existing reviews/businesses/projects/bots/users data; re-run with -yes to confirm")
	}

	ctx := context.Background()

	pool, err := pgxpool.New(ctx, cfg.Database.PostgresURL)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		log.Fatalf("failed to ping database: %v", err)
	}

	if err := migrations.Apply(ctx, pool); err != nil {
		log.Fatalf("failed to apply migrations: %v", err)
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		log.Fatalf("failed to begin transaction: %v", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck // no-op once committed

	if *reset {
		log.Println("resetting reviews, review_votes, reports, claims, projects, businesses, categories, moderation_actions, bots, discord_accounts, users ...")
		if _, err := tx.Exec(ctx, `TRUNCATE
			reviews, review_votes, reports, claims, projects, businesses, categories,
			moderation_actions, bots, discord_accounts, users
			RESTART IDENTITY CASCADE`); err != nil {
			log.Fatalf("failed to truncate tables: %v", err)
		}
	}

	if err := seedUsers(ctx, tx); err != nil {
		log.Fatalf("failed to seed users: %v", err)
	}

	if err := seedCategories(ctx, tx); err != nil {
		log.Fatalf("failed to seed categories: %v", err)
	}

	if err := seedBusinesses(ctx, tx); err != nil {
		log.Fatalf("failed to seed businesses: %v", err)
	}

	if err := seedProjects(ctx, tx); err != nil {
		log.Fatalf("failed to seed projects: %v", err)
	}

	if err := seedBots(ctx, tx); err != nil {
		log.Fatalf("failed to seed bots: %v", err)
	}

	if err := seedActions(ctx, tx); err != nil {
		log.Fatalf("failed to seed actions: %v", err)
	}

	if err := seedReviews(ctx, tx); err != nil {
		log.Fatalf("failed to seed reviews: %v", err)
	}

	if err := tx.Commit(ctx); err != nil {
		log.Fatalf("failed to commit: %v", err)
	}

	fmt.Println("seed complete")
	fmt.Println()
	fmt.Println("users: 4 seeded, each with a linked Discord account")
	fmt.Println("categories: 2 seeded (restaurants, software)")
	fmt.Println("businesses: 2 seeded")
	fmt.Println("projects: 2 seeded (pending/approved)")
	fmt.Println("bots: 5 seeded (pending/under_review/approved/denied)")
	fmt.Println("actions: 4 seeded (claim/approve/deny)")
	fmt.Println("reviews: 5 seeded")
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
