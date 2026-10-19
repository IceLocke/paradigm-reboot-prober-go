package repository

import (
	"cmp"
	"errors"
	"fmt"
	"paradigm-reboot-prober-go/config"
	"paradigm-reboot-prober-go/internal/model"
	"paradigm-reboot-prober-go/pkg/rating"
	"slices"
	"time"

	"github.com/jellydator/ttlcache/v3"
	"gorm.io/gorm"
)

type RecordRepository struct {
	db    *gorm.DB
	cache *repoCache
}

func NewRecordRepository(db *gorm.DB) *RecordRepository {
	return &RecordRepository{
		db:    db,
		cache: newRepoCache(RecordCacheTTL),
	}
}

// applyRecordFilter applies optional level range, difficulty, and season (B15) filters
// to a GORM query that has Chart and Chart.Song joined.
func applyRecordFilter(query *gorm.DB, filter model.RecordFilter) *gorm.DB {
	if filter.MinLevel != nil {
		query = query.Where(`"Chart".level >= ?`, *filter.MinLevel)
	}
	if filter.MaxLevel != nil {
		query = query.Where(`"Chart".level <= ?`, *filter.MaxLevel)
	}
	if len(filter.Difficulties) > 0 {
		query = query.Where(`"Chart".difficulty IN ?`, filter.Difficulties)
	}
	if filter.B15 != nil {
		query = query.Where(`"Chart__Song".b15 = ?`, *filter.B15)
	}
	return query
}

// bestRecordsQuery keeps official and fitted rankings, and their counts, on the
// same live per-chart candidate pool. GORM does not scope manually joined tables.
func (r *RecordRepository) bestRecordsQuery(username string, filter model.RecordFilter) *gorm.DB {
	query := r.db.Model(&model.PlayRecord{}).
		Joins("JOIN best_play_records ON best_play_records.play_record_id = play_records.id AND best_play_records.deleted_at IS NULL").
		Joins("Chart").
		Joins("Chart.Song").
		Where("best_play_records.username = ?", username).
		Where(`"Chart".id IS NOT NULL AND "Chart__Song".id IS NOT NULL`)
	return applyRecordFilter(query, filter)
}

// applyCountFilter applies optional level range, difficulty, and season (B15) filters
// to a count query, joining the charts and songs tables only when needed.
func applyCountFilter(query *gorm.DB, filter model.RecordFilter, chartIDColumn string) *gorm.DB {
	if filter.IsEmpty() {
		return query
	}
	query = query.Joins(fmt.Sprintf("JOIN charts ON charts.id = %s", chartIDColumn))
	if filter.MinLevel != nil {
		query = query.Where("charts.level >= ?", *filter.MinLevel)
	}
	if filter.MaxLevel != nil {
		query = query.Where("charts.level <= ?", *filter.MaxLevel)
	}
	if len(filter.Difficulties) > 0 {
		query = query.Where("charts.difficulty IN ?", filter.Difficulties)
	}
	if filter.B15 != nil {
		query = query.Joins("JOIN songs ON songs.id = charts.song_id").
			Where("songs.b15 = ?", *filter.B15)
	}
	return query
}

// invalidateUserRecords removes all cached record entries for a given username.
func (r *RecordRepository) invalidateUserRecords(username string) {
	if r.cache != nil {
		invalidateByPrefix(r.cache, username+":")
	}
}

// CreateRecord creates a new play record and updates the best record if necessary
func (r *RecordRepository) CreateRecord(record *model.PlayRecord, isReplaced bool) (*model.PlayRecord, error) {
	var result *model.PlayRecord
	err := r.db.Transaction(func(tx *gorm.DB) error {
		var txErr error
		result, txErr = r.createRecordInTx(tx, record, isReplaced)
		return txErr
	})
	// Invalidate all cached records for this user after successful TX
	if err == nil {
		r.invalidateUserRecords(record.Username)
	}
	return result, err
}

