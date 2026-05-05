package parser

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/nats-io/nats.go/jetstream"

	dem "github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs"
	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/common"
	events "github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/events"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/alekseitsvetkov/dem/internal/domain"
	"github.com/alekseitsvetkov/dem/pkg/natsutil"
)

// ParserService implements service.Service. It consumes parse jobs from NATS,
// streams .dem.gz from MinIO directly to demoinfocs-golang, registers 12 event
// handlers, and batch-inserts structured game events into Postgres per round.
type ParserService struct {
	cfg    Config
	logger *slog.Logger

	js    jetstream.JetStream
	minio *minio.Client
	pool  *pgxpool.Pool
}

// ParserOption is a functional option for configuring a ParserService.
type ParserOption func(*ParserService)

// NewParserService creates a new ParserService with the given config and options.
// Dependencies (NATS, Minio, Postgres) are injected via functional options per CROS-02.
func NewParserService(cfg Config, opts ...ParserOption) *ParserService {
	p := &ParserService{
		cfg:    cfg,
		logger: slog.Default(),
	}
	for _, opt := range opts {
		opt(p)
	}
	return p
}

// WithNATS sets the JetStream context for the parser.
func WithNATS(js jetstream.JetStream) ParserOption {
	return func(p *ParserService) { p.js = js }
}

// WithMinio sets the MinIO client for streaming demo files.
func WithMinio(client *minio.Client) ParserOption {
	return func(p *ParserService) { p.minio = client }
}

// WithPostgres sets the pgxpool connection pool for event persistence.
func WithPostgres(pool *pgxpool.Pool) ParserOption {
	return func(p *ParserService) { p.pool = pool }
}

// WithLogger sets the structured logger.
func WithLogger(logger *slog.Logger) ParserOption {
	return func(p *ParserService) { p.logger = logger }
}

// parseJob is the NATS message payload for a parse request.
type parseJob struct {
	Bucket    string `json:"bucket"`
	ObjectKey string `json:"object_key"`
	MatchID   string `json:"match_id"`
	MatchURL  string `json:"match_url"`
	EventName string `json:"event_name"`
	Team1     string `json:"team1"`
	Team2     string `json:"team2"`
	MatchDate string `json:"match_date"`
}

// Run implements service.Service.
//
// Per PARS-01: creates or updates a durable pull consumer on DEM_PARSE
// filtering dem.parse.jobs. Per D-06: MaxAckPending is set from config
// (default 1 — single parser at a time). Pulls messages in a loop and
// processes each with conditional defer covering D-10 (Ack on success,
// NakWithDelay on failure).
func (p *ParserService) Run(ctx context.Context) error {
	// Ensure JetStream streams exist (idempotent — no-op if already created).
	if err := natsutil.CreateStreams(ctx, p.js); err != nil {
		return fmt.Errorf("create streams: %w", err)
	}

	// Create or update the durable pull consumer.
	// Per D-06: MaxAckPending set from Concurrency config (default 1).
	cons, err := p.js.CreateOrUpdateConsumer(ctx, natsutil.StreamParse, jetstream.ConsumerConfig{
		Durable:       "parse-worker",
		FilterSubject: natsutil.SubjectParse,
		AckPolicy:     jetstream.AckExplicitPolicy,
		MaxDeliver:    3,
		MaxAckPending: p.cfg.Concurrency,
		AckWait:       p.cfg.AckWait,
	})
	if err != nil {
		return fmt.Errorf("create parser consumer: %w", err)
	}

	p.logger.Info("parser consumer ready",
		slog.String("stream", natsutil.StreamParse),
		slog.String("consumer", "parse-worker"),
		slog.Int("max_ack_pending", p.cfg.Concurrency),
	)

	// Pull messages one at a time per D-06.
	iter, err := cons.Messages(jetstream.PullMaxMessages(1))
	if err != nil {
		return fmt.Errorf("parser messages: %w", err)
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		msg, err := iter.Next()
		if err != nil {
			return fmt.Errorf("parser iter next: %w", err)
		}

		go p.processMessage(ctx, msg)
	}
}

