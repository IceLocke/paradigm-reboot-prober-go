package model

// RecordScope selects the record collection returned by a query.
type RecordScope string

const (
	RecordScopeB50       RecordScope = "b50"
	RecordScopeBest      RecordScope = "best"
	RecordScopeAll       RecordScope = "all"
	RecordScopeAllCharts RecordScope = "all-charts"
)

// IsValid reports whether the scope is a declared record scope.
func (scope RecordScope) IsValid() bool {
	switch scope {
	case RecordScopeB50, RecordScopeBest, RecordScopeAll, RecordScopeAllCharts:
		return true
	default:
		return false
	}
}

// IsValidForSongOrChart reports whether the scope supports a song or chart query.
func (scope RecordScope) IsValidForSongOrChart() bool {
	switch scope {
	case RecordScopeBest, RecordScopeAll:
		return true
	default:
		return false
	}
}

// RecordSort selects the rating or record field used for sorting.
type RecordSort string

const (
	RecordSortRating        RecordSort = "rating"
	RecordSortFittingRating RecordSort = "fitting_rating"
	RecordSortScore         RecordSort = "score"
	RecordSortRecordTime    RecordSort = "record_time"
)

// IsValid reports whether the sort is a declared record sort.
func (sortBy RecordSort) IsValid() bool {
	switch sortBy {
	case RecordSortRating, RecordSortFittingRating, RecordSortScore, RecordSortRecordTime:
		return true
	default:
		return false
	}
}

// IsValidForScope reports whether the scope supports sorting by this field.
func (sortBy RecordSort) IsValidForScope(scope RecordScope) bool {
	if !sortBy.IsValid() {
		return false
	}
	switch scope {
	case RecordScopeB50:
		return sortBy == RecordSortRating || sortBy == RecordSortFittingRating
	case RecordScopeBest:
		return true
	case RecordScopeAll:
		return sortBy != RecordSortFittingRating
	default:
		return false
	}
}

// NormalizeForScope falls back to the official rating when the sort is unsupported.
func (sortBy RecordSort) NormalizeForScope(scope RecordScope) RecordSort {
	if sortBy.IsValidForScope(scope) {
		return sortBy
	}
	return RecordSortRating
}

// SortOrder selects the display order of query results.
type SortOrder string

const (
	SortOrderDesc SortOrder = "desc"
	SortOrderAsc  SortOrder = "asc"
)

// IsValid reports whether the order is a declared sort order.
func (order SortOrder) IsValid() bool {
	switch order {
	case SortOrderDesc, SortOrderAsc:
		return true
	default:
		return false
	}
}

// IsDescending reports whether results should be displayed in descending order.
func (order SortOrder) IsDescending() bool {
	return order == SortOrderDesc
}
