package parser

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/alekseitsvetkov/dem/internal/domain"
)

// EventWriter manages per-round event buffering and batch Postgres inserts.
// Per D-07: events are accumulated per round and flushed as a batch on RoundEnd.
// Per PARS-04: all write paths use ON CONFLICT with deterministic event IDs for idempotency.
type EventWriter struct {
	pool   *pgxpool.Pool
	logger *slog.Logger

	matchID    string
	matchMeta  *domain.MatchMetadata
	mu         sync.Mutex
	kills      []domain.KillEvent
	damages    []domain.DamageEvent
	round      *domain.RoundInfo

	// In-memory dedup set for player upsert (avoids DB round-trip per connect event).
	playerNames map[string]bool
}

// NewEventWriter creates a new EventWriter for the given match.
func NewEventWriter(pool *pgxpool.Pool, matchID string, logger *slog.Logger) *EventWriter {
	return &EventWriter{
		pool:        pool,
		logger:      logger,
		matchID:     matchID,
		playerNames: make(map[string]bool),
	}
}

// SetRound sets the current round info. Called on RoundStart.
func (w *EventWriter) SetRound(r domain.RoundInfo) {
	w.mu.Lock()
	w.round = &r
	w.mu.Unlock()
}

// AddKill appends a kill event to the current round buffer.
func (w *EventWriter) AddKill(e domain.KillEvent) {
	w.mu.Lock()
	w.kills = append(w.kills, e)
	w.mu.Unlock()
}

// AddDamage appends a damage event to the current round buffer.
func (w *EventWriter) AddDamage(e domain.DamageEvent) {
	w.mu.Lock()
	w.damages = append(w.damages, e)
	w.mu.Unlock()
}

// WriteMatch upserts the match row. Called on MatchStart.
// Per PARS-04: ON CONFLICT (match_id) DO UPDATE to refresh parsed_at and duration on re-parse.
func (w *EventWriter) WriteMatch(ctx context.Context, meta domain.MatchMetadata) error {
	w.mu.Lock()
	w.matchMeta = &meta
	w.mu.Unlock()

	tag, err := w.pool.Exec(ctx, `
		INSERT INTO matches (match_id, team1, team2, map_name, tick_rate, duration_secs, parsed_at)
		VALUES ($1, $2, $3, $4, $5, $6, NOW())
		ON CONFLICT (match_id) DO UPDATE SET
			tick_rate = EXCLUDED.tick_rate,
			duration_secs = EXCLUDED.duration_secs,
			parsed_at = NOW()
	`, meta.MatchID, meta.Team1, meta.Team2, meta.MapName, meta.TickRate, meta.DurationSecs)
	if err != nil {
		return fmt.Errorf("write match %s: %w", meta.MatchID, err)
	}
	w.logger.Info("match row written", slog.String("match_id", meta.MatchID), slog.Int64("rows_affected", tag.RowsAffected()))
	return nil
}

// UpsertPlayer inserts a player name if not already seen this match, and inserts into
// match_players junction. Called on PlayerConnect.
// Per PARS-04: ON CONFLICT (name) DO UPDATE ... RETURNING id for players;
// ON CONFLICT (match_id, player_id) DO NOTHING for match_players.
func (w *EventWriter) UpsertPlayer(ctx context.Context, playerName, team string) error {
	// In-memory dedup to avoid unnecessary DB round-trips.
	w.mu.Lock()
	if w.playerNames[playerName] {
		w.mu.Unlock()
		return nil
	}
	w.playerNames[playerName] = true
	w.mu.Unlock()

	// Insert or get existing player ID.
	var playerID int64
	err := w.pool.QueryRow(ctx, `
		INSERT INTO players (name) VALUES ($1)
		ON CONFLICT (name) DO UPDATE SET name = EXCLUDED.name
		RETURNING id
	`, playerName).Scan(&playerID)
	if err != nil {
		return fmt.Errorf("upsert player %s: %w", playerName, err)
	}

	// Insert match_players junction.
	_, err = w.pool.Exec(ctx, `
		INSERT INTO match_players (match_id, player_id, team) VALUES ($1, $2, $3)
		ON CONFLICT (match_id, player_id) DO NOTHING
	`, w.matchID, playerID, team)
	if err != nil {
		return fmt.Errorf("insert match_player %s: %w", playerName, err)
	}
	return nil
}

