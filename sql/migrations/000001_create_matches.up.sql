CREATE TABLE matches (
    match_id       TEXT PRIMARY KEY,
    team1          TEXT NOT NULL,
    team2          TEXT NOT NULL,
    map_name       TEXT,
    event_name     TEXT,
    match_date     DATE,
    tick_rate      DOUBLE PRECISION,
    duration_secs  DOUBLE PRECISION,
    demo_url       TEXT,
    minio_key      TEXT,
    parsed_at      TIMESTAMPTZ DEFAULT NOW(),
    created_at     TIMESTAMPTZ DEFAULT NOW()
);
