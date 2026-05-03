CREATE TABLE damage_events (
    id             BIGSERIAL PRIMARY KEY,
    match_id       TEXT NOT NULL REFERENCES matches(match_id),
    round_number   INTEGER NOT NULL,
    tick           INTEGER NOT NULL,
    attacker       TEXT NOT NULL,
    victim         TEXT NOT NULL,
    weapon         TEXT,
    damage         INTEGER NOT NULL,
    hit_group      TEXT
);

CREATE INDEX idx_damage_events_match_round ON damage_events(match_id, round_number);
