CREATE TABLE match_players (
    match_id       TEXT NOT NULL REFERENCES matches(match_id),
    player_id      BIGINT NOT NULL REFERENCES players(id),
    team           TEXT,
    PRIMARY KEY (match_id, player_id)
);
