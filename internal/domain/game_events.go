package domain

import "time"

// MatchMetadata represents a parsed demo's match-level information.
type MatchMetadata struct {
	MatchID      string    `json:"match_id"`
	Team1        string    `json:"team1"`
	Team2        string    `json:"team2"`
	MapName      string    `json:"map_name"`
	TickRate     float64   `json:"tick_rate"`
	DurationSecs float64   `json:"duration_seconds"`
	ParsedAt     time.Time `json:"parsed_at"`
}

// KillEvent represents a single kill during a match.
type KillEvent struct {
	MatchID     string `json:"match_id"`
	RoundNumber int    `json:"round_number"`
	Tick        int    `json:"tick"`
	Killer      string `json:"killer"`
	Victim      string `json:"victim"`
	Weapon      string `json:"weapon"`
	IsHeadshot  bool   `json:"is_headshot"`
	Wallbang    bool   `json:"wallbang"`
	KillerTeam  string `json:"killer_team"`
	VictimTeam  string `json:"victim_team"`
}

// RoundInfo represents a round's start and end state.
type RoundInfo struct {
	MatchID     string `json:"match_id"`
	RoundNumber int    `json:"round_number"`
	StartTick   int    `json:"start_tick"`
	EndTick     int    `json:"end_tick"`
	Winner      string `json:"winner"`
	EndReason   string `json:"end_reason"`
	TTeam       string `json:"t_team"`
	CTTeam      string `json:"ct_team"`
}

// DamageEvent represents player damage (for future spatial analysis).
type DamageEvent struct {
	MatchID     string `json:"match_id"`
	RoundNumber int    `json:"round_number"`
	Tick        int    `json:"tick"`
	Attacker    string `json:"attacker"`
	Victim      string `json:"victim"`
	Weapon      string `json:"weapon"`
	Damage      int    `json:"damage"`
	HitGroup    string `json:"hit_group"`
}
