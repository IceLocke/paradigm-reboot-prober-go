package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRecordScopeValidation(t *testing.T) {
	for _, tc := range []struct {
		scope       RecordScope
		valid       bool
		songOrChart bool
	}{
		{RecordScopeB50, true, false},
		{RecordScopeBest, true, true},
		{RecordScopeAll, true, true},
		{RecordScopeAllCharts, true, false},
		{"", false, false},
		{"unknown", false, false},
	} {
		t.Run(string(tc.scope), func(t *testing.T) {
			assert.Equal(t, tc.valid, tc.scope.IsValid())
			assert.Equal(t, tc.songOrChart, tc.scope.IsValidForSongOrChart())
		})
	}
}

func TestRecordSortValidation(t *testing.T) {
	for _, tc := range []struct {
		sortBy RecordSort
		valid  bool
		b50    bool
		all    bool
	}{
		{RecordSortRating, true, true, true},
		{RecordSortFittingRating, true, true, false},
		{RecordSortScore, true, false, true},
		{RecordSortRecordTime, true, false, true},
		{"", false, false, false},
		{"unknown", false, false, false},
		{"rating desc; DROP TABLE play_records;--", false, false, false},
	} {
		t.Run(string(tc.sortBy), func(t *testing.T) {
			assert.Equal(t, tc.valid, tc.sortBy.IsValid())
			for _, scope := range []struct {
				value RecordScope
				valid bool
			}{
				{RecordScopeB50, tc.b50},
				{RecordScopeBest, tc.valid},
				{RecordScopeAll, tc.all},
				{RecordScopeAllCharts, false},
				{"", false},
				{"unknown", false},
			} {
				t.Run(string(scope.value), func(t *testing.T) {
					assert.Equal(t, scope.valid, tc.sortBy.IsValidForScope(scope.value))
					want := RecordSortRating
					if scope.valid {
						want = tc.sortBy
					}
					assert.Equal(t, want, tc.sortBy.NormalizeForScope(scope.value))
				})
			}
		})
	}
}

func TestSortOrderValidation(t *testing.T) {
	for _, tc := range []struct {
		order      SortOrder
		valid      bool
		descending bool
	}{
		{SortOrderDesc, true, true},
		{SortOrderAsc, true, false},
		{"", false, false},
		{"DESC", false, false},
		{"unknown", false, false},
	} {
		t.Run(string(tc.order), func(t *testing.T) {
			assert.Equal(t, tc.valid, tc.order.IsValid())
			assert.Equal(t, tc.descending, tc.order.IsDescending())
		})
	}
}
