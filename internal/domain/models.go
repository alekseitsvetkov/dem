package domain

// Event represents an HLTV event with tier information.
type Event struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	StartDate string `json:"start_date,omitempty"`
	EndDate   string `json:"end_date,omitempty"`
	Location  string `json:"location,omitempty"`
	Tier      string `json:"tier,omitempty"`
	PrizePool int    `json:"prize_pool,omitempty"`
	SourceURL string `json:"source_url"`
}

// Result represents a completed HLTV match result.
type Result struct {
	MatchID   string `json:"match_id"`
	Team1     string `json:"team1"`
	Team2     string `json:"team2"`
	Score     string `json:"score"`
	Event     string `json:"event,omitempty"`
	Date      string `json:"date,omitempty"`
	Format    string `json:"format,omitempty"`
	SourceURL string `json:"source_url"`
}

// DemoLink represents a demo download link for an HLTV match.
type DemoLink struct {
	MatchID  string `json:"match_id"`
	MatchURL string `json:"match_url"`
	DemoURL  string `json:"demo_url,omitempty"`
}
