CREATE TABLE processed_matches (
    match_id    BIGINT PRIMARY KEY,
    processed_at TIMESTAMPTZ DEFAULT NOW()
);