// processMessage handles a single parse job with conditional defer covering D-10.
// msgErr starts nil. Every error path sets msgErr and returns. The success path
// leaves msgErr nil. The deferred function checks msgErr once: nil = Ack, non-nil
// = NakWithDelay(1m). No path executes both Ack and Nak.
func (p *ParserService) processMessage(ctx context.Context, msg jetstream.Msg) {
	var msgErr error
	defer func() {
		if msgErr == nil {
			if ackErr := msg.Ack(); ackErr != nil {
				p.logger.Error("ack failed", slog.String("error", ackErr.Error()))
			}
		} else {
			p.logger.Error("parse failed, nacking with delay",
				slog.String("error", msgErr.Error()),
			)
			if nakErr := msg.NakWithDelay(1 * time.Minute); nakErr != nil {
				p.logger.Error("nak failed", slog.String("error", nakErr.Error()))
			}
		}
	}()

	var job parseJob
	if err := json.Unmarshal(msg.Data(), &job); err != nil {
		p.logger.Error("unmarshal parse job failed", slog.String("error", err.Error()))
		msgErr = fmt.Errorf("unmarshal parse job: %w", err)
		return
	}

	logger := p.logger.With(slog.String("match_id", job.MatchID))
	logger.Info("starting parse")

	// Per PARS-02: stream from MinIO directly to demoinfocs, never buffered.
	parseCtx, cancel := context.WithTimeout(ctx, p.cfg.ParseTimeout)
	defer cancel()

	obj, err := p.minio.GetObject(parseCtx, job.Bucket, job.ObjectKey, minio.GetObjectOptions{})
	if err != nil {
		logger.Error("minio get object failed", slog.String("error", err.Error()))
		msgErr = fmt.Errorf("minio get object %s/%s: %w", job.Bucket, job.ObjectKey, err)
		return
	}
	defer obj.Close()

	// Per PARS-02: stream directly to demoinfocs, never read into memory.
	parser := dem.NewParser(obj)
	// Per D-09: defer p.Close() is non-negotiable — prevents 250 MB C-memory leak.
	defer parser.Close()

	writer := NewEventWriter(p.pool, job.MatchID, logger)

		// Ensure the match row exists before any dependent writes.
		// MatchStart may not fire for CS2 demos; this guarantees FK constraints pass.
		if err := writer.WriteMatch(ctx, domain.MatchMetadata{
			MatchID: job.MatchID,
			Team1:   job.Team1,
			Team2:   job.Team2,
		}); err != nil {
			logger.Error("write match failed", slog.String("error", err.Error()))
			msgErr = fmt.Errorf("write match: %w", err)
			return
		}

	// Per D-08: register 12 event handlers wired to EventWriter.
	p.registerHandlers(parser, writer, &job, logger)

	// Parse the demo to completion.
	if err := parser.ParseToEnd(); err != nil {
		logger.Error("parse to end failed", slog.String("error", err.Error()))
		msgErr = fmt.Errorf("parse to end: %w", err)
		return
	}

	// Flush any remaining events from the final round.
	if err := writer.Flush(context.Background()); err != nil {
		logger.Error("final flush failed", slog.String("error", err.Error()))
		msgErr = fmt.Errorf("final flush: %w", err)
		return
	}

		// Delete demo from MinIO — re-download if schema expands later.
		if err := p.minio.RemoveObject(ctx, job.Bucket, job.ObjectKey, minio.RemoveObjectOptions{}); err != nil {
			logger.Warn("failed to delete demo from minio", slog.String("error", err.Error()))
		}

	logger.Info("parse complete")
	// Defer runs: msgErr is nil → msg.Ack()
}

