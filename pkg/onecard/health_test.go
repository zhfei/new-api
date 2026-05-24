package onecard

import "testing"

func TestDefaultHealthCheckerEvaluatesPoolHealth(t *testing.T) {
	report := NewDefaultHealthChecker().Evaluate(PoolHealthSnapshot{
		Group:           GroupPlus,
		Total:           10,
		Enabled:         7,
		ManualDisabled:  1,
		AutoDisabled:    2,
		AvgResponseTime: 1200,
		UsedQuota:       123,
		Balance:         4.5,
		Models:          []string{"gpt-5"},
	})

	if report.Group != GroupPlus {
		t.Fatalf("expected group %q, got %q", GroupPlus, report.Group)
	}
	if report.Disabled != 3 {
		t.Fatalf("expected 3 disabled channels, got %d", report.Disabled)
	}
	if report.AvailabilityRate != 0.7 {
		t.Fatalf("expected availability rate 0.7, got %v", report.AvailabilityRate)
	}
	if report.FailureRate != 0.2 {
		t.Fatalf("expected failure rate 0.2, got %v", report.FailureRate)
	}
	if report.HealthScore != 79 {
		t.Fatalf("expected health score 79, got %d", report.HealthScore)
	}
}

func TestCalculateHealthScoreForEmptyPool(t *testing.T) {
	if score := CalculateHealthScore(0, 0, 0, 0); score != 0 {
		t.Fatalf("expected empty pool score 0, got %d", score)
	}
}
