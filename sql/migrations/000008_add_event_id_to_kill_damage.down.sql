ALTER TABLE kill_events DROP CONSTRAINT IF EXISTS kill_events_event_id_unique;
ALTER TABLE kill_events DROP COLUMN IF EXISTS event_id;

ALTER TABLE damage_events DROP CONSTRAINT IF EXISTS damage_events_event_id_unique;
ALTER TABLE damage_events DROP COLUMN IF EXISTS event_id;
