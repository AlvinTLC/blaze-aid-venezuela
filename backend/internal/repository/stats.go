package repository

import (
	"context"
	"fmt"
	"time"
)

// StatsCounts returns active-row counts per entity plus events in the last 24h.
func (r *Repository) StatsCounts(ctx context.Context) (map[string]int, error) {
	const q = `SELECT
		(SELECT count(*) FROM aid_projects    WHERE deleted_at IS NULL),
		(SELECT count(*) FROM resources       WHERE deleted_at IS NULL),
		(SELECT count(*) FROM missing_persons WHERE deleted_at IS NULL),
		(SELECT count(*) FROM volunteers      WHERE deleted_at IS NULL),
		(SELECT count(*) FROM events WHERE occurred_at > now() - interval '24 hours')`
	var p, res, m, v, e int
	if err := r.pool.QueryRow(ctx, q).Scan(&p, &res, &m, &v, &e); err != nil {
		return nil, err
	}
	return map[string]int{
		"projects": p, "resources": res, "missing": m, "volunteers": v, "events_24h": e,
	}, nil
}

// statsGroupable whitelists the (table, column) pairs allowed in GroupCount, so
// the interpolated identifiers can never come from user input.
var statsGroupable = map[string]map[string]bool{
	"aid_projects":    {"status": true, "region": true},
	"resources":       {"status": true, "region": true},
	"missing_persons": {"status": true, "last_seen_region": true},
	"volunteers":      {"status": true, "region": true},
}

// GroupCount returns count(*) grouped by a whitelisted column for active rows.
func (r *Repository) GroupCount(ctx context.Context, table, col string) (map[string]int, error) {
	if !statsGroupable[table][col] {
		return nil, fmt.Errorf("not groupable: %s.%s", table, col)
	}
	q := "SELECT COALESCE(NULLIF(" + col + ", ''), 'unknown') AS k, count(*) " +
		"FROM " + table + " WHERE deleted_at IS NULL GROUP BY k"
	rows, err := r.pool.Query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make(map[string]int)
	for rows.Next() {
		var k string
		var n int
		if err := rows.Scan(&k, &n); err != nil {
			return nil, err
		}
		out[k] = n
	}
	return out, rows.Err()
}

// EventsTimeline returns daily event-count buckets over the last `days` days.
func (r *Repository) EventsTimeline(ctx context.Context, days int) (labels []string, counts []int, err error) {
	const q = `
SELECT time_bucket('1 day', occurred_at) AS day, count(*)
FROM events
WHERE occurred_at > now() - make_interval(days => $1)
GROUP BY day ORDER BY day`
	rows, err := r.pool.Query(ctx, q, days)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var day time.Time
		var n int
		if err := rows.Scan(&day, &n); err != nil {
			return nil, nil, err
		}
		labels = append(labels, day.UTC().Format("2006-01-02"))
		counts = append(counts, n)
	}
	return labels, counts, rows.Err()
}
