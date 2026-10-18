package model

// RecordScope selects the record collection returned by a query.
type RecordScope string

const (
	RecordScopeB50       RecordScope = "b50"
	RecordScopeBest      RecordScope = "best"
	RecordScopeAll       RecordScope = "all"
	RecordScopeAllCharts RecordScope = "all-charts"
)

// RecordSort selects the rating or record field used for sorting.
type RecordSort string

const (
	RecordSortRating        RecordSort = "rating"
	RecordSortFittingRating RecordSort = "fitting_rating"
	RecordSortScore         RecordSort = "score"
	RecordSortRecordTime    RecordSort = "record_time"
)

// SortOrder selects the display order of query results.
type SortOrder string

const (
	SortOrderDesc SortOrder = "desc"
	SortOrderAsc  SortOrder = "asc"
)