// BatchCreateRecords creates multiple play records atomically in a single transaction
func (r *RecordRepository) BatchCreateRecords(records []*model.PlayRecord, isReplaced bool) ([]*model.PlayRecord, error) {
	var results []*model.PlayRecord
	err := r.db.Transaction(func(tx *gorm.DB) error {
		for _, record := range records {
			savedRecord, err := r.createRecordInTx(tx, record, isReplaced)
			if err != nil {
				return err
			}
			results = append(results, savedRecord)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	// Invalidate cached records for all affected users
	invalidated := make(map[string]bool)
	for _, record := range records {
		if !invalidated[record.Username] {
			r.invalidateUserRecords(record.Username)
			invalidated[record.Username] = true
		}
	}
	return results, nil
}

// createRecordInTx handles creating a single record within an existing transaction.
func (r *RecordRepository) createRecordInTx(tx *gorm.DB, record *model.PlayRecord, isReplaced bool) (*model.PlayRecord, error) {
	var chart model.Chart
	if err := tx.Where("id = ?", record.ChartID).First(&chart).Error; err != nil {
		return nil, errors.New("chart does not exist")
	}

	// Calculate rating
	calculatedRating := rating.SingleRating(chart.Level, *record.Score)
	record.Rating = calculatedRating
	record.RecordTime = time.Now()

	if err := tx.Create(record).Error; err != nil {
		return nil, err
	}

	// Upsert best record: INSERT if not exists, or UPDATE if the new score is
	// higher (or isReplaced is true). This is atomic and race-condition-free,
	// leveraging the unique index idx_best_user_chart(username, chart_id).
	if err := tx.Exec(`
		INSERT INTO best_play_records (username, chart_id, play_record_id)
		VALUES (?, ?, ?)
		ON CONFLICT (username, chart_id) DO UPDATE
		  SET play_record_id = EXCLUDED.play_record_id
		  WHERE ? OR (SELECT score FROM play_records WHERE id = best_play_records.play_record_id) < ?`,
		record.Username, record.ChartID, record.ID, isReplaced, *record.Score,
	).Error; err != nil {
		return nil, err
	}

	return record, nil
}

// GetBest50Records ranks the current per-chart best records in each season, then applies display order.
// Fitting queries rank the full candidate pool before applying the season limits.
func (r *RecordRepository) GetBest50Records(username string, underflow int, sortBy model.RecordSort, descending bool, filter model.RecordFilter) ([]model.PlayRecord, []model.PlayRecord, error) {
	sortBy = sortBy.NormalizeForScope(model.RecordScopeB50)
	key := b50CacheKey(username, underflow, sortBy, descending, filter)
	if r.cache != nil {
		if item := r.cache.Get(key); item != nil {
			entry := item.Value().(*b50CacheEntry)
			b35 := make([]model.PlayRecord, len(entry.B35))
			copy(b35, entry.B35)
			b15 := make([]model.PlayRecord, len(entry.B15))
			copy(b15, entry.B15)
			return b35, b15, nil
		}
	}

	if sortBy == model.RecordSortFittingRating {
		// The fitting binary publishes directly to charts. Do not cache these results,
		// since its writes cannot invalidate this process's repository cache.
		records, err := r.getFittingBestRecords(username, filter)
		if err != nil {
			return nil, nil, err
		}
		var b35, b15 []model.PlayRecord
		for _, record := range records {
			if record.Chart == nil || record.Chart.Song == nil {
				continue
			}
			if record.Chart.Song.B15 {
				if len(b15) < config.GlobalConfig.Game.B15Limit+underflow {
					b15 = append(b15, record)
				}
			} else if len(b35) < config.GlobalConfig.Game.B35Limit+underflow {
				b35 = append(b35, record)
			}
		}
		if !descending {
			slices.Reverse(b35)
			slices.Reverse(b15)
		}
		return b35, b15, nil
	}

	var b35 []model.PlayRecord
	var b15 []model.PlayRecord

	// Base query for best records
	baseQuery := r.bestRecordsQuery(username, filter)

	// B35: Not B15 songs
	if err := baseQuery.Session(&gorm.Session{}).
		Where(`"Chart__Song".b15 = ?`, false).
		Order("rating desc, play_records.id desc").
		Limit(config.GlobalConfig.Game.B35Limit + underflow).
		Find(&b35).Error; err != nil {
		return nil, nil, err
	}

	// B15: B15 songs
	if err := baseQuery.Session(&gorm.Session{}).
		Where(`"Chart__Song".b15 = ?`, true).
		Order("rating desc, play_records.id desc").
		Limit(config.GlobalConfig.Game.B15Limit + underflow).
		Find(&b15).Error; err != nil {
		return nil, nil, err
	}

	if !descending {
		slices.Reverse(b35)
		slices.Reverse(b15)
	}
	if r.cache != nil {
		r.cache.Set(key, &b50CacheEntry{B35: b35, B15: b15}, ttlcache.DefaultTTL)
	}
	return b35, b15, nil
}

// GetAllRecords retrieves all records for a user with pagination and sorting
func (r *RecordRepository) GetAllRecords(username string, pageSize, pageIndex int, sortBy model.RecordSort, order bool, filter model.RecordFilter) ([]model.PlayRecord, error) {
	var records []model.PlayRecord
	query := r.db.Where("username = ?", username).
		Joins("Chart").
		Joins("Chart.Song")
	query = applyRecordFilter(query, filter)

	orderStr := string(model.SortOrderDesc)
	if !order {
		orderStr = string(model.SortOrderAsc)
	}

	// Normalize to a declared SQL-backed sort field before building ORDER BY.
	safeSortBy := string(sortBy.NormalizeForScope(model.RecordScopeAll))
	query = query.Order(safeSortBy + " " + orderStr)

	// pageIndex is 0-indexed from the service layer
	err := query.Offset(pageSize * pageIndex).Limit(pageSize).Find(&records).Error
	return records, err
}

// GetBestRecords retrieves the best records for a user with pagination and sorting
func (r *RecordRepository) GetBestRecords(username string, pageSize, pageIndex int, sortBy model.RecordSort, order bool, filter model.RecordFilter) ([]model.PlayRecord, error) {
	if sortBy == model.RecordSortFittingRating {
		records, err := r.getFittingBestRecords(username, filter)
		if err != nil {
			return nil, err
		}
		if !order {
			slices.Reverse(records)
		}
		// Paginate only after ranking the complete per-chart best-record pool.
		if pageSize <= 0 || pageIndex < 0 || pageIndex > len(records)/pageSize {
			return []model.PlayRecord{}, nil
		}
		start := pageSize * pageIndex
		end := start + min(pageSize, len(records)-start)
		return records[start:end], nil
	}

	var records []model.PlayRecord
	query := r.bestRecordsQuery(username, filter)

	orderStr := string(model.SortOrderDesc)
	if !order {
		orderStr = string(model.SortOrderAsc)
	}

	// Normalize to a declared SQL-backed sort field before building ORDER BY.
	safeSortBy := string(sortBy.NormalizeForScope(model.RecordScopeAll))
	// Break ties by record ID so paginating the full best-record pool is stable.
	query = query.Order(safeSortBy + " " + orderStr + ", play_records.id " + orderStr)

	// pageIndex is 0-indexed from the service layer
	err := query.Offset(pageSize * pageIndex).Limit(pageSize).Find(&records).Error
	return records, err
}

// getFittingBestRecords reuses the Go rating formula so SQLite and PostgreSQL
// return identical integer ratings without a second stored rating or SQL formula.
func (r *RecordRepository) getFittingBestRecords(username string, filter model.RecordFilter) ([]model.PlayRecord, error) {
	var records []model.PlayRecord
	if err := r.bestRecordsQuery(username, filter).Find(&records).Error; err != nil {
		return nil, err
	}
	// Compute once per candidate rather than once per sort comparison.
	ratings := make(map[int]int, len(records))
	for i := range records {
		ratings[records[i].ID] = records[i].FittingRating()
	}
	slices.SortFunc(records, func(a, b model.PlayRecord) int {
		if order := cmp.Compare(ratings[b.ID], ratings[a.ID]); order != 0 {
			return order
		}
		return cmp.Compare(b.ID, a.ID)
	})
	return records, nil
}

// GetAllChartsWithBestScores retrieves all charts with the user's best score (if any)
func (r *RecordRepository) GetAllChartsWithBestScores(username string, filter model.RecordFilter) ([]model.ChartWithScore, error) {
	key := allChartsCacheKey(username, filter)
	if r.cache != nil {
		if item := r.cache.Get(key); item != nil {
			original := item.Value().([]model.ChartWithScore)
			cp := make([]model.ChartWithScore, len(original))
			copy(cp, original)
			return cp, nil
		}
	}

	var results []model.ChartWithScore

	query := r.db.Table("charts").
		Select("charts.id, COALESCE(charts.override_title, songs.title) as title, COALESCE(charts.override_version, songs.version) as version, charts.difficulty, charts.level, COALESCE(play_records.score, 0) as score").
		Joins("JOIN songs ON charts.song_id = songs.id").
		Joins("LEFT JOIN play_records ON charts.id = play_records.chart_id AND play_records.username = ?", username).
		Joins("LEFT JOIN best_play_records ON play_records.id = best_play_records.play_record_id").
		Where("play_records.id IS NULL OR best_play_records.play_record_id IS NOT NULL")

	// Apply filters directly on charts / songs tables
	if filter.MinLevel != nil {
		query = query.Where("charts.level >= ?", *filter.MinLevel)
	}
	if filter.MaxLevel != nil {
		query = query.Where("charts.level <= ?", *filter.MaxLevel)
	}
	if len(filter.Difficulties) > 0 {
		query = query.Where("charts.difficulty IN ?", filter.Difficulties)
	}
	if filter.B15 != nil {
		query = query.Where("songs.b15 = ?", *filter.B15)
	}

	err := query.Scan(&results).Error
	if err != nil {
		return results, err
	}

	if r.cache != nil {
		r.cache.Set(key, results, ttlcache.DefaultTTL)
	}
	return results, nil
}

// CountBestRecords counts the number of best records for a user
func (r *RecordRepository) CountBestRecords(username string, filter model.RecordFilter) (int64, error) {
	var count int64
	err := r.bestRecordsQuery(username, filter).Count(&count).Error
	return count, err
}

// CountAllRecords counts the total number of records for a user
func (r *RecordRepository) CountAllRecords(username string, filter model.RecordFilter) (int64, error) {
	var count int64
	query := r.db.Model(&model.PlayRecord{}).
		Where("play_records.username = ?", username)
	query = applyCountFilter(query, filter, "play_records.chart_id")
	err := query.Count(&count).Error
	return count, err
}

// GetBestRecordsBySong retrieves the best record per difficulty for a specific song
func (r *RecordRepository) GetBestRecordsBySong(username string, songID int) ([]model.PlayRecord, error) {
	key := bestSongCacheKey(username, songID)
	if r.cache != nil {
		if item := r.cache.Get(key); item != nil {
			original := item.Value().([]model.PlayRecord)
			cp := make([]model.PlayRecord, len(original))
			copy(cp, original)
			return cp, nil
		}
	}

	var records []model.PlayRecord
	err := r.db.Model(&model.PlayRecord{}).
		Joins("JOIN best_play_records ON best_play_records.play_record_id = play_records.id").
		Joins("Chart").
		Joins("Chart.Song").
		Where(`play_records.username = ? AND "Chart".song_id = ?`, username, songID).
		Order("rating desc").
		Find(&records).Error
	if err != nil {
		return records, err
	}

	if r.cache != nil {
		r.cache.Set(key, records, ttlcache.DefaultTTL)
	}
	return records, err
}

// GetAllRecordsBySong retrieves all records for a specific song with pagination and sorting
func (r *RecordRepository) GetAllRecordsBySong(username string, songID int, pageSize, pageIndex int, sortBy model.RecordSort, order bool) ([]model.PlayRecord, error) {
	var records []model.PlayRecord
	query := r.db.Where("play_records.username = ?", username).
		Joins("Chart").
		Joins("Chart.Song").
		Where(`"Chart".song_id = ?`, songID)

	orderStr := string(model.SortOrderDesc)
	if !order {
		orderStr = string(model.SortOrderAsc)
	}

	safeSortBy := string(sortBy.NormalizeForScope(model.RecordScopeAll))
	query = query.Order(safeSortBy + " " + orderStr)

	err := query.Offset(pageSize * pageIndex).Limit(pageSize).Find(&records).Error
	return records, err
}

// CountAllRecordsBySong counts the total number of records for a specific song
func (r *RecordRepository) CountAllRecordsBySong(username string, songID int) (int64, error) {
	var count int64
	err := r.db.Model(&model.PlayRecord{}).
		Joins("JOIN charts ON charts.id = play_records.chart_id").
		Where("play_records.username = ? AND charts.song_id = ?", username, songID).
		Count(&count).Error
	return count, err
}

// GetBestRecordByChart retrieves the best record for a specific chart
func (r *RecordRepository) GetBestRecordByChart(username string, chartID int) (*model.PlayRecord, error) {
	key := bestChartCacheKey(username, chartID)
	if r.cache != nil {
		if item := r.cache.Get(key); item != nil {
			original := item.Value().(*model.PlayRecord)
			cp := *original
			return &cp, nil
		}
	}

	var record model.PlayRecord
	err := r.db.Model(&model.PlayRecord{}).
		Joins("JOIN best_play_records ON best_play_records.play_record_id = play_records.id").
		Joins("Chart").
		Joins("Chart.Song").
		Where("play_records.username = ? AND play_records.chart_id = ?", username, chartID).
		First(&record).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	if r.cache != nil {
		r.cache.Set(key, &record, ttlcache.DefaultTTL)
		cp := record
		return &cp, nil
	}
	return &record, nil
}

// GetAllRecordsByChart retrieves all records for a specific chart with pagination and sorting
func (r *RecordRepository) GetAllRecordsByChart(username string, chartID int, pageSize, pageIndex int, sortBy model.RecordSort, order bool) ([]model.PlayRecord, error) {
	var records []model.PlayRecord
	query := r.db.Where("play_records.username = ? AND play_records.chart_id = ?", username, chartID).
		Joins("Chart").
		Joins("Chart.Song")

	orderStr := string(model.SortOrderDesc)
	if !order {
		orderStr = string(model.SortOrderAsc)
	}

	safeSortBy := string(sortBy.NormalizeForScope(model.RecordScopeAll))
	query = query.Order(safeSortBy + " " + orderStr)

	err := query.Offset(pageSize * pageIndex).Limit(pageSize).Find(&records).Error
	return records, err
}

// CountAllRecordsByChart counts the total number of records for a specific chart
func (r *RecordRepository) CountAllRecordsByChart(username string, chartID int) (int64, error) {
	var count int64
	err := r.db.Model(&model.PlayRecord{}).
		Where("username = ? AND chart_id = ?", username, chartID).
		Count(&count).Error
	return count, err
}

// RecalculateRatingsByChart recalculates ratings for all play records of a given chart.
// Designed to be called within an existing transaction when a chart's level changes.
func RecalculateRatingsByChart(tx *gorm.DB, chartID int, newLevel float64) error {
	var records []model.PlayRecord
	if err := tx.Where("chart_id = ?", chartID).Find(&records).Error; err != nil {
		return err
	}
	if len(records) == 0 {
		return nil
	}

	// Group record IDs by their new rating to minimize UPDATE queries.
	// DB ops: 1 SELECT + M UPDATEs (M = distinct rating values, typically << N records)
	ratingGroups := make(map[int][]int) // newRating -> []playRecordID
	for _, r := range records {
		newRating := rating.SingleRating(newLevel, *r.Score)
		ratingGroups[newRating] = append(ratingGroups[newRating], r.ID)
	}
	for newRating, ids := range ratingGroups {
		if err := tx.Model(&model.PlayRecord{}).
			Where("id IN ?", ids).
			Update("rating", newRating).Error; err != nil {
			return err
		}
	}
	return nil
}
