CREATE TABLE kill_events (
    id             BIGSERIAL PRIMARY KEY,
    match_id       TEXT NOT NULL REFERENCES matches(match_id),
    round_number   INTEGER NOT NULL,
    tick           INTEGER NOT NULL,
    killer         TEXT NOT NULL,
    victim         TEXT NOT NULL,
    weapon         TEXT,
    is_headshot    BOOLEAN DEFAULT FALSE,
    wallbang       BOOLEAN DEFAULT FALSE,
    killer_team    TEXT,
    victim_team    TEXT
);

CREATE INDEX idx_kill_events_match_round ON kill_events(match_id, round_number);
