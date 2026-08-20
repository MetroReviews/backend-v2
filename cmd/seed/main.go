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
	reset := flag.Bool("reset", false, "truncate reviews, review_invites, businesses, projects, categories, moderation_actions, claims, reports, discord_accounts, local_accounts, sessions and users before seeding")
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
		log.Fatal("-reset will delete existing reviews/businesses/projects/users data; re-run with -yes to confirm")
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
	defer tx.Rollback(ctx)

	if *reset {
		log.Println("resetting reviews, review_invites, review_votes, reports, claims, projects, businesses, categories, moderation_actions, discord_accounts, local_accounts, sessions, users ...")
		if _, err := tx.Exec(ctx, `TRUNCATE
			reviews, review_invites, review_votes, reports, claims, projects, businesses, categories,
			moderation_actions, discord_accounts, local_accounts, sessions, users
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

	if err := seedActions(ctx, tx); err != nil {
		log.Fatalf("failed to seed actions: %v", err)
	}

	if err := seedReviews(ctx, tx); err != nil {
		log.Fatalf("failed to seed reviews: %v", err)
	}

	if err := seedInvites(ctx, tx); err != nil {
		log.Fatalf("failed to seed invites: %v", err)
	}

	if err := tx.Commit(ctx); err != nil {
		log.Fatalf("failed to commit: %v", err)
	}

	fmt.Println("seed complete")
	fmt.Println()
	fmt.Println("users: 5 seeded — 4 with a linked Discord account (alpha_seed also has a linked password), 1 email/password only")
	fmt.Println(`local accounts: alpha_seed@example.com / local_seed@example.com, password "` + seedPassword + `" — try POST /auth/login/password`)
	fmt.Println("categories: 2 seeded (restaurants, software)")
	fmt.Println("businesses: 2 seeded")
	fmt.Println("projects: 2 seeded (pending/approved)")
	fmt.Println("actions: 1 seeded (approve)")
	fmt.Println("reviews: 4 seeded (1 verified via a redeemed invite, 1 with a photo)")
	fmt.Println("businesses: the-copper-spoon is featured, geo-tagged (lat/lng), has a gallery photo and a view_count")
	fmt.Println(`review invites: 2 seeded against the-copper-spoon — "seed-invite-redeemed-alpha" (redeemed), "seed-invite-pending-alpha" (pending) — try GET /invites/{token}`)
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
