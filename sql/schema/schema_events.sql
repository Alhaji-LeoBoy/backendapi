CREATE TABLE events (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  user_id INTEGER NOT NULL,
  title TEXT NOT NULL,
  owner_name TEXT NOT NULL,
  description TEXT NOT NULL,
  location TEXT NOT NULL,
  start_time DATETIME NOT NULL,
  image_url TEXT NOT NULL DEFAULT '',
  price REAL NOT NULL DEFAULT 0,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

-- Index for performance
CREATE INDEX idx_events_user_id ON events(user_id);

-- Auto-update timestamps
CREATE TRIGGER events_updated_at_trigger
AFTER UPDATE ON events
FOR EACH ROW
BEGIN
  UPDATE events SET updated_at = CURRENT_TIMESTAMP WHERE id = OLD.id;
END;