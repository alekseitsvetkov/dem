CREATE TABLE players (
    id             BIGSERIAL PRIMARY KEY,
    name           TEXT NOT NULL,
    UNIQUE (name)
);
