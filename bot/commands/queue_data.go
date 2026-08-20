package commands

import (
	"sort"

	"github.com/MetroReviews/backend-v2/state"
	"github.com/MetroReviews/backend-v2/types"
)

func fetchQueueEntries(showAll bool, filter string) ([]queueEntry, error) {
	var out []queueEntry
	var err error

	switch filter {
	case "project":
		out, err = fetchProjectEntries(showAll)
	default:
		out, err = fetchBusinessEntries(showAll)
	}
	if err != nil {
		return nil, err
	}

	sort.Slice(out, func(a, b int) bool {
		if out[a].state != out[b].state {
			return out[a].state < out[b].state
		}
		return out[a].name < out[b].name
	})

	return out, nil
}

func fetchBusinessEntries(showAll bool) ([]queueEntry, error) {
	query := `
		SELECT l.id, l.name, su.username, sda.discord_id, c.slug, l.status
		FROM businesses l
		JOIN categories c ON c.id = l.category_id
		JOIN users su ON su.id = l.submitted_by
		LEFT JOIN discord_accounts sda ON sda.user_id = su.id`
	var args []any
	if !showAll {
		query += " WHERE l.status = $1 OR l.status = $2"
		args = []any{types.StatePending, types.StateUnderReview}
	}

	rows, err := state.Pool.Query(state.Context, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []queueEntry
	for rows.Next() {
		var id, categorySlug, name string
		var submittedByUsername *string
		var submittedByDiscordID *int64
		var st types.State
		if err := rows.Scan(&id, &name, &submittedByUsername, &submittedByDiscordID, &categorySlug, &st); err != nil {
			continue
		}
		out = append(out, queueEntry{
			subjectType: "business", id: id, name: name,
			submittedBy: mentionOrName(submittedByDiscordID, submittedByUsername), extra: categorySlug, state: st,
		})
	}
	return out, rows.Err()
}

func fetchProjectEntries(showAll bool) ([]queueEntry, error) {
	query := `
		SELECT p.id, p.title, su.username, sda.discord_id, b.name, p.status
		FROM projects p
		JOIN businesses b ON b.id = p.business_id
		JOIN users su ON su.id = p.submitted_by
		LEFT JOIN discord_accounts sda ON sda.user_id = su.id`
	var args []any
	if !showAll {
		query += " WHERE p.status = $1 OR p.status = $2"
		args = []any{types.StatePending, types.StateUnderReview}
	}

	rows, err := state.Pool.Query(state.Context, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []queueEntry
	for rows.Next() {
		var id, businessName, title string
		var submittedByUsername *string
		var submittedByDiscordID *int64
		var st types.State
		if err := rows.Scan(&id, &title, &submittedByUsername, &submittedByDiscordID, &businessName, &st); err != nil {
			continue
		}
		out = append(out, queueEntry{
			subjectType: "project", id: id, name: title,
			submittedBy: mentionOrName(submittedByDiscordID, submittedByUsername), extra: businessName, state: st,
		})
	}
	return out, rows.Err()
}

func queuePageCount(total int) int {
	if total == 0 {
		return 1
	}
	return (total + queuePageSize - 1) / queuePageSize
}

func queuePageSlice(entries []queueEntry, page int) []queueEntry {
	start := page * queuePageSize
	if start >= len(entries) {
		return nil
	}
	end := min(start+queuePageSize, len(entries))
	return entries[start:end]
}