// registerHandlers registers the 12 required event handlers per D-08.
// Each handler collects events into per-round buffers on the EventWriter
// and flushes on RoundEnd.
func (p *ParserService) registerHandlers(
	parser dem.Parser,
	writer *EventWriter,
	job *parseJob,
	logger *slog.Logger,
) {
	var currentRound int

	// 1. MatchStart
	parser.RegisterEventHandler(func(e events.MatchStart) {
		meta := domain.MatchMetadata{
			MatchID:      job.MatchID,
			Team1:        job.Team1,
			Team2:        job.Team2,
			MapName:      "", // demoinfocs-golang v5 does not expose MapName through the Parser interface
			TickRate:     parser.TickRate(),
			DurationSecs: parser.CurrentTime().Seconds(),
		}
		if err := writer.WriteMatch(context.Background(), meta); err != nil {
			logger.Error("write match failed", slog.String("error", err.Error()))
		}
	})

	// 2. RoundStart
	parser.RegisterEventHandler(func(e events.RoundStart) {
		roundNum := parser.GameState().TotalRoundsPlayed() + 1
		currentRound = roundNum
		writer.SetRound(domain.RoundInfo{
			MatchID:     job.MatchID,
			RoundNumber: roundNum,
			StartTick:   parser.GameState().IngameTick(),
		})
	})

	// 3. RoundEnd — flush buffered events for this round.
	parser.RegisterEventHandler(func(e events.RoundEnd) {
		// Update round info with winner and end reason.
		endTick := parser.GameState().IngameTick()
		winner := teamToString(e.Winner)
		writer.SetRound(domain.RoundInfo{
			MatchID:     job.MatchID,
			RoundNumber: currentRound,
			StartTick:   0, // already set on RoundStart; Flush will use the latest SetRound
			EndTick:     endTick,
			Winner:      winner,
			EndReason:   roundEndReasonToString(e.Reason),
		})
		// Per D-07: flush all events for this round in a batch.
		if err := writer.Flush(context.Background()); err != nil {
			logger.Error("round flush failed",
				slog.String("match_id", job.MatchID),
				slog.Int("round", currentRound),
				slog.String("error", err.Error()),
			)
		}
	})

	// 4. Kill
	parser.RegisterEventHandler(func(e events.Kill) {
		if e.Killer == nil || e.Victim == nil {
			return
		}
		weapon := ""
		if e.Weapon != nil {
			weapon = e.Weapon.String()
		}
		writer.AddKill(domain.KillEvent{
			MatchID:     job.MatchID,
			RoundNumber: currentRound,
			Tick:        parser.GameState().IngameTick(),
			Killer:      e.Killer.Name,
			Victim:      e.Victim.Name,
			Weapon:      weapon,
			IsHeadshot:  e.IsHeadshot,
			Wallbang:    e.PenetratedObjects > 0,
			KillerTeam:  teamToString(e.Killer.Team),
			VictimTeam:  teamToString(e.Victim.Team),
		})
	})

	// 5. PlayerHurt (DamageEvent)
	parser.RegisterEventHandler(func(e events.PlayerHurt) {
		if e.Attacker == nil || e.Player == nil {
			return
		}
		weapon := ""
		if e.Weapon != nil {
			weapon = e.Weapon.String()
		}
		writer.AddDamage(domain.DamageEvent{
			MatchID:     job.MatchID,
			RoundNumber: currentRound,
			Tick:        parser.GameState().IngameTick(),
			Attacker:    e.Attacker.Name,
			Victim:      e.Player.Name,
			Weapon:      weapon,
			Damage:      e.HealthDamage,
			HitGroup:    hitGroupToString(e.HitGroup),
		})
	})

	// 6. WeaponFire — one of 12 required handlers (D-08). Log at debug level.
	parser.RegisterEventHandler(func(e events.WeaponFire) {
		playerName := "unknown"
		if e.Shooter != nil {
			playerName = e.Shooter.Name
		}
		weaponName := "unknown"
		if e.Weapon != nil {
			weaponName = e.Weapon.String()
		}
		logger.Debug("weapon fire",
			slog.String("player", playerName),
			slog.String("weapon", weaponName),
			slog.Int("tick", parser.GameState().IngameTick()),
		)
	})

	// 7. BombPlant (we also handle BombPlanted to get player and site).
	// demoinfocs fires BombPlanted (with Player + Site), not BombPlant directly.
	// BombPlantBegin fires at the start of planting (no site).
	parser.RegisterEventHandler(func(e events.BombPlanted) {
		playerName := "unknown"
		if e.Player != nil {
			playerName = e.Player.Name
		}
		logger.Debug("bomb planted",
			slog.String("planter", playerName),
			slog.Int("tick", parser.GameState().IngameTick()),
		)
	})

	// 8. BombDefuse (BombDefused event with Player + Site)
	parser.RegisterEventHandler(func(e events.BombDefused) {
		playerName := "unknown"
		if e.Player != nil {
			playerName = e.Player.Name
		}
		logger.Debug("bomb defused",
			slog.String("defuser", playerName),
			slog.Int("tick", parser.GameState().IngameTick()),
		)
	})

	// 9. BombExplode
	parser.RegisterEventHandler(func(e events.BombExplode) {
		logger.Debug("bomb exploded",
			slog.Int("tick", parser.GameState().IngameTick()),
		)
	})

	// 10. GrenadeProjectileThrow
	parser.RegisterEventHandler(func(e events.GrenadeProjectileThrow) {
		playerName := "unknown"
		if e.Projectile != nil && e.Projectile.Thrower != nil {
			playerName = e.Projectile.Thrower.Name
		}
		grenadeType := "unknown"
		if e.Projectile != nil && e.Projectile.WeaponInstance != nil {
			grenadeType = e.Projectile.WeaponInstance.String()
		}
		logger.Debug("grenade thrown",
			slog.String("player", playerName),
			slog.String("type", grenadeType),
			slog.Int("tick", parser.GameState().IngameTick()),
		)
	})

	// 11. PlayerConnect
	parser.RegisterEventHandler(func(e events.PlayerConnect) {
		if e.Player == nil {
			return
		}
		team := teamToString(e.Player.Team)
		if err := writer.UpsertPlayer(context.Background(), e.Player.Name, team); err != nil {
			logger.Error("upsert player failed",
				slog.String("player", e.Player.Name),
				slog.String("error", err.Error()),
			)
		}
	})

	// 12. TeamSideSwitch
	parser.RegisterEventHandler(func(e events.TeamSideSwitch) {
		logger.Debug("team side switch",
			slog.Int("tick", parser.GameState().IngameTick()),
		)
	})
}

