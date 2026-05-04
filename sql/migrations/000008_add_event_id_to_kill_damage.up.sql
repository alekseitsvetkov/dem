ALTER TABLE kill_events ADD COLUMN event_id TEXT NOT NULL DEFAULT '';
ALTER TABLE kill_events ADD CONSTRAINT kill_events_event_id_unique UNIQUE (event_id);

ALTER TABLE damage_events ADD COLUMN event_id TEXT NOT NULL DEFAULT '';
ALTER TABLE damage_events ADD CONSTRAINT damage_events_event_id_unique UNIQUE (event_id);
