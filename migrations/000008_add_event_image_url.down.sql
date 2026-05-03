-- SQLite doesn't support DROP COLUMN before 3.35.0; recreate table if needed.
-- For simplicity, this migration is not reversible in older SQLite versions.
ALTER TABLE events DROP COLUMN image_url;