// teamToString converts a demoinfocs common.Team to its string representation.
func teamToString(team common.Team) string {
	switch team {
	case common.TeamCounterTerrorists:
		return "CT"
	case common.TeamTerrorists:
		return "T"
	default:
		return "UNKNOWN"
	}
}

// hitGroupToString converts a demoinfocs HitGroup to its string representation.
// The events.HitGroup type is a byte without a String() method.
func hitGroupToString(hg events.HitGroup) string {
	switch hg {
	case events.HitGroupGeneric:
		return "generic"
	case events.HitGroupHead:
		return "head"
	case events.HitGroupChest:
		return "chest"
	case events.HitGroupStomach:
		return "stomach"
	case events.HitGroupLeftArm:
		return "left_arm"
	case events.HitGroupRightArm:
		return "right_arm"
	case events.HitGroupLeftLeg:
		return "left_leg"
	case events.HitGroupRightLeg:
		return "right_leg"
	case events.HitGroupNeck:
		return "neck"
	case events.HitGroupGear:
		return "gear"
	default:
		return "unknown"
	}
}

// roundEndReasonToString converts a demoinfocs RoundEndReason to its string representation.
// The events.RoundEndReason type is a byte without a String() method.
func roundEndReasonToString(reason events.RoundEndReason) string {
	switch reason {
	case events.RoundEndReasonTargetBombed:
		return "target_bombed"
	case events.RoundEndReasonBombDefused:
		return "bomb_defused"
	case events.RoundEndReasonCTWin:
		return "ct_win"
	case events.RoundEndReasonTerroristsWin:
		return "t_win"
	case events.RoundEndReasonCTSurrender:
		return "ct_surrender"
	case events.RoundEndReasonTerroristsSurrender:
		return "t_surrender"
	case events.RoundEndReasonDraw:
		return "draw"
	case events.RoundEndReasonHostagesRescued:
		return "hostages_rescued"
	case events.RoundEndReasonVIPEscaped:
		return "vip_escaped"
	case events.RoundEndReasonVIPKilled:
		return "vip_killed"
	case events.RoundEndReasonTerroristsEscaped:
		return "terrorists_escaped"
	case events.RoundEndReasonCTStoppedEscape:
		return "ct_stopped_escape"
	case events.RoundEndReasonTerroristsStopped:
		return "terrorists_stopped"
	case events.RoundEndReasonTargetSaved:
		return "target_saved"
	case events.RoundEndReasonHostagesNotRescued:
		return "hostages_not_rescued"
	case events.RoundEndReasonTerroristsNotEscaped:
		return "terrorists_not_escaped"
	case events.RoundEndReasonVIPNotEscaped:
		return "vip_not_escaped"
	case events.RoundEndReasonGameStart:
		return "game_start"
	case events.RoundEndReasonTerroristsPlanted:
		return "terrorists_planted"
	case events.RoundEndReasonCTsReachedHostage:
		return "cts_reached_hostage"
	default:
		return fmt.Sprintf("unknown_%d", reason)
	}
}
