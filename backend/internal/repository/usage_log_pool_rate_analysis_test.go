package repository

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Wei-Shaw/sub2api/internal/pkg/usagestats"
	"github.com/stretchr/testify/require"
)

func TestAccountPoolRateAnalysisOrderBy(t *testing.T) {
	require.Equal(t,
		"inferred_multiplier ASC NULLS LAST, group_name ASC, group_id ASC",
		accountPoolRateAnalysisOrderBy("inferred_multiplier", "asc"),
	)
	require.Equal(t,
		"inferred_multiplier DESC NULLS LAST, group_name ASC, group_id ASC",
		accountPoolRateAnalysisOrderBy("unknown", "desc"),
	)
	require.Equal(t,
		"account_cost DESC, group_name ASC, group_id ASC",
		accountPoolRateAnalysisOrderBy("account_cost", "desc"),
	)
	require.Equal(t,
		"coverage_rate ASC, group_name ASC, group_id ASC",
		accountPoolRateAnalysisOrderBy("coverage_rate", "invalid"),
	)
}

func TestUsageLogRepositoryGetAccountPoolRateAnalysis(t *testing.T) {
	db, mock := newSQLMock(t)
	repo := &usageLogRepository{sql: db}

	start := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 0, 7)

	rows := sqlmock.NewRows([]string{
		"group_id",
		"group_name",
		"platform",
		"configured_rate_multiplier",
		"requests",
		"valid_requests",
		"uncovered_requests",
		"input_tokens",
		"output_tokens",
		"valid_input_tokens",
		"valid_output_tokens",
		"theoretical_cost",
		"account_cost",
		"inferred_multiplier",
		"coverage_rate",
	}).
		AddRow(int64(1), "Pool A", "openai", 1.2, int64(3), int64(2), int64(1), int64(300), int64(600), int64(250), int64(500), 10.0, 1.8, 0.18, 0.6666666667).
		AddRow(int64(2), "Pool B", "anthropic", 1.0, int64(1), int64(0), int64(1), int64(100), int64(200), int64(0), int64(0), 0.0, 0.0, nil, 0.0)

	mock.ExpectQuery("account_stats_cost \\* COALESCE\\(ul\\.account_rate_multiplier, 1\\)").
		WithArgs(start, end).
		WillReturnRows(rows)

	got, err := repo.GetAccountPoolRateAnalysis(context.Background(), usagestats.AccountPoolRateAnalysisQuery{
		StartTime: start,
		EndTime:   end,
		SortBy:    "inferred_multiplier",
		SortOrder: "asc",
	})

	require.NoError(t, err)
	require.Len(t, got.Items, 2)
	require.Equal(t, int64(2), got.Summary.Groups)
	require.Equal(t, int64(1), got.Summary.ValidGroups)
	require.Equal(t, int64(4), got.Summary.Requests)
	require.Equal(t, int64(2), got.Summary.ValidRequests)
	require.Equal(t, int64(2), got.Summary.UncoveredRequests)
	require.Equal(t, 10.0, got.Summary.TheoreticalCost)
	require.Equal(t, 1.8, got.Summary.AccountCost)

	require.NotNil(t, got.Items[0].InferredMultiplier)
	require.InDelta(t, 0.18, *got.Items[0].InferredMultiplier, 0.0001)
	require.Nil(t, got.Items[1].InferredMultiplier)
	require.NoError(t, mock.ExpectationsWereMet())
}
