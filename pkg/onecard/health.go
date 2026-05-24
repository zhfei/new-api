package onecard

const (
	ChannelStatusEnabled          = 1
	ChannelStatusManuallyDisabled = 2
	ChannelStatusAutoDisabled     = 3
)

type HealthChecker interface {
	Name() string
	Evaluate(snapshot PoolHealthSnapshot) PoolHealthReport
}

type BaseHealthChecker struct {
	name string
}

func (h *BaseHealthChecker) Name() string {
	return h.name
}

func (h *BaseHealthChecker) Evaluate(snapshot PoolHealthSnapshot) PoolHealthReport {
	report := PoolHealthReport{
		Group:           snapshot.Group,
		Total:           snapshot.Total,
		Enabled:         snapshot.Enabled,
		ManualDisabled:  snapshot.ManualDisabled,
		AutoDisabled:    snapshot.AutoDisabled,
		AvgResponseTime: snapshot.AvgResponseTime,
		UsedQuota:       snapshot.UsedQuota,
		Balance:         snapshot.Balance,
		ModelCount:      len(snapshot.Models),
		Models:          append([]string(nil), snapshot.Models...),
	}
	report.Disabled = report.Total - report.Enabled
	if report.Total > 0 {
		report.AvailabilityRate = float64(report.Enabled) / float64(report.Total)
		report.FailureRate = float64(report.AutoDisabled) / float64(report.Total)
	}
	report.HealthScore = CalculateHealthScore(report.Total, report.Enabled, report.ModelCount, report.AvgResponseTime)
	return report
}

type CodexHealthChecker struct {
	BaseHealthChecker
}

type OpenAIHealthChecker struct {
	BaseHealthChecker
}

type ClaudeHealthChecker struct {
	BaseHealthChecker
}

type GeminiHealthChecker struct {
	BaseHealthChecker
}

func NewDefaultHealthChecker() HealthChecker {
	return &BaseHealthChecker{name: "default"}
}

func CalculateHealthScore(total int64, enabled int64, modelCount int, avgResponseTime float64) int {
	if total <= 0 {
		return 0
	}
	score := int(float64(enabled) / float64(total) * 70)
	if modelCount > 0 {
		score += 20
	}
	switch {
	case avgResponseTime <= 0:
		score += 10
	case avgResponseTime <= 1500:
		score += 10
	case avgResponseTime <= 3000:
		score += 6
	case avgResponseTime <= 6000:
		score += 3
	}
	if score > 100 {
		return 100
	}
	return score
}

type PoolHealthSnapshot struct {
	Group           string
	Total           int64
	Enabled         int64
	ManualDisabled  int64
	AutoDisabled    int64
	AvgResponseTime float64
	UsedQuota       int64
	Balance         float64
	Models          []string
}

type PoolHealthReport struct {
	Group            string   `json:"group"`
	Total            int64    `json:"total"`
	Enabled          int64    `json:"enabled"`
	Disabled         int64    `json:"disabled"`
	ManualDisabled   int64    `json:"manual_disabled"`
	AutoDisabled     int64    `json:"auto_disabled"`
	AvailabilityRate float64  `json:"availability_rate"`
	FailureRate      float64  `json:"failure_rate"`
	ModelCount       int      `json:"model_count"`
	Models           []string `json:"models"`
	AvgResponseTime  float64  `json:"avg_response_time"`
	UsedQuota        int64    `json:"used_quota"`
	Balance          float64  `json:"balance"`
	HealthScore      int      `json:"health_score"`
}
