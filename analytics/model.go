package analytics

type SalesStats struct {
	TotalRevenue  float64 `json:"totalRevenue"`
	OrderCount    int     `json:"orderCount"`
	AvgOrderValue float64 `json:"avgOrderValue"`
}

type SystemStats struct {
	TotalWorkflows     int     `json:"totalWorkflows"`
	SuccessRate        float64 `json:"successRate"`
	FailureRate        float64 `json:"failureRate"`
	AvgExecutionTimeMs float64 `json:"avgExecutionTimeMs"`
}
