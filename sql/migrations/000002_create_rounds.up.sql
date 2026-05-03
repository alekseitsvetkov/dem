CREATE TABLE rounds (
    id             BIGSERIAL PRIMARY KEY,
    match_id       TEXT NOT NULL REFERENCES matches(match_id),
    round_number   INTEGER NOT NULL,
    start_tick     INTEGER NOT NULL,
    end_tick       INTEGER,
    winner         TEXT,
    end_reason     TEXT,
    t_team         TEXT,
    ct_team        TEXT,
    UNIQUE (match_id, round_number)
);

CREATE INDEX idx_rounds_match ON rounds(match_id);
