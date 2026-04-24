package fitting

import (
	"context"
	"fmt"
	"sort"
	"time"

	"paradigm-reboot-prober-go/internal/model"
)

// PlayerSkill is the memoised per-player skill snapshot that drives the
// proximity and volume weights in ComputeFitting. AvgRating is the mean of a
// player's top-K single-chart ratings (K = min(total best records,
// Params.SkillTopK)), expressed as a float rating (not the int×100 form
// stored in DB). The top-K is taken across the player's whole best_play_records
// pool — i.e. it is b15-agnostic, **not** the B35+B15 "season" B50 the probe
// server computes for display — because a chart fitter cares about players'
// true skill ceiling, not about which song packs they happen to own this season.
type PlayerSkill struct {
	AvgRating  float64 // mean of top-K ratings / 100
	NumRecords int     // total best_play_records count
}

// collectPlayerSkills builds a username → PlayerSkill map by streaming the
// entire best_play_records table once, grouped per user in pages. Pagination
// is keyset-based on username so we never OFFSET over huge tables.
//
// We bypass repository caching on purpose: this is the fitting microservice's
// responsibility, not the probe's, and caching a 50k-entry map would blow the
// cache TTLs and invalidate logic for the main service.
func (r *Runner) collectPlayerSkills(ctx context.Context) (map[string]PlayerSkill, error) {
	skills := make(map[string]PlayerSkill)
	batch := r.cfg.PlayerBatchSize
	if batch <= 0 {
		batch = 500
	}
	lastUsername := ""
	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		// 1. Grab the next page of distinct usernames.
		var usernames []string
		q := r.db.WithContext(ctx).
			Model(&model.BestPlayRecord{}).
			Distinct("username").
			Where("username > ?", lastUsername).
			Order("username ASC").
			Limit(batch)
		if err := q.Pluck("username", &usernames).Error; err != nil {
			return nil, fmt.Errorf("fetch usernames page: %w", err)
		}
		if len(usernames) == 0 {
			break
		}

		// 2. Fetch (username, rating) for this page of users, sorted so we can
		//    group them in a single linear pass.
		type ratingRow struct {
			Username string
			Rating   int
		}
		var rows []ratingRow
		if err := r.db.WithContext(ctx).
			Table("play_records").
			Select("play_records.username AS username, play_records.rating AS rating").
			Joins("JOIN best_play_records ON best_play_records.play_record_id = play_records.id").
			Where("play_records.username IN ?", usernames).
			Where("play_records.deleted_at IS NULL").
			Where("best_play_records.deleted_at IS NULL").
			Order("play_records.username ASC, play_records.rating DESC").
			Scan(&rows).Error; err != nil {
			return nil, fmt.Errorf("fetch ratings batch: %w", err)
		}

		// 3. Linear group-by-user and accumulate skill.
		topK := r.params.SkillTopK
		if topK < 1 {
			topK = 50 // defensive fallback; config validation should keep us out of here
		}
		curUser := ""
		topRatings := make([]int, 0, topK)
		totalCount := 0
		flush := func() {
			if curUser == "" {
				return
			}
			k := topRatings
			if len(k) > topK {
				k = k[:topK]
			}
			sum := 0
			for _, v := range k {
				sum += v
			}
			avg := 0.0
			if len(k) > 0 {
				avg = float64(sum) / float64(len(k)) / 100.0
			}
			skills[curUser] = PlayerSkill{
				AvgRating:  avg,
				NumRecords: totalCount,
			}
		}
		for _, row := range rows {
			if row.Username != curUser {
				flush()
				curUser = row.Username
				topRatings = topRatings[:0]
				totalCount = 0
			}
			totalCount++
			// topRatings keeps only the top-K (rows are already sorted DESC by rating).
			if len(topRatings) < topK {
				topRatings = append(topRatings, row.Rating)
			}
		}
		flush()

		lastUsername = usernames[len(usernames)-1]

		// Brief pause to ease DB pressure. Skipped when BatchPause is 0.
		if r.cfg.BatchPause > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(r.cfg.BatchPause):
			}
		}
	}
	return skills, nil
}

// fetchChartsSorted returns [id, level] for every non-deleted chart, sorted by
// id. We order by id so pagination by chart id gives stable, deterministic
// batches even if the chart table grows between runs.
func (r *Runner) fetchChartsSorted(ctx context.Context) ([]chartRow, error) {
	var rows []chartRow
	if err := r.db.WithContext(ctx).
		Model(&model.Chart{}).
		Select("id, level").
		Order("id ASC").
		Scan(&rows).Error; err != nil {
		return nil, err
	}
	// sort.Slice is a no-op (already sorted) but cheaply guards against
	// drivers that don't preserve ORDER BY on Scan with projections.
	sort.Slice(rows, func(i, j int) bool { return rows[i].ID < rows[j].ID })
	return rows, nil
}

type chartRow struct {
	ID    int
	Level float64
}

// fetchBestSamples pulls (username, score, record_time) for every best record on the
// given charts, joined in memory with the player-skill map. RecordTime is
// translated into Sample.AgeDays (days since now) so the calculator can
// apply the optional sample-age decay weight.
func (r *Runner) fetchBestSamples(
	ctx context.Context,
	chartIDs []int,
	skills map[string]PlayerSkill,
) (map[int][]Sample, error) {
	type row struct {
		ChartID    int
		Username   string
		Score      int
		RecordTime time.Time
	}
	var rows []row
	if err := r.db.WithContext(ctx).
		Table("best_play_records").
		Select("best_play_records.chart_id AS chart_id, best_play_records.username AS username, play_records.score AS score, play_records.record_time AS record_time").
		Joins("JOIN play_records ON play_records.id = best_play_records.play_record_id").
		Where("best_play_records.chart_id IN ?", chartIDs).
		Where("best_play_records.deleted_at IS NULL").
		Where("play_records.deleted_at IS NULL").
		Scan(&rows).Error; err != nil {
		return nil, err
	}

	now := r.now()
	minRecords := r.params.MinPlayerRecords
	out := make(map[int][]Sample, len(chartIDs))
	for _, row := range rows {
		skill, ok := skills[row.Username]
		if !ok {
			continue // player has no skill snapshot (e.g. zero records — impossible here)
		}
		if minRecords > 0 && skill.NumRecords < minRecords {
			continue
		}
		ageDays := 0.0
		if !row.RecordTime.IsZero() {
			ageDays = now.Sub(row.RecordTime).Hours() / 24.0
			if ageDays < 0 {
				// Future-dated record_time (e.g. client clock skew). Treat as fresh.
				ageDays = 0
			}
		}
		out[row.ChartID] = append(out[row.ChartID], Sample{
			Username:      row.Username,
			Score:         row.Score,
			PlayerSkill:   skill.AvgRating,
			PlayerRecords: skill.NumRecords,
			AgeDays:       ageDays,
		})
	}
	return out, nil
}