// Flush writes all buffered events for the current round to Postgres in a single batch.
// Called on RoundEnd. Uses pgx.Batch for atomicity within the round.
// After flush, clears kill/damage/round buffers for the next round.
//
// Per D-07: deterministic event IDs {match_id}-{round_number}-kill-{seq} and
// {match_id}-{round_number}-dmg-{seq} persist the event_id column added by migration
// 000008. Per PARS-04: ON CONFLICT (event_id) DO NOTHING on kill_events and
// damage_events; ON CONFLICT (match_id, round_number) DO NOTHING on rounds.
func (w *EventWriter) Flush(ctx context.Context) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	batch := &pgx.Batch{}

	// Insert round info with ON CONFLICT (match_id, round_number) DO NOTHING.
	// The UNIQUE(match_id, round_number) constraint handles idempotency without
	// needing a deterministic event_id column.
	if w.round != nil {
		round := *w.round
		batch.Queue(`
			INSERT INTO rounds (match_id, round_number, start_tick, end_tick, winner, end_reason, t_team, ct_team)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
			ON CONFLICT (match_id, round_number) DO NOTHING
		`, round.MatchID, round.RoundNumber, round.StartTick, round.EndTick,
			round.Winner, round.EndReason, round.TTeam, round.CTTeam)
	}

	// Kill events with deterministic event_id: {match_id}-{round_number}-kill-{sequence}.
	// event_id has a UNIQUE constraint (migration 000008) — ON CONFLICT (event_id) DO NOTHING
	// ensures re-parsing the same demo produces no duplicate data.
	for seq, k := range w.kills {
		eventID := fmt.Sprintf("%s-%d-kill-%d", k.MatchID, k.RoundNumber, seq)
		batch.Queue(`
			INSERT INTO kill_events (event_id, match_id, round_number, tick, killer, victim, weapon, is_headshot, wallbang, killer_team, victim_team)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
			ON CONFLICT (event_id) DO NOTHING
		`, eventID, k.MatchID, k.RoundNumber, k.Tick, k.Killer, k.Victim, k.Weapon,
			k.IsHeadshot, k.Wallbang, k.KillerTeam, k.VictimTeam)
	}

	// Damage events with deterministic event_id: {match_id}-{round_number}-dmg-{sequence}.
	for seq, d := range w.damages {
		eventID := fmt.Sprintf("%s-%d-dmg-%d", d.MatchID, d.RoundNumber, seq)
		batch.Queue(`
			INSERT INTO damage_events (event_id, match_id, round_number, tick, attacker, victim, weapon, damage, hit_group)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
			ON CONFLICT (event_id) DO NOTHING
		`, eventID, d.MatchID, d.RoundNumber, d.Tick, d.Attacker, d.Victim, d.Weapon,
			d.Damage, d.HitGroup)
	}

	// Execute the batch if there are statements queued.
	if batch.Len() > 0 {
		br := w.pool.SendBatch(ctx, batch)
		defer br.Close()

		for i := 0; i < batch.Len(); i++ {
			_, err := br.Exec()
			if err != nil {
				w.logger.Error("batch exec failed",
					slog.Int("queue_index", i),
					slog.String("error", err.Error()),
				)
				// Continue — partial round data is acceptable; re-parse will fill
				// gaps via ON CONFLICT DO NOTHING.
			}
		}
	}

	// Clear buffers for next round.
	w.kills = w.kills[:0]
	w.damages = w.damages[:0]
	w.round = nil

	return nil
}
