package main

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func seedCategories(ctx context.Context, tx pgx.Tx) error {
	categories := []struct {
		id          uuid.UUID
		slug, name  string
		description string
		icon        string
	}{
		{categoryRestaurants, "restaurants", "Restaurants", "Places to eat, reviewed by diners.", "https://example.com/icons/restaurants.png"},
		{categorySoftware, "software", "Software & SaaS", "Apps and services, reviewed by their users.", "https://example.com/icons/software.png"},
	}
	for _, c := range categories {
		if _, err := tx.Exec(ctx, `
			INSERT INTO categories (id, slug, name, description, icon)
			VALUES ($1, $2, $3, $4, $5)
			ON CONFLICT (id) DO UPDATE SET
				slug = EXCLUDED.slug, name = EXCLUDED.name,
				description = EXCLUDED.description, icon = EXCLUDED.icon`,
			c.id, c.slug, c.name, c.description, c.icon,
		); err != nil {
			return err
		}
	}
	return nil
}

func seedBusinesses(ctx context.Context, tx pgx.Tx) error {
	nyLat, nyLng := 40.7308, -73.9973

	businesses := []struct {
		id          uuid.UUID
		categoryID  uuid.UUID
		slug, name  string
		description string
		submittedBy uuid.UUID
		ownerID     *uuid.UUID
		status      int
		reviewer    *uuid.UUID
		latitude    *float64
		longitude   *float64
		gallery     []string
		featured    bool
		viewCount   int
	}{

		{businessAlpha, categoryRestaurants, "the-copper-spoon", "The Copper Spoon",
			"A neighborhood bistro used as seed data.", userAlphaID, &userAlphaID, 2, &userReviewerID,
			&nyLat, &nyLng, []string{"https://example.com/gallery/copper-spoon-1.jpg"}, true, 42},

		{businessBeta, categorySoftware, "taskflow", "TaskFlow",
			"A project management SaaS used as seed data.", userBetaID, nil, 0, nil,
			nil, nil, []string{}, false, 0},
	}
	for _, l := range businesses {
		if _, err := tx.Exec(ctx, `
			INSERT INTO businesses (
				id, category_id, slug, name, description, submitted_by, owner_id, status, reviewer,
				latitude, longitude, gallery, featured, view_count
			)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
			ON CONFLICT (id) DO UPDATE SET
				category_id = EXCLUDED.category_id, slug = EXCLUDED.slug, name = EXCLUDED.name,
				description = EXCLUDED.description, submitted_by = EXCLUDED.submitted_by,
				owner_id = EXCLUDED.owner_id, status = EXCLUDED.status, reviewer = EXCLUDED.reviewer,
				latitude = EXCLUDED.latitude, longitude = EXCLUDED.longitude, gallery = EXCLUDED.gallery,
				featured = EXCLUDED.featured, view_count = EXCLUDED.view_count`,
			l.id, l.categoryID, l.slug, l.name, l.description, l.submittedBy, l.ownerID, l.status, l.reviewer,
			l.latitude, l.longitude, l.gallery, l.featured, l.viewCount,
		); err != nil {
			return fmt.Errorf("business %s: %w", l.slug, err)
		}
	}
	return nil
}

func seedProjects(ctx context.Context, tx pgx.Tx) error {
	projects := []struct {
		id          uuid.UUID
		businessID  uuid.UUID
		title       string
		description string
		submittedBy uuid.UUID
		status      int
		reviewer    *uuid.UUID
	}{

		{projectKitchen, businessAlpha, "Kitchen Remodel", "A full kitchen remodel used as seed data.", userAlphaID, 2, &userReviewerID},

		{projectPatio, businessAlpha, "Patio Expansion", "An outdoor seating expansion used as seed data.", userAlphaID, 0, nil},
	}
	for _, p := range projects {
		if _, err := tx.Exec(ctx, `
			INSERT INTO projects (id, business_id, title, description, submitted_by, status, reviewer)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
			ON CONFLICT (id) DO UPDATE SET
				business_id = EXCLUDED.business_id, title = EXCLUDED.title,
				description = EXCLUDED.description, submitted_by = EXCLUDED.submitted_by,
				status = EXCLUDED.status, reviewer = EXCLUDED.reviewer`,
			p.id, p.businessID, p.title, p.description, p.submittedBy, p.status, p.reviewer,
		); err != nil {
			return fmt.Errorf("project %s: %w", p.title, err)
		}
	}
	return nil
}
