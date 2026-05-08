CREATE TABLE request_sessions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    session_token_hash BLOB UNIQUE NOT NULL,
    ip_address TEXT,
    user_agent TEXT,
    expires_at DATETIME NOT NULL,
    last_seen_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    revoked_at DATETIME
);

CREATE INDEX idx_request_sessions_user ON request_sessions(user_id);
CREATE INDEX idx_request_sessions_expiry ON request_sessions(expires_at);
CREATE INDEX idx_request_sessions_revoked ON request_sessions(revoked_at);
