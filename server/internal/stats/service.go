package stats

// Service implements the dashboard stats aggregation.
// TODO: replace stub with real DB queries as features are implemented.
type Service struct{}

// New creates a Stats Service.
func New() *Service { return &Service{} }

// Data mirrors the shape expected by statsStore.ts.
type Data struct {
	ActiveDownloads    int            `json:"active_downloads"`
	PendingUploads     int            `json:"pending_uploads"`
	TotalClips         int            `json:"total_clips"`
	TotalShorts        int            `json:"total_shorts"`
	ChannelsCount      int            `json:"channels_count"`
	QuotaUsedToday     int            `json:"quota_used_today"`
	RecentPipelineRuns []RunSummary   `json:"recent_pipeline_runs"`
}

// RunSummary is a lightweight pipeline run entry for the dashboard.
type RunSummary struct {
	ID         int64  `json:"id"`
	Status     string `json:"status"`
	Mode       string `json:"mode"`
	StartedAt  int64  `json:"started_at"`
	FinishedAt *int64 `json:"finished_at"`
}

// Get returns aggregate dashboard stats.
func (s *Service) Get() Data {
	return Data{
		RecentPipelineRuns: []RunSummary{},
	}
}
